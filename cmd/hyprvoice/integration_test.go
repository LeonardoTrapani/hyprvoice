//go:build integration

package main

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/leonardotrapani/hyprvoice/internal/config"
	"github.com/leonardotrapani/hyprvoice/internal/llm"
	"github.com/leonardotrapani/hyprvoice/internal/models/whisper"
	"github.com/leonardotrapani/hyprvoice/internal/provider"
	"github.com/leonardotrapani/hyprvoice/internal/recording"
	"github.com/leonardotrapani/hyprvoice/internal/transcriber"
)

const (
	testSampleRate    = 16000
	testChannels      = 1
	testBitsPerSample = 16
	testTimeout       = 45 * time.Second
)

var testKeywords = []string{"Hyprvoice", "transcription", "dictation"}

func TestTranscriptionModels(t *testing.T) {
	audio, err := loadTestAudio(t)
	if err != nil {
		t.Fatalf("failed to load test audio: %v", err)
	}

	cfg := loadTestConfig(t)

	providerNames := provider.ListProvidersWithTranscription()
	sort.Strings(providerNames)

	smallestLocalModel := selectSmallestLocalModel()

	for _, providerName := range providerNames {
		p := provider.GetProvider(providerName)
		if p == nil {
			continue
		}

		models := provider.ModelsOfType(p, provider.Transcription)
		sort.Slice(models, func(i, j int) bool {
			return models[i].ID < models[j].ID
		})

		for _, model := range models {
			if model.Local && providerName == provider.ProviderWhisperCpp && model.ID != smallestLocalModel {
				continue
			}

			modes := getModesForModel(model)
			languages := getLanguagesForModel(model)
			keywordOptions := []bool{true, false}

			for _, mode := range modes {
				for _, lang := range languages {
					for _, useKeywords := range keywordOptions {
						testName := fmt.Sprintf("%s/%s/%s/lang=%s/keywords=%v",
							providerName, model.ID, mode, langDisplay(lang), useKeywords)

						model := model
						mode := mode
						lang := lang
						useKeywords := useKeywords
						providerName := providerName

						t.Run(testName, func(t *testing.T) {
							t.Parallel()
							runTranscriptionTest(t, cfg, providerName, model, mode, lang, useKeywords, audio)
						})
					}
				}
			}
		}
	}
}

func TestLLMModels(t *testing.T) {
	cfg := loadTestConfig(t)

	providerNames := provider.ListProvidersWithLLM()
	sort.Strings(providerNames)

	for _, providerName := range providerNames {
		p := provider.GetProvider(providerName)
		if p == nil {
			continue
		}

		models := provider.ModelsOfType(p, provider.LLM)
		sort.Slice(models, func(i, j int) bool {
			return models[i].ID < models[j].ID
		})

		for _, model := range models {
			for _, useKeywords := range []bool{true, false} {
				testName := fmt.Sprintf("%s/%s/keywords=%v", providerName, model.ID, useKeywords)

				model := model
				useKeywords := useKeywords
				providerName := providerName

				t.Run(testName, func(t *testing.T) {
					t.Parallel()
					runLLMTest(t, cfg, providerName, model, useKeywords)
				})
			}
		}
	}
}

