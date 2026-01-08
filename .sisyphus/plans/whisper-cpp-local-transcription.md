# Plan: Add Local whisper.cpp Transcription

## Summary
Add `whisper-cpp` as a new transcription provider using CLI subprocess, with integrated model download in `hyprvoice configure` and standalone `hyprvoice model` commands.

---

## Tasks

### 1. Model Management Package
**File:** `internal/whisper/models.go` (NEW)

```go
package whisper

const DefaultModelsDir = "~/.local/share/hyprvoice/models"

type ModelInfo struct {
    Name     string
    Size     string
    Desc     string
    URL      string
    Filename string
}

var AvailableModels = []ModelInfo{
    // English-only (faster)
    {Name: "tiny.en", Size: "75MB", Desc: "Fastest, English only", ...},
    {Name: "base.en", Size: "142MB", Desc: "Fast, good accuracy (recommended)", ...},
    {Name: "small.en", Size: "466MB", Desc: "Better accuracy, slower", ...},
    // Multilingual
    {Name: "tiny", Size: "75MB", Desc: "Fastest, 99 languages", ...},
    {Name: "base", Size: "142MB", Desc: "Fast, 99 languages", ...},
    {Name: "small", Size: "466MB", Desc: "Better accuracy, 99 languages", ...},
}

func GetModelsDir() string
func DownloadModel(name string, onProgress func(downloaded, total int64)) error
func ListInstalledModels() ([]string, error)
func GetModelPath(name string) string
func RemoveModel(name string) error
func IsModelInstalled(name string) bool
```

Download URL pattern: `https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-{name}.bin`

---

### 2. Whisper.cpp Adapter
**File:** `internal/transcriber/adapter_whisper_cpp.go` (NEW)

```go
package transcriber

type WhisperCppAdapter struct {
    modelPath string
    language  string
    threads   int
}

func NewWhisperCppAdapter(config Config) *WhisperCppAdapter

func (a *WhisperCppAdapter) Transcribe(ctx context.Context, audioData []byte) (string, error)
```

**Implementation:**
1. Write audioData to temp WAV file (reuse `convertToWAV`)
2. Build command: `whisper-cli -m <model> -l <lang> -t <threads> --no-timestamps -f <temp.wav>`
3. Execute with context timeout
4. Parse stdout - whisper-cli outputs transcription to stdout
5. Cleanup temp file
6. Return text

**Error handling:**
- whisper-cli not found → clear error message with install instructions
- Model file not found → suggest `hyprvoice model download`
- Transcription timeout → configurable via context

---

### 3. Config Updates
**File:** `internal/config/config.go` (MODIFY)

Add to `TranscriptionConfig`:
```go
ModelPath string `toml:"model_path"` // path to .bin model file
Threads   int    `toml:"threads"`    // CPU threads (default: 4)
```

Add validation for `whisper-cpp`:
```go
case "whisper-cpp":
    if config.ModelPath == "" {
        return fmt.Errorf("model_path required for whisper-cpp provider")
    }
    if _, err := os.Stat(expandPath(config.ModelPath)); os.IsNotExist(err) {
        return fmt.Errorf("model file not found: %s (run 'hyprvoice model download')", config.ModelPath)
    }
    // No API key required
```

Default threads to 4 if not set.

---

### 4. Transcriber Factory Update
**File:** `internal/transcriber/transcriber.go` (MODIFY)

Add case:
```go
case "whisper-cpp":
    adapter = NewWhisperCppAdapter(config)
```

Note: No API key check for whisper-cpp.

---

### 5. CLI Model Commands
**File:** `cmd/hyprvoice/main.go` (MODIFY)

Add commands:
```go
rootCmd.AddCommand(modelCmd())

func modelCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "model",
        Short: "Manage whisper.cpp models",
    }
    cmd.AddCommand(
        modelListCmd(),
        modelDownloadCmd(),
        modelRemoveCmd(),
    )
    return cmd
}
```

