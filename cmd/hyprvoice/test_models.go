package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/leonardotrapani/hyprvoice/internal/config"
	"github.com/leonardotrapani/hyprvoice/internal/llm"
	"github.com/leonardotrapani/hyprvoice/internal/models/whisper"
	"github.com/leonardotrapani/hyprvoice/internal/provider"
	"github.com/leonardotrapani/hyprvoice/internal/recording"
	"github.com/leonardotrapani/hyprvoice/internal/transcriber"
	"github.com/spf13/cobra"
)

const (
	mockSampleRate    = 16000
	mockChannels      = 1
	mockBitsPerSample = 16
	defaultSampleURL  = "https://raw.githubusercontent.com/mozilla/DeepSpeech/master/data/smoke_test/LDC93S1.wav"
	defaultSampleName = "testaudio.wav"
)

var (
	defaultTestKeywords = []string{"Hyprvoice", "transcription", "dictation"}
	defaultTestLanguage = "en"
)

type testModelsOptions struct {
	audioPath     string
	recordFor     time.Duration
	timeout       time.Duration
	outputPath    string
	realtime      bool
	bothModes     bool
	localModel    string
	downloadLocal bool
	language      string
	keywords      []string
	noKeywords    bool
	noLanguage    bool
}

type modelTest struct {
	provider string
	model    provider.Model
	mode     string
}

type modelTestResult struct {
	Provider    string `json:"provider"`
	Model       string `json:"model"`
	Type        string `json:"type"`
	Mode        string `json:"mode"`
	Local       bool   `json:"local"`
	Status      string `json:"status"`
	DurationMS  int64  `json:"duration_ms"`
	Output      string `json:"output,omitempty"`
	OutputChars int    `json:"output_chars,omitempty"`
	Error       string `json:"error,omitempty"`
}

type testReport struct {
	StartedAt  time.Time         `json:"started_at"`
	AudioSrc   string            `json:"audio_src"`
	Results    []modelTestResult `json:"results"`
	PassCount  int               `json:"pass_count"`
	FailCount  int               `json:"fail_count"`
	SkipCount  int               `json:"skip_count"`
	TotalCount int               `json:"total_count"`
}

func testModelsCmd() *cobra.Command {
	var opts testModelsOptions

	cmd := &cobra.Command{
		Use:   "test-models",
		Short: "Run E2E tests for all providers/models",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTestModels(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVar(&opts.audioPath, "audio", "", "WAV file to use (defaults to downloaded sample)")
	cmd.Flags().DurationVar(&opts.recordFor, "record-seconds", 0, "Record mic audio (e.g. 5s)")
	cmd.Flags().DurationVar(&opts.timeout, "timeout", 45*time.Second, "Per-model timeout")
	cmd.Flags().BoolVar(&opts.realtime, "realtime", true, "Pace streaming chunks in real time")
	cmd.Flags().BoolVar(&opts.bothModes, "both-modes", true, "Test batch+streaming models in both modes")
	cmd.Flags().StringVar(&opts.outputPath, "output", "", "Write JSON report to file")
	cmd.Flags().StringVar(&opts.localModel, "local-model", "", "whisper-cpp model ID to test")
	cmd.Flags().BoolVar(&opts.downloadLocal, "download-local", false, "Download local whisper model if missing")
	cmd.Flags().StringVar(&opts.language, "language", defaultTestLanguage, "Language code to test")
	cmd.Flags().StringSliceVar(&opts.keywords, "keywords", defaultTestKeywords, "Keywords to test")
	cmd.Flags().BoolVar(&opts.noKeywords, "no-keywords", false, "Skip keyword testing")
	cmd.Flags().BoolVar(&opts.noLanguage, "no-language", false, "Skip language (use auto-detect)")

	return cmd
}

func runTestModels(ctx context.Context, opts testModelsOptions) error {
	if opts.audioPath != "" && opts.recordFor > 0 {
		return fmt.Errorf("use either --audio or --record-seconds, not both")
	}
	if opts.timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	startedAt := time.Now().UTC()

	audio, audioSrc, err := loadTestAudio(ctx, opts)
	if err != nil {
		return err
	}

	cfg, err := loadConfigForTests()
	if err != nil {
		return err
	}

	transcriptionTests, err := buildTranscriptionTests(opts)
	if err != nil {
		return err
	}

	llmTests := buildLLMTests()

	var results []modelTestResult

	for _, test := range transcriptionTests {
		result := runTranscriptionTest(ctx, cfg, test, audio, opts)
		results = append(results, result)
	}

	for _, test := range llmTests {
		result := runLLMTest(ctx, cfg, test, opts)
		results = append(results, result)
	}

	report := summarizeReport(startedAt, audioSrc, results)
	printReport(report)

	if opts.outputPath != "" {
		if err := writeReport(opts.outputPath, report); err != nil {
			return err
		}
	}

	if report.FailCount > 0 || report.SkipCount > 0 {
		return fmt.Errorf("%d failed, %d skipped", report.FailCount, report.SkipCount)
	}

	return nil
}

func loadConfigForTests() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		if errors.Is(err, config.ErrConfigNotFound) {
			return config.DefaultConfig(), nil
		}
		return nil, err
	}
	if cfg.Providers == nil {
		cfg.Providers = make(map[string]config.ProviderConfig)
	}
	return cfg, nil
}