func runTranscriptionTest(t *testing.T, cfg *config.Config, providerName string, model provider.Model, mode, lang string, useKeywords bool, audio []byte) {
	if model.Local {
		if _, err := exec.LookPath("whisper-cli"); err != nil {
			t.Skip("whisper-cli not found")
		}
		if !whisper.IsInstalled(model.ID) {
			t.Skipf("local model %s not installed", model.ID)
		}
	}

	apiKey := resolveTestAPIKey(cfg, providerName)
	if testProviderRequiresKey(providerName) && apiKey == "" {
		t.Skipf("missing api key for %s", providerName)
	}

	var keywords []string
	if useKeywords {
		keywords = testKeywords
	}

	streaming := mode == "streaming"
	transcribeCfg := transcriber.Config{
		Provider:  providerName,
		APIKey:    apiKey,
		Language:  lang,
		Model:     model.ID,
		Keywords:  keywords,
		Threads:   0,
		Streaming: streaming,
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	text, err := runTestTranscriber(ctx, transcribeCfg, audio)
	if err != nil {
		t.Errorf("transcription failed: %v", err)
		return
	}

	text = strings.TrimSpace(text)
	if text == "" {
		t.Error("transcription returned empty text")
		return
	}

	t.Logf("output (%d chars): %q", len(text), truncateTestString(text, 100))
}

func runLLMTest(t *testing.T, cfg *config.Config, providerName string, model provider.Model, useKeywords bool) {
	apiKey := resolveTestAPIKey(cfg, providerName)
	if testProviderRequiresKey(providerName) && apiKey == "" {
		t.Skipf("missing api key for %s", providerName)
	}

	var keywords []string
	if useKeywords {
		keywords = testKeywords
	}

	llmCfg := llm.Config{
		Provider:          providerName,
		APIKey:            apiKey,
		Model:             model.ID,
		RemoveStutters:    true,
		AddPunctuation:    true,
		FixGrammar:        true,
		RemoveFillerWords: true,
		CustomPrompt:      "",
		Keywords:          keywords,
	}

	adapter, err := llm.NewAdapter(llmCfg)
	if err != nil {
		t.Errorf("failed to create adapter: %v", err)
		return
	}

	input := "uh i i i want to test hyprvoice you know this is just a cleanup check"
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	output, err := adapter.Process(ctx, input)
	if err != nil {
		t.Errorf("llm processing failed: %v", err)
		return
	}

	output = strings.TrimSpace(output)
	if output == "" {
		t.Error("llm returned empty output")
		return
	}

	t.Logf("output (%d chars): %q", len(output), truncateTestString(output, 100))
}

func runTestTranscriber(ctx context.Context, cfg transcriber.Config, audio []byte) (string, error) {
	tr, err := transcriber.NewTranscriber(cfg)
	if err != nil {
		return "", err
	}

	frameCh := make(chan recording.AudioFrame, 8)
	errCh, err := tr.Start(ctx, frameCh)
	if err != nil {
		return "", err
	}

	sendErr := sendTestAudioFrames(ctx, frameCh, audio)
	close(frameCh)

	stopErr := tr.Stop(ctx)
	errChErr := readTestErrorChannel(errCh)

	if sendErr != nil {
		return "", sendErr
	}
	if stopErr != nil {
		return "", stopErr
	}
	if errChErr != nil {
		return "", errChErr
	}

	return tr.GetFinalTranscription()
}

func sendTestAudioFrames(ctx context.Context, frameCh chan<- recording.AudioFrame, audio []byte) error {
	const chunkBytes = 3200
	bytesPerSecond := testSampleRate * (testBitsPerSample / 8) * testChannels
	chunkDuration := time.Duration(float64(chunkBytes) / float64(bytesPerSecond) * float64(time.Second))

	for offset := 0; offset < len(audio); offset += chunkBytes {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		end := offset + chunkBytes
		if end > len(audio) {
			end = len(audio)
		}

		frame := recording.AudioFrame{Data: audio[offset:end], Timestamp: time.Now()}
		select {
		case frameCh <- frame:
		case <-ctx.Done():
			return ctx.Err()
		}

		time.Sleep(chunkDuration)
	}

	return nil
}

func readTestErrorChannel(errCh <-chan error) error {
	if errCh == nil {
		return nil
	}

	var firstErr error
	idleTimer := time.NewTimer(150 * time.Millisecond)
	defer idleTimer.Stop()

	for {
		select {
		case err, ok := <-errCh:
			if !ok {
				return firstErr
			}
			if err != nil && firstErr == nil {
				firstErr = err
			}
			if !idleTimer.Stop() {
				<-idleTimer.C
			}
			idleTimer.Reset(150 * time.Millisecond)
		case <-idleTimer.C:
			return firstErr
		}
	}
}

func loadTestAudio(t *testing.T) ([]byte, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("could not determine current file path")
	}

	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(currentFile)))
	samplePath := filepath.Join(projectRoot, "testdata", "sample.wav")

	data, err := os.ReadFile(samplePath)
	if err != nil {
		return nil, fmt.Errorf("could not read sample audio: %w", err)
	}

	return parseTestWAV(data)
}

func parseTestWAV(data []byte) ([]byte, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("invalid wav: too short")
	}
	if string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return nil, fmt.Errorf("invalid wav: missing riff/wave header")
	}

	offset := 12
	var fmtFound, dataFound bool
	var sampleRate, channels, bitsPerSample int
	var audioData []byte

	for offset+8 <= len(data) {
		chunkID := string(data[offset : offset+4])
		chunkSize := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		offset += 8
		if offset+chunkSize > len(data) {
			return nil, fmt.Errorf("invalid wav: chunk overflows file")
		}

		switch chunkID {
		case "fmt ":
			if chunkSize < 16 {
				return nil, fmt.Errorf("invalid wav: fmt chunk too short")
			}
			audioFormat := binary.LittleEndian.Uint16(data[offset : offset+2])
			if audioFormat != 1 {
				return nil, fmt.Errorf("unsupported wav format: %d", audioFormat)
			}
			channels = int(binary.LittleEndian.Uint16(data[offset+2 : offset+4]))
			sampleRate = int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
			bitsPerSample = int(binary.LittleEndian.Uint16(data[offset+14 : offset+16]))
			fmtFound = true
		case "data":
			audioData = data[offset : offset+chunkSize]
			dataFound = true
		}

		offset += chunkSize
		if chunkSize%2 == 1 {
			offset++
		}
	}

	if !fmtFound || !dataFound {
		return nil, fmt.Errorf("invalid wav: missing fmt or data chunk")
	}
	if bitsPerSample != testBitsPerSample {
		return nil, fmt.Errorf("unsupported wav bits per sample: %d", bitsPerSample)
	}

	monoData, err := downmixTestToMono(audioData, channels)
	if err != nil {
		return nil, err
	}
	resampled := resampleTestPCM16(monoData, sampleRate, testSampleRate)
	if len(resampled) == 0 {
		return nil, fmt.Errorf("invalid wav: empty audio data")
	}

	return resampled, nil
}