#### `hyprvoice model list`
```
Available models:
  NAME       SIZE    DESCRIPTION
  tiny.en    75MB    Fastest, English only
  base.en    142MB   Fast, good accuracy (recommended)
  small.en   466MB   Better accuracy, slower
  tiny       75MB    Fastest, 99 languages
  base       142MB   Fast, 99 languages  
  small      466MB   Better accuracy, 99 languages

Installed:
  ✓ base.en  (~/.local/share/hyprvoice/models/ggml-base.en.bin)
```

#### `hyprvoice model download <name>`
```
$ hyprvoice model download base.en
Downloading ggml-base.en.bin (142MB)...
[████████████████████████████████] 100% 142MB/142MB

✓ Model saved to ~/.local/share/hyprvoice/models/ggml-base.en.bin

To use this model, add to your config:
  [transcription]
  provider = "whisper-cpp"
  model_path = "~/.local/share/hyprvoice/models/ggml-base.en.bin"
```

#### `hyprvoice model remove <name>`
```
$ hyprvoice model remove base.en
Remove model base.en? [y/N] y
✓ Removed ~/.local/share/hyprvoice/models/ggml-base.en.bin
```

---

### 6. Configure Wizard Updates
**File:** `cmd/hyprvoice/main.go` (MODIFY)

Add to provider selection:
```
Select transcription provider:
  1. openai               - OpenAI Whisper API (cloud-based)
  2. groq-transcription   - Groq Whisper API (fast transcription)
  3. groq-translation     - Groq Whisper API (translate to English)
  4. mistral-transcription - Mistral Voxtral API
  5. elevenlabs           - ElevenLabs Scribe API
  6. whisper-cpp          - Local transcription (offline, private)
```

When whisper-cpp selected:
```
🔒 whisper.cpp - Local Transcription

Checking for whisper-cli... ✓ found

Checking for installed models...
  No models found in ~/.local/share/hyprvoice/models/

Would you like to download a model now? [Y/n] y

Select model:
  English-only (faster):
    1. tiny.en   (75MB)  - Fastest
    2. base.en   (142MB) - Recommended for dictation
    3. small.en  (466MB) - Better accuracy

  Multilingual (99 languages):
    4. tiny      (75MB)  - Fastest
    5. base      (142MB) - Good balance
    6. small     (466MB) - Better accuracy

Model [1-6] (default: 2): 2

Downloading ggml-base.en.bin...
[████████████████████████████████] 100%

✓ Model downloaded!

Note: You can adjust threads in config.toml (default: 4)
```

If whisper-cli not found:
```
⚠ whisper-cli not found!

Install whisper.cpp first:
  Arch Linux: yay -S whisper.cpp
  Other: see https://github.com/ggerganov/whisper.cpp

Continue anyway? [y/N]
```

---

### 7. README Updates
**File:** `README.md` (MODIFY)

#### Update provider list in Features section:
```markdown
- **Multiple transcription backends**: OpenAI, Groq, Mistral, Eleven Labs, and **whisper.cpp (local/offline)**
```

#### Add new section after ElevenLabs:

```markdown
#### whisper.cpp Local (Privacy-First)

**100% offline transcription** - your voice never leaves your machine. No API keys, no cloud, no data collection.

```toml
[transcription]
provider = "whisper-cpp"
model_path = "~/.local/share/hyprvoice/models/ggml-base.en.bin"
language = "en"    # or empty for auto-detect
threads = 4        # CPU threads (adjust based on your CPU)
```

**Quick setup:**
```bash
# 1. Install whisper.cpp
yay -S whisper.cpp          # Arch Linux
# or build from source: https://github.com/ggerganov/whisper.cpp