func buildTranscriptionTests(opts testModelsOptions) ([]modelTest, error) {
	providerNames := provider.ListProvidersWithTranscription()
	sort.Strings(providerNames)

	localModel := opts.localModel
	if localModel == "" {
		localModel = selectSmallestWhisperModel()
	}

	var tests []modelTest
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
			if model.Local {
				if providerName == provider.ProviderWhisperCpp && model.ID != localModel {
					// test only the smallest local model; if it works the rest should too
					continue
				}
			}

			if opts.bothModes && model.SupportsBothModes() {
				tests = append(tests, modelTest{provider: providerName, model: model, mode: "batch"})
				tests = append(tests, modelTest{provider: providerName, model: model, mode: "streaming"})
				continue
			}

			mode := "batch"
			if model.SupportsStreaming && !model.SupportsBatch {
				mode = "streaming"
			}
			tests = append(tests, modelTest{provider: providerName, model: model, mode: mode})
		}
	}

	return tests, nil
}

func buildLLMTests() []modelTest {
	providerNames := provider.ListProvidersWithLLM()
	sort.Strings(providerNames)

	var tests []modelTest
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
			tests = append(tests, modelTest{provider: providerName, model: model, mode: "batch"})
		}
	}

	return tests
}

func runTranscriptionTest(ctx context.Context, cfg *config.Config, test modelTest, audio []byte, opts testModelsOptions) modelTestResult {
	result := modelTestResult{
		Provider: test.provider,
		Model:    test.model.ID,
		Type:     "transcription",
		Mode:     test.mode,
		Local:    test.model.Local,
		Status:   "fail",
	}

	if test.model.Local {
		if _, err := exec.LookPath("whisper-cli"); err != nil {
			result.Status = "skip"
			result.Error = "whisper-cli not found"
			return result
		}
		if !whisper.IsInstalled(test.model.ID) {
			if opts.downloadLocal {
				dlCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
				defer cancel()
				if err := downloadLocalModel(dlCtx, test.model.ID); err != nil {
					result.Status = "fail"
					result.Error = err.Error()
					return result
				}
			} else {
				result.Status = "skip"
				result.Error = "local model not installed"
				return result
			}
		}
	}

	apiKey := resolveAPIKey(cfg, test.provider)
	if providerRequiresKey(test.provider) && apiKey == "" {
		result.Status = "skip"
		result.Error = "missing api key"
		return result
	}

	language := opts.language
	if opts.noLanguage {
		language = ""
	}
	keywords := opts.keywords
	if opts.noKeywords {
		keywords = nil
	}

	streaming := test.mode == "streaming"
	transcribeCfg := transcriber.Config{
		Provider:  test.provider,
		APIKey:    apiKey,
		Language:  language,
		Model:     test.model.ID,
		Keywords:  keywords,
		Threads:   0,
		Streaming: streaming,
	}

	testCtx, cancel := context.WithTimeout(ctx, opts.timeout)
	defer cancel()
	start := time.Now()
	text, err := runTranscriber(testCtx, transcribeCfg, audio, opts.realtime)
	result.DurationMS = time.Since(start).Milliseconds()
	if err != nil {
		result.Error = err.Error()
		return result
	}

	result.Status = "pass"
	result.Output = strings.TrimSpace(text)
	result.OutputChars = len(result.Output)
	return result
}

