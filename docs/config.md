# Configuration Reference

This document covers manual configuration of hyprvoice via the `config.toml` file. For most users, the interactive wizard is recommended:

```bash
hyprvoice configure
```

Configuration is stored in `~/.config/hyprvoice/config.toml` and changes are applied immediately without restarting the daemon.

## Table of Contents

- [Unified Provider System](#unified-provider-system)
- [Transcription Providers](#transcription-providers)
  - [Cloud Providers](#cloud-providers)
  - [Local Transcription (whisper-cpp)](#local-transcription-whisper-cpp)
  - [Streaming Transcription](#streaming-transcription)
  - [Language Configuration](#language-configuration)
- [Model Management](#model-management)
- [LLM Post-Processing](#llm-post-processing)
- [Keywords](#keywords)
- [Recording Configuration](#recording-configuration)
- [Text Injection](#text-injection)
- [Notifications](#notifications)
- [Example Configurations](#example-configurations)
- [Migration from Old Config Format](#migration-from-old-config-format)

## Unified Provider System

Hyprvoice uses a unified provider system where API keys are configured once and shared between transcription and LLM features:

```toml
# Configure API keys for providers you want to use
[providers.openai]
  api_key = "sk-..."           # Or set OPENAI_API_KEY env var

[providers.groq]
  api_key = "gsk_..."          # Or set GROQ_API_KEY env var

[providers.mistral]
  api_key = "..."              # Or set MISTRAL_API_KEY env var

[providers.elevenlabs]
  api_key = "..."              # Or set ELEVENLABS_API_KEY env var

[providers.deepgram]
  api_key = "..."              # Or set DEEPGRAM_API_KEY env var
```

**API key resolution order:**

1. `[providers.X]` section in config
2. Environment variable (`OPENAI_API_KEY`, `GROQ_API_KEY`, etc.)

## Transcription Providers

Hyprvoice supports multiple transcription backends. See [docs/providers.md](./providers.md) for detailed comparisons.

### Cloud Providers

### OpenAI Whisper API

Cloud-based transcription using OpenAI's Whisper API:

```toml
[transcription]
provider = "openai"
model = "whisper-1"
language = ""                   # Empty for auto-detect, or "en", "es", "fr", etc.
```

**Features:**

- High-quality transcription
- Supports 50+ languages
- Auto-detection or specify language for better accuracy

### Groq Whisper API (Transcription)

Fast cloud-based transcription using Groq's Whisper API:

```toml
[transcription]
provider = "groq-transcription"
model = "whisper-large-v3"      # Or "whisper-large-v3-turbo" for faster processing
language = ""                   # Empty for auto-detect, or "en", "es", "fr", etc.
```

**Features:**

- Ultra-fast transcription (significantly faster than OpenAI)
- Same Whisper model quality
- Supports 50+ languages
- Free tier available with generous limits

### Groq Translation API

Fast translation of audio to English using Groq's Whisper API:

```toml
[transcription]
provider = "groq-translation"
language = "es"                 # Optional: hint source language for better accuracy
model = "whisper-large-v3"
```

**Features:**

- Translates any language audio → English text
- Ultra-fast processing
- Language field hints at source language (improves accuracy)
- Always outputs English regardless of input language

### Mistral Voxtral

Transcription using Mistral's Voxtral API, excellent for European languages:

```toml
[transcription]
provider = "mistral-transcription"
model = "voxtral-mini-latest"   # Or "voxtral-mini-2507"
language = ""                   # Empty for auto-detect
```

### ElevenLabs Scribe

Transcription using ElevenLabs' Scribe API with 57+ language support:

```toml
[transcription]
provider = "elevenlabs"
model = "scribe_v1"             # Or "scribe_v2" for lower latency
language = ""                   # Empty for auto-detect
```

**Features:**

- 57+ languages supported
- Both batch and streaming models available
- Ultra-low latency streaming options

### Deepgram Nova

Fast streaming transcription using Deepgram's Nova models:

```toml
[providers.deepgram]
  api_key = "..."               # Or set DEEPGRAM_API_KEY env var

[transcription]
provider = "deepgram"
model = "nova-3"                # Or "nova-2" for different language support
language = ""                   # Empty for auto-detect
```

**Features:**

- All models are streaming-only
- Nova-3: 42 languages, best accuracy
- Nova-2: 33 languages, faster with filler word detection
- Excellent for real-time transcription and live captions

### Local Transcription (whisper-cpp)

Run Whisper models locally on your machine. No API keys, no network latency, complete privacy.

**Prerequisites:**

1. Install whisper-cli: https://github.com/ggerganov/whisper.cpp
2. Download a model: `hyprvoice model download base.en`

```toml
[transcription]
provider = "whisper-cpp"
model = "base.en"               # English-only model (fastest)
language = ""                   # Empty for auto-detect (use "en" for English-only models)
threads = 0                     # 0 = auto (uses NumCPU - 1)
```

**Available models:**

| Model | Size | Languages | Best For |
|-------|------|-----------|----------|
| `tiny.en` | 75MB | English only | Quick tests, low-power devices |
| `base.en` | 142MB | English only | Daily use, good balance |
| `small.en` | 466MB | English only | Better accuracy |
| `medium.en` | 1.5GB | English only | Best English accuracy |
| `tiny` | 75MB | 57 languages | Quick multilingual |
| `base` | 142MB | 57 languages | Daily multilingual use |
| `small` | 466MB | 57 languages | Better multilingual |
| `medium` | 1.5GB | 57 languages | Great accuracy |
| `large-v3` | 3GB | 57 languages | Best accuracy |

**Threads configuration:**

- `threads = 0` (default): auto-detects, uses NumCPU - 1 to leave one core free
- `threads = 4`: explicitly use 4 threads
- Higher thread count = faster transcription but more CPU usage

### Streaming Transcription

For real-time transcription, use streaming models:

```toml
# ElevenLabs streaming
[transcription]
provider = "elevenlabs"
model = "scribe_v1-streaming"   # Or "scribe_v2-streaming" for <150ms latency

# Deepgram streaming (all models are streaming)
[transcription]
provider = "deepgram"
model = "nova-3"

# OpenAI Realtime
[transcription]
provider = "openai"
model = "gpt-4o-realtime-preview"
```

**Streaming models:**

| Provider | Model | Latency | Languages |
|----------|-------|---------|-----------|
| ElevenLabs | `scribe_v1-streaming` | Low | 57+ |
| ElevenLabs | `scribe_v2-streaming` | <150ms | 57+ |
| Deepgram | `nova-3` | Low | 42 |
| Deepgram | `nova-2` | Very Low | 33 |
| OpenAI | `gpt-4o-realtime-preview` | Low | 57 |

### Language Configuration

Language is configured per transcription model in the `[transcription]` section:

```toml
[transcription]
provider = "openai"
model = "whisper-1"
language = ""                   # Empty for auto-detect (recommended)
# language = "en"               # English
# language = "es"               # Spanish
# language = "fr"               # French
```

When using `hyprvoice configure`, you select the language after choosing the model. Only languages supported by the selected model are shown.

**Recommendations:**

- Use auto-detect (`language = ""`) for most cases - it works well
- Specify a language if you always speak the same language (slight accuracy boost)
- English-only models (e.g., `base.en`) only support `language = "en"` or auto-detect

### Supported Languages

Hyprvoice supports 57 languages:

Afrikaans (af), Arabic (ar), Armenian (hy), Azerbaijani (az), Belarusian (be), Bosnian (bs), Bulgarian (bg), Catalan (ca), Chinese (zh), Croatian (hr), Czech (cs), Danish (da), Dutch (nl), English (en), Estonian (et), Finnish (fi), French (fr), Galician (gl), German (de), Greek (el), Hebrew (he), Hindi (hi), Hungarian (hu), Icelandic (is), Indonesian (id), Italian (it), Japanese (ja), Kannada (kn), Kazakh (kk), Korean (ko), Latvian (lv), Lithuanian (lt), Macedonian (mk), Malay (ms), Marathi (mr), Maori (mi), Nepali (ne), Norwegian (no), Persian (fa), Polish (pl), Portuguese (pt), Romanian (ro), Russian (ru), Serbian (sr), Slovak (sk), Slovenian (sl), Spanish (es), Swahili (sw), Swedish (sv), Tagalog (tl), Tamil (ta), Thai (th), Turkish (tr), Ukrainian (uk), Urdu (ur), Vietnamese (vi), Welsh (cy)

### Language-Model Compatibility

Some models only support English. When configuring via `hyprvoice configure`, only supported languages are shown for selection.

**English-only models:**

| Provider | Model |
|----------|-------|
| Groq | `distil-whisper-large-v3-en` |
| whisper-cpp | `tiny.en`, `base.en`, `small.en`, `medium.en` |

**Deepgram models** support fewer languages than the full 57 - see [providers.md](./providers.md#deepgram-language-support).

**Validation behavior:**

1. **At config time (TUI):** Only languages supported by the selected model are shown
2. **At runtime:** If config was manually edited to an invalid combination, hyprvoice logs a warning, sends a desktop notification, and falls back to auto-detect

```toml
# This combination will be rejected at validation:
[transcription]
provider = "groq-transcription"
model = "distil-whisper-large-v3-en"   # English only!
language = "es"                         # Error: model does not support Spanish
```

## Model Management

Manage local whisper models with CLI commands:

### List Models

```bash
# List all models
hyprvoice model list

# Filter by provider
hyprvoice model list --provider whisper-cpp

# Filter by type
hyprvoice model list --type transcription
```

Shows installed status `[x]` for local models and model details.

### Download Models

```bash
# Download a whisper model
hyprvoice model download base.en

# Download with progress
hyprvoice model download large-v3
```

Cloud models (OpenAI, Groq, etc.) don't require download - this is for local models only.

### Remove Models

```bash
# Remove a downloaded model
hyprvoice model remove base.en
```

## LLM Post-Processing

LLM post-processing is **enabled by default** and significantly improves transcription quality. After transcription, the text is processed by an LLM to:

- Remove stutters and repeated words ("I I I want" → "I want")
- Add proper punctuation
- Fix grammar errors
- Remove filler words ("um", "uh", "like", "you know", etc.)

### Basic Configuration

```toml
[llm]
  enabled = true               # Disable with false if you want raw transcriptions
  provider = "openai"          # "openai" or "groq"
  model = "gpt-4o-mini"        # OpenAI: "gpt-4o-mini", Groq: "llama-3.3-70b-versatile"
```

### Post-Processing Options

All options are enabled by default. Disable specific ones as needed:

```toml
[llm.post_processing]
  remove_stutters = true       # "I I I want" → "I want"
  add_punctuation = true       # Adds periods, commas, etc.
  fix_grammar = true           # Fixes grammatical errors
  remove_filler_words = true   # Removes "um", "uh", "like", "you know"
```

### Custom Prompts

Add custom instructions for specific use cases:

```toml
[llm.custom_prompt]
  enabled = true
  prompt = "Format as bullet points"
```

**Use cases for custom prompts:**

- "Format as bullet points" - for note-taking
- "Keep technical terms exactly as spoken" - for programming dictation
- "Use formal language" - for professional documents
- "Translate to Spanish" - for translation workflows

### LLM Provider Recommendations

| Provider | Model                   | Best For                            |
| -------- | ----------------------- | ----------------------------------- |
| OpenAI   | gpt-4o-mini             | Best quality/cost balance (default) |
| Groq     | llama-3.3-70b-versatile | Fastest processing, free tier       |

## Keywords

Keywords help both transcription and LLM understand domain-specific terms, names, and technical vocabulary:

```toml
keywords = ["Hyprland", "Wayland", "PipeWire", "Claude", "TypeScript"]
```

**How keywords work:**

- **Transcription**: Passed as initial_prompt to Whisper, improving recognition of these terms
- **LLM**: Included in the system prompt to ensure correct spelling

**When to use keywords:**

- Names of people, companies, or products
- Technical terminology specific to your field
- Acronyms or abbreviations
- Words commonly misheard by speech-to-text

## Recording Configuration

Audio capture settings:

```toml
[recording]
sample_rate = 16000        # Audio sample rate in Hz (16000 recommended for speech)
channels = 1               # Number of audio channels (1 = mono, 2 = stereo)
format = "s16"             # Audio format (s16 = 16-bit signed integers)
buffer_size = 8192         # Internal buffer size in bytes (larger = less CPU, more latency)
device = ""                # PipeWire device name (empty = default microphone)
channel_buffer_size = 30   # Audio frame buffer size (frames to buffer)
timeout = "5m"             # Maximum recording duration (e.g., "30s", "2m", "5m")
```

### Recording Timeout

- Prevents accidental long recordings that could consume resources
- Default: 5 minutes (`"5m"`)
- Format: Go duration strings like `"30s"`, `"2m"`, `"10m"`
- Recording automatically stops when timeout is reached

## Text Injection

Configurable text injection with multiple backends:

```toml
[injection]
backends = ["ydotool", "wtype", "clipboard"]  # Ordered fallback chain
ydotool_timeout = "5s"
wtype_timeout = "5s"
clipboard_timeout = "3s"
```

### Injection Backends

- **`ydotool`**: Uses ydotool (requires `ydotoold` daemon for ydotool v1.0.0+). Most compatible with Chromium/Electron apps.
- **`wtype`**: Uses wtype for Wayland. May have issues with some Chromium-based apps (known upstream bug).
- **`clipboard`**: Copies text to clipboard only. Most reliable, but requires manual paste.

### Fallback Chain

Backends are tried in order. The first successful one wins. Example configurations:

```toml
# Clipboard only (safest, always works)
backends = ["clipboard"]

# wtype with clipboard fallback
backends = ["wtype", "clipboard"]

# Full fallback chain (default) - best compatibility
backends = ["ydotool", "wtype", "clipboard"]

# ydotool only (if you have it set up)
backends = ["ydotool"]
```

### ydotool Setup

ydotool requires the `ydotoold` daemon running (for ydotool v1.0.0+) and access to `/dev/uinput`:

```bash
# Start ydotool daemon (systemd)
systemctl --user enable --now ydotool

# Or add user to input group
sudo usermod -aG input $USER
# Then logout/login

# For Hyprland, add to config to set correct keyboard layout:
# device:ydotoold-virtual-device {
#     kb_layout = us
# }
```

## Notifications

Desktop notification settings:

```toml
[notifications]
enabled = true             # Enable/disable notifications
type = "desktop"           # "desktop", "log", or "none"
```

### Notification Types

- **`desktop`**: Use notify-send for desktop notifications
- **`log`**: Log messages to console only
- **`none`**: Disable all notifications

### Custom Notification Messages

You can customize notification text via the `[notifications.messages]` section:

```toml
[notifications.messages]
  [notifications.messages.recording_started]
    title = "Hyprvoice"
    body = "Recording Started"
  [notifications.messages.transcribing]
    title = "Hyprvoice"
    body = "Recording Ended... Transcribing"
  [notifications.messages.llm_processing]
    title = "Hyprvoice"
    body = "Processing..."
  [notifications.messages.config_reloaded]
    title = "Hyprvoice"
    body = "Config Reloaded"
  [notifications.messages.operation_cancelled]
    title = "Hyprvoice"
    body = "Operation Cancelled"
  [notifications.messages.recording_aborted]
    body = "Recording Aborted"
  [notifications.messages.injection_aborted]
    body = "Injection Aborted"
```

**Emoji-only example** (for minimal pill-style notifications):

```toml
[notifications.messages.recording_started]
  title = ""
  body = "🎙️"
```

## Example Configurations

### Fast Transcription Only (No LLM)

```toml
[providers.groq]
  api_key = "gsk_..."

[transcription]
  provider = "groq-transcription"
  model = "whisper-large-v3-turbo"
  language = ""                 # Auto-detect

[llm]
  enabled = false
```

### High Quality with OpenAI (Default)

```toml
[providers.openai]
  api_key = "sk-..."

[transcription]
  provider = "openai"
  model = "whisper-1"
  language = ""                 # Auto-detect

[llm]
  enabled = true
  provider = "openai"
  model = "gpt-4o-mini"
```

### Budget-Friendly with Groq

```toml
[providers.groq]
  api_key = "gsk_..."

[transcription]
  provider = "groq-transcription"
  model = "whisper-large-v3-turbo"
  language = ""                 # Auto-detect

[llm]
  enabled = true
  provider = "groq"
  model = "llama-3.3-70b-versatile"
```

### Mixed Providers (Groq Transcription + OpenAI LLM)

```toml
[providers.openai]
  api_key = "sk-..."

[providers.groq]
  api_key = "gsk_..."

[transcription]
  provider = "groq-transcription"
  model = "whisper-large-v3-turbo"
  language = ""                 # Auto-detect

[llm]
  enabled = true
  provider = "openai"
  model = "gpt-4o-mini"
```

### Local Transcription (Privacy-First)

```toml
# No API keys needed!

[transcription]
  provider = "whisper-cpp"
  model = "base.en"
  language = ""                 # Auto-detect
  threads = 0                   # Auto-detect (NumCPU - 1)

[llm]
  enabled = false               # No LLM for full privacy
```

### Real-Time Streaming with Deepgram

```toml
[providers.deepgram]
  api_key = "..."

[transcription]
  provider = "deepgram"
  model = "nova-3"              # All Deepgram models are streaming
  language = ""                 # Auto-detect

[llm]
  enabled = false               # Streaming doesn't need LLM post-processing
```

### Ultra-Low Latency Streaming

```toml
[providers.elevenlabs]
  api_key = "..."

[transcription]
  provider = "elevenlabs"
  model = "scribe_v2-streaming" # <150ms latency
  language = ""                 # Auto-detect

[llm]
  enabled = false
```

### Specific Language Setup

```toml
[providers.openai]
  api_key = "sk-..."

[transcription]
  provider = "openai"
  model = "whisper-1"
  language = "es"               # Always transcribe as Spanish

[llm]
  enabled = true
  provider = "openai"
  model = "gpt-4o-mini"
```

## Migration from Old Config Format

### Language Configuration Change

Language is now configured per transcription model in `[transcription].language`. If you had `[general].language` set, move it to the transcription section:

**Old format:**

```toml
[general]
  language = "en"

[transcription]
  provider = "openai"
  model = "whisper-1"
```

**New format:**

```toml
[transcription]
  provider = "openai"
  model = "whisper-1"
  language = "en"
```

Run `hyprvoice configure` to interactively update your config.

### API Key Migration

If you're upgrading from an older version with `transcription.api_key`:

**Old format (still works):**

```toml
[transcription]
  provider = "openai"
  api_key = "sk-..."  # Legacy location
  model = "whisper-1"
```

**New format (recommended):**

```toml
[providers.openai]
  api_key = "sk-..."  # Unified location

[transcription]
  provider = "openai"
  model = "whisper-1"

[llm]
  enabled = true
  provider = "openai"
  model = "gpt-4o-mini"
```

Run `hyprvoice configure` to interactively update your config to the new format.

## Configuration Hot-Reloading

The daemon automatically watches the config file for changes and applies them immediately:

- **Notification settings**: Applied instantly
- **Injection settings**: Applied to current and future operations
- **Recording/Transcription/LLM settings**: Applied to new recording sessions
- **Invalid configs**: Rejected with error notification, daemon continues with previous config