# 2. Download a model and configure
hyprvoice configure         # interactive setup with model download
# or manually:
hyprvoice model download base.en
```

**Available models:**

| Model    | Size  | Speed   | Languages | Best For                     |
| -------- | ----- | ------- | --------- | ---------------------------- |
| tiny.en  | 75MB  | Fastest | English   | Quick notes, testing         |
| base.en  | 142MB | Fast    | English   | **Daily dictation (recommended)** |
| small.en | 466MB | Moderate| English   | When accuracy matters        |
| tiny     | 75MB  | Fastest | 99        | Multilingual, speed priority |
| base     | 142MB | Fast    | 99        | Multilingual, balanced       |
| small    | 466MB | Moderate| 99        | Multilingual, accuracy       |

**Tips:**
- `.en` models are faster and more accurate for English
- Use multilingual models only if you need other languages
- Adjust `threads` based on your CPU (4-8 is usually good)
- First transcription may be slower (model loading)

**Features:**
- 🔒 100% offline - complete privacy
- ⚡ Fast inference on modern CPUs  
- 🎯 Optimized quantized models
- 🌍 99 language support (multilingual models)
```

#### Update Development Status table:
```markdown
| whisper.cpp support    | ✅     | Local offline transcription                   |
```

Remove the "⏳ Planned" entries for whisper.cpp.

#### Update default config example:
Add whisper-cpp to provider comment:
```toml
provider = "openai"  # "openai", "groq-transcription", "groq-translation", "mistral-transcription", "elevenlabs", or "whisper-cpp"
```

---

### 8. Default Config Template
**File:** `internal/config/config.go` (MODIFY)

Update `SaveDefaultConfig()` to include whisper-cpp options in comments:

```toml
# Speech Transcription Configuration
[transcription]
  provider = "openai"          # Options: openai, groq-transcription, groq-translation, mistral-transcription, elevenlabs, whisper-cpp
  api_key = ""                 # API key (not needed for whisper-cpp)
  language = ""                # Language code (empty for auto-detect)
  model = "whisper-1"          # Model name (ignored for whisper-cpp)
  # model_path = ""            # For whisper-cpp: path to .bin model file
  # threads = 4                # For whisper-cpp: CPU threads to use
```

---

### 9. AUR Package Update
**File:** `packaging/PKGBUILD` (MODIFY)

Add optional dependency:
```bash
optdepends=(
    'whisper.cpp: local offline transcription'
)
```

---

## File Summary

| File | Action | Description |
|------|--------|-------------|
| `internal/whisper/models.go` | NEW | Model download/management |
| `internal/transcriber/adapter_whisper_cpp.go` | NEW | CLI subprocess adapter |
| `internal/config/config.go` | MODIFY | Add model_path, threads fields + validation |
| `internal/transcriber/transcriber.go` | MODIFY | Add whisper-cpp case to factory |
| `cmd/hyprvoice/main.go` | MODIFY | Add model commands + configure wizard |
| `README.md` | MODIFY | Documentation for local transcription |
| `packaging/PKGBUILD` | MODIFY | Add optdepends |

---

## Implementation Order

1. `internal/whisper/models.go` - model management (foundation)
2. `internal/transcriber/adapter_whisper_cpp.go` - the adapter
3. `internal/config/config.go` - config fields + validation
4. `internal/transcriber/transcriber.go` - factory update
5. `cmd/hyprvoice/main.go` - model commands + configure wizard
6. `README.md` - documentation
7. `packaging/PKGBUILD` - AUR update
8. Test end-to-end

---

## Testing Checklist

- [ ] `hyprvoice model list` shows available/installed models
- [ ] `hyprvoice model download base.en` downloads with progress
- [ ] `hyprvoice model remove base.en` removes model
- [ ] `hyprvoice configure` with whisper-cpp offers model download
- [ ] Configure wizard handles missing whisper-cli gracefully
- [ ] Transcription works with downloaded model
- [ ] Config validation catches missing model file
- [ ] Threads setting respected
- [ ] Language setting works (en vs auto-detect)
- [ ] Context cancellation stops transcription
- [ ] Error messages are clear and actionable