func runLLMTest(ctx context.Context, cfg *config.Config, test modelTest, opts testModelsOptions) modelTestResult {
	result := modelTestResult{
		Provider: test.provider,
		Model:    test.model.ID,
		Type:     "llm",
		Mode:     "batch",
		Local:    test.model.Local,
		Status:   "fail",
	}

	apiKey := resolveAPIKey(cfg, test.provider)
	if providerRequiresKey(test.provider) && apiKey == "" {
		result.Status = "skip"
		result.Error = "missing api key"
		return result
	}

	keywords := opts.keywords
	if opts.noKeywords {
		keywords = nil
	}

	llmCfg := llm.Config{
		Provider:          test.provider,
		APIKey:            apiKey,
		Model:             test.model.ID,
		RemoveStutters:    true,
		AddPunctuation:    true,
		FixGrammar:        true,
		RemoveFillerWords: true,
		CustomPrompt:      "",
		Keywords:          keywords,
	}

	adapter, err := llm.NewAdapter(llmCfg)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	input := "uh i i i want to test hyprvoice you know this is just a cleanup check"
	testCtx, cancel := context.WithTimeout(ctx, opts.timeout)
	defer cancel()
	start := time.Now()
	output, err := adapter.Process(testCtx, input)
	result.DurationMS = time.Since(start).Milliseconds()
	if err != nil {
		result.Error = err.Error()
		return result
	}

	result.Status = "pass"
	result.Output = strings.TrimSpace(output)
	result.OutputChars = len(result.Output)
	return result
}

func runTranscriber(ctx context.Context, cfg transcriber.Config, audio []byte, realtime bool) (string, error) {
	t, err := transcriber.NewTranscriber(cfg)
	if err != nil {
		return "", err
	}

	frameCh := make(chan recording.AudioFrame, 8)
	errCh, err := t.Start(ctx, frameCh)
	if err != nil {
		return "", err
	}

	sendErr := sendAudioFrames(ctx, frameCh, audio, realtime)
	close(frameCh)

	stopErr := t.Stop(ctx)
	errChErr := readErrorChannel(errCh)

	if sendErr != nil {
		return "", sendErr
	}
	if stopErr != nil {
		return "", stopErr
	}
	if errChErr != nil {
		return "", errChErr
	}

	return t.GetFinalTranscription()
}

func sendAudioFrames(ctx context.Context, frameCh chan<- recording.AudioFrame, audio []byte, realtime bool) error {
	const chunkBytes = 3200
	bytesPerSecond := mockSampleRate * (mockBitsPerSample / 8) * mockChannels
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

		if realtime {
			time.Sleep(chunkDuration)
		}
	}

	return nil
}

func readErrorChannel(errCh <-chan error) error {
	var firstErr error
	if errCh == nil {
		return nil
	}

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

func loadTestAudio(ctx context.Context, opts testModelsOptions) ([]byte, string, error) {
	if opts.audioPath != "" {
		wav, err := readWAVFile(opts.audioPath)
		if err != nil {
			return nil, "", err
		}
		return wav.data, opts.audioPath, nil
	}

	if opts.recordFor > 0 {
		audio, err := recordAudio(ctx, opts.recordFor)
		if err != nil {
			return nil, "", err
		}
		return audio, fmt.Sprintf("recording:%s", opts.recordFor), nil
	}

	path, err := ensureDefaultSample(ctx)
	if err != nil {
		return nil, "", err
	}
	wav, err := readWAVFile(path)
	if err != nil {
		return nil, "", err
	}
	return wav.data, path, nil
}

type wavData struct {
	data          []byte
	sampleRate    int
	channels      int
	bitsPerSample int
}

func readWAVFile(path string) (*wavData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseWAV(data)
}

func parseWAV(data []byte) (*wavData, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("invalid wav: too short")
	}
	if string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return nil, fmt.Errorf("invalid wav: missing riff/wave header")
	}

	offset := 12
	var fmtFound bool
	var dataFound bool
	var info wavData

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
			info.channels = int(binary.LittleEndian.Uint16(data[offset+2 : offset+4]))
			info.sampleRate = int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
			info.bitsPerSample = int(binary.LittleEndian.Uint16(data[offset+14 : offset+16]))
			fmtFound = true
		case "data":
			info.data = data[offset : offset+chunkSize]
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
	if info.bitsPerSample != mockBitsPerSample {
		return nil, fmt.Errorf("unsupported wav bits per sample: %d", info.bitsPerSample)
	}
	if info.sampleRate <= 0 {
		return nil, fmt.Errorf("invalid wav sample rate: %d", info.sampleRate)
	}
	if info.channels <= 0 {
		return nil, fmt.Errorf("invalid wav: channels=%d", info.channels)
	}
	if len(info.data)%2 != 0 {
		return nil, fmt.Errorf("invalid wav: pcm data not aligned")
	}

	monoData, err := downmixToMono(info.data, info.channels)
	if err != nil {
		return nil, err
	}
	resampled := resamplePCM16(monoData, info.sampleRate, mockSampleRate)
	if len(resampled) == 0 {
		return nil, fmt.Errorf("invalid wav: empty audio data")
	}
	info.data = resampled
	info.sampleRate = mockSampleRate
	info.channels = mockChannels
	info.bitsPerSample = mockBitsPerSample
	return &info, nil
}