func downmixTestToMono(data []byte, channels int) ([]byte, error) {
	if channels == 1 {
		return data, nil
	}
	if channels <= 0 {
		return nil, fmt.Errorf("invalid channel count: %d", channels)
	}
	frameSize := 2 * channels
	if len(data)%frameSize != 0 {
		return nil, fmt.Errorf("invalid pcm data length")
	}

	frames := len(data) / frameSize
	out := make([]byte, frames*2)
	for i := 0; i < frames; i++ {
		var sum int32
		for c := 0; c < channels; c++ {
			idx := (i*channels + c) * 2
			sample := int16(binary.LittleEndian.Uint16(data[idx : idx+2]))
			sum += int32(sample)
		}
		mono := int16(sum / int32(channels))
		out[i*2] = byte(mono)
		out[i*2+1] = byte(mono >> 8)
	}

	return out, nil
}

func resampleTestPCM16(data []byte, inRate, outRate int) []byte {
	if inRate <= 0 || outRate <= 0 || inRate == outRate {
		return data
	}
	if len(data) < 2 {
		return data
	}

	numInSamples := len(data) / 2
	numOutSamples := int(math.Round(float64(numInSamples) * float64(outRate) / float64(inRate)))
	if numOutSamples <= 0 {
		return nil
	}

	out := make([]byte, numOutSamples*2)
	for i := 0; i < numOutSamples; i++ {
		srcPos := float64(i) * float64(inRate) / float64(outRate)
		srcIdx := int(srcPos)
		frac := srcPos - float64(srcIdx)

		sample1 := sampleTestAtPCM16(data, srcIdx)
		sample2 := sampleTestAtPCM16(data, srcIdx+1)
		outSample := int16(float64(sample1)*(1-frac) + float64(sample2)*frac)

		out[i*2] = byte(outSample)
		out[i*2+1] = byte(outSample >> 8)
	}

	return out
}

func sampleTestAtPCM16(data []byte, idx int) int16 {
	if idx <= 0 {
		return int16(binary.LittleEndian.Uint16(data[0:2]))
	}
	pos := idx * 2
	if pos+1 >= len(data) {
		last := len(data) - 2
		if last < 0 {
			return 0
		}
		return int16(binary.LittleEndian.Uint16(data[last : last+2]))
	}
	return int16(binary.LittleEndian.Uint16(data[pos : pos+2]))
}

func loadTestConfig(t *testing.T) *config.Config {
	cfg, err := config.Load()
	if err != nil {
		if errors.Is(err, config.ErrConfigNotFound) {
			return config.DefaultConfig()
		}
		t.Logf("warning: could not load config: %v", err)
		return config.DefaultConfig()
	}
	if cfg.Providers == nil {
		cfg.Providers = make(map[string]config.ProviderConfig)
	}
	return cfg
}

func resolveTestAPIKey(cfg *config.Config, providerName string) string {
	base := provider.BaseProviderName(providerName)
	if cfg != nil && cfg.Providers != nil {
		if pc, ok := cfg.Providers[base]; ok && pc.APIKey != "" {
			return pc.APIKey
		}
	}
	if envVar := provider.EnvVarForProvider(providerName); envVar != "" {
		return os.Getenv(envVar)
	}
	return ""
}

func testProviderRequiresKey(providerName string) bool {
	p := provider.GetProvider(provider.BaseProviderName(providerName))
	if p == nil {
		return false
	}
	return p.RequiresAPIKey()
}

func selectSmallestLocalModel() string {
	models := whisper.ListModels()
	if len(models) == 0 {
		return ""
	}
	sort.Slice(models, func(i, j int) bool {
		if models[i].SizeBytes == models[j].SizeBytes {
			return !models[i].Multilingual && models[j].Multilingual
		}
		return models[i].SizeBytes < models[j].SizeBytes
	})
	return models[0].ID
}

func getModesForModel(model provider.Model) []string {
	if model.SupportsBothModes() {
		return []string{"batch", "streaming"}
	}
	if model.SupportsStreaming && !model.SupportsBatch {
		return []string{"streaming"}
	}
	return []string{"batch"}
}

func getLanguagesForModel(model provider.Model) []string {
	// always test auto-detect, plus the first supported language if available
	if len(model.SupportedLanguages) > 0 {
		return []string{model.SupportedLanguages[0], ""}
	}
	return []string{""}
}

func langDisplay(lang string) string {
	if lang == "" {
		return "auto"
	}
	return lang
}

func truncateTestString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