func downmixToMono(data []byte, channels int) ([]byte, error) {
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

func resamplePCM16(data []byte, inRate, outRate int) []byte {
	if inRate <= 0 || outRate <= 0 {
		return data
	}
	if inRate == outRate {
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

		sample1 := sampleAtPCM16(data, srcIdx)
		sample2 := sampleAtPCM16(data, srcIdx+1)
		outSample := int16(float64(sample1)*(1-frac) + float64(sample2)*frac)

		out[i*2] = byte(outSample)
		out[i*2+1] = byte(outSample >> 8)
	}

	return out
}

func sampleAtPCM16(data []byte, idx int) int16 {
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

func recordAudio(ctx context.Context, duration time.Duration) ([]byte, error) {
	recorder := recording.NewRecorder(recording.Config{
		SampleRate:        mockSampleRate,
		Channels:          mockChannels,
		Format:            "s16",
		BufferSize:        8192,
		Device:            "",
		ChannelBufferSize: 30,
		Timeout:           duration + 2*time.Second,
	})

	frameCh, errCh, err := recorder.Start(ctx)
	if err != nil {
		return nil, err
	}

	var audio []byte
	stopCh := make(chan struct{})
	go func() {
		for frame := range frameCh {
			audio = append(audio, frame.Data...)
		}
		close(stopCh)
	}()

	select {
	case <-time.After(duration):
		recorder.Stop()
	case <-ctx.Done():
		recorder.Stop()
	}

	<-stopCh
	if err := readErrorChannel(errCh); err != nil {
		return nil, err
	}

	return audio, nil
}

func ensureDefaultSample(ctx context.Context) (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(cacheDir, "hyprvoice", defaultSampleName)
	if info, err := os.Stat(path); err == nil && info.Size() > 0 {
		return path, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}

	fmt.Printf("test-models: downloading sample audio...\n")
	if err := downloadSample(ctx, defaultSampleURL, path); err != nil {
		return "", fmt.Errorf("download sample: %w (use --audio or --record-seconds to skip download)", err)
	}
	return path, nil
}

func downloadSample(ctx context.Context, url, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	tmpPath := path + ".downloading"
	out, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	defer func() {
		out.Close()
		_ = os.Remove(tmpPath)
	}()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func resolveAPIKey(cfg *config.Config, providerName string) string {
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

func providerRequiresKey(providerName string) bool {
	p := provider.GetProvider(provider.BaseProviderName(providerName))
	if p == nil {
		return false
	}
	return p.RequiresAPIKey()
}

func selectSmallestWhisperModel() string {
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

func downloadLocalModel(ctx context.Context, modelID string) error {
	var lastPercent int64
	return whisper.Download(ctx, modelID, func(downloaded, total int64) {
		if total <= 0 {
			return
		}
		percent := downloaded * 100 / total
		if percent >= lastPercent+10 {
			fmt.Printf("downloading %s... %d%%\n", modelID, percent)
			lastPercent = percent
		}
	})
}

func summarizeReport(startedAt time.Time, audioSrc string, results []modelTestResult) testReport {
	report := testReport{
		StartedAt: startedAt,
		AudioSrc:  audioSrc,
		Results:   results,
	}
	for _, r := range results {
		report.TotalCount++
		switch r.Status {
		case "pass":
			report.PassCount++
		case "fail":
			report.FailCount++
		case "skip":
			report.SkipCount++
		}
	}
	return report
}

func printReport(report testReport) {
	fmt.Printf("test-models: total=%d pass=%d fail=%d skip=%d\n", report.TotalCount, report.PassCount, report.FailCount, report.SkipCount)
	fmt.Printf("audio: %s\n", report.AudioSrc)
	for _, r := range report.Results {
		line := fmt.Sprintf("%s %s/%s %s", r.Status, r.Provider, r.Model, r.Mode)
		if r.Type == "llm" {
			line = fmt.Sprintf("%s %s/%s llm", r.Status, r.Provider, r.Model)
		}
		if r.DurationMS > 0 {
			line += fmt.Sprintf(" %dms", r.DurationMS)
		}
		if r.Error != "" {
			line += fmt.Sprintf(" error=%s", truncateString(r.Error, 160))
		}
		if r.Output != "" {
			line += fmt.Sprintf(" output=%q", truncateString(r.Output, 120))
		}
		fmt.Println(line)
	}
}

func writeReport(path string, report testReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func truncateString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
