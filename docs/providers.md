# Provider Comparison Guide

This guide helps you choose the right transcription provider for your use case.

## Transcription Providers

| Provider | Type | Models | Languages | Streaming | Speed | Quality | Cost |
|----------|------|--------|-----------|-----------|-------|---------|------|
| **OpenAI** | Cloud | 4 | 57 | Yes | Fast | Excellent | $0.006/min |
| **Groq** | Cloud | 3 | 57 (1 EN-only) | No | Very Fast | Excellent | Free tier |
| **Mistral** | Cloud | 2 | 57 | No | Fast | Good | Pay per use |
| **ElevenLabs** | Cloud | 4 | 57+ | Yes | Fast | Excellent | Pay per use |
| **Deepgram** | Cloud | 4 | 33-42 | Yes | Very Fast | Excellent | Pay per use |
| **whisper-cpp** | Local | 9 | 57 (4 EN-only) | No | Varies | Excellent | Free |

### OpenAI

The original Whisper provider. Reliable and well-documented.

**Models:**
- `whisper-1` - Production speech-to-text (batch)
- `gpt-4o-transcribe` - High quality with GPT-4o (batch)
- `gpt-4o-mini-transcribe` - Faster with GPT-4o Mini (batch)
- `gpt-4o-realtime-preview` - Real-time streaming

**Best for:** General use, high accuracy requirements, streaming needs

### Groq

Extremely fast inference using specialized hardware. OpenAI-compatible API.

**Models:**
- `whisper-large-v3` - Full Whisper v3, best accuracy
- `whisper-large-v3-turbo` - Faster with slightly lower accuracy
- `distil-whisper-large-v3-en` - **English only**, fastest option

**Best for:** Speed-critical applications, English-only use cases, budget-conscious users

### Mistral

European provider with Voxtral transcription models.

**Models:**
- `voxtral-mini-latest` - Latest Voxtral, recommended
- `voxtral-mini-2507` - Stable version from July 2025

**Best for:** European data residency requirements, Mistral ecosystem users

### ElevenLabs

Known for voice synthesis, also offers excellent transcription via Scribe.

**Models:**
- `scribe_v1` - 90+ languages, best accuracy (batch)
- `scribe_v2` - Lower latency, real-time optimized (batch)
- `scribe_v1-streaming` - Real-time transcription
- `scribe_v2-streaming` - Real-time with <150ms latency

**Best for:** Applications needing both TTS and STT, ultra-low latency streaming

### Deepgram

Streaming-first provider with Nova models. Excellent for real-time applications.

**Models:**
- `nova-3` - Best accuracy, 42 languages
- `nova-3-general` - Same as nova-3
- `nova-2` - Fast, 33 languages, filler word detection
- `nova-2-general` - Same as nova-2

**Language Support:** Nova-3 supports 42 languages, Nova-2 supports 33 languages. Not all 57 languages from the master list are available.

**Best for:** Real-time transcription, live captions, meeting transcription

### whisper-cpp (Local)

Run Whisper models locally on your machine. No API keys, no network latency, complete privacy.

**Requires:** `whisper-cli` binary installed on your system.

**English-only models (faster):**
| Model | Size | Speed | Quality |
|-------|------|-------|---------|
| `tiny.en` | 75MB | Fastest | Basic |
| `base.en` | 142MB | Fast | Good |
| `small.en` | 466MB | Medium | Better |
| `medium.en` | 1.5GB | Slow | Best EN |

**Multilingual models:**
| Model | Size | Speed | Quality |
|-------|------|-------|---------|
| `tiny` | 75MB | Fastest | Basic |
| `base` | 142MB | Fast | Good |
| `small` | 466MB | Medium | Better |
| `medium` | 1.5GB | Slow | Great |
| `large-v3` | 3GB | Slowest | Best |

**Best for:** Privacy-sensitive applications, offline use, avoiding API costs

---

## LLM Providers

Used for post-processing transcriptions (formatting, summarization, etc.)

| Provider | Models | Quality | Cost |
|----------|--------|---------|------|
| **OpenAI** | gpt-4o, gpt-4o-mini | Excellent | Pay per token |
| **Groq** | llama-3.3-70b, llama-3.1-8b, mixtral-8x7b | Good-Excellent | Free tier |

---

## Choosing a Provider

### Decision Flowchart

```
Need complete privacy?
├─ Yes → whisper-cpp (local)
└─ No
   └─ Need real-time streaming?
      ├─ Yes
      │  └─ Latency critical (<150ms)?
      │     ├─ Yes → ElevenLabs scribe_v2-streaming
      │     └─ No → Deepgram nova-3 or OpenAI realtime
      └─ No (batch)
         └─ Need fastest response?
            ├─ Yes
            │  └─ English only?
            │     ├─ Yes → Groq distil-whisper-large-v3-en
            │     └─ No → Groq whisper-large-v3-turbo
            └─ No
               └─ Need highest accuracy?
                  ├─ Yes → OpenAI gpt-4o-transcribe or Groq whisper-large-v3
                  └─ No → OpenAI whisper-1 (reliable default)
```

### Quick Recommendations

| Use Case | Recommended Provider | Model |
|----------|---------------------|-------|
| General dictation | OpenAI | whisper-1 |
| Fast English | Groq | distil-whisper-large-v3-en |
| Fast multilingual | Groq | whisper-large-v3-turbo |
| Live captions | Deepgram | nova-3 |
| Ultra-low latency | ElevenLabs | scribe_v2-streaming |
| Offline/privacy | whisper-cpp | base.en or base |
| High accuracy | OpenAI | gpt-4o-transcribe |

---

## Language Support

All providers support **auto-detect mode** (recommended for most users) which automatically identifies the spoken language.

### Full Language Support (57 languages)

OpenAI, Groq (except distil model), Mistral, ElevenLabs, and whisper-cpp multilingual models support all 57 languages:

Afrikaans, Arabic, Armenian, Azerbaijani, Belarusian, Bosnian, Bulgarian, Catalan, Chinese, Croatian, Czech, Danish, Dutch, English, Estonian, Finnish, French, Galician, German, Greek, Hebrew, Hindi, Hungarian, Icelandic, Indonesian, Italian, Japanese, Kannada, Kazakh, Korean, Latvian, Lithuanian, Macedonian, Malay, Marathi, Maori, Nepali, Norwegian, Persian, Polish, Portuguese, Romanian, Russian, Serbian, Slovak, Slovenian, Spanish, Swahili, Swedish, Tagalog, Tamil, Thai, Turkish, Ukrainian, Urdu, Vietnamese, Welsh

### English-Only Models

These models only support English but are faster:

| Provider | Model |
|----------|-------|
| Groq | `distil-whisper-large-v3-en` |
| whisper-cpp | `tiny.en`, `base.en`, `small.en`, `medium.en` |

If you select an English-only model with a non-English language, hyprvoice will:
1. **At config time:** Show an error and prevent saving
2. **At runtime:** Fall back to auto-detect with a warning notification

### Deepgram Language Support

Deepgram Nova models support a subset of languages:

**Nova-3 (42 languages):** Arabic, Belarusian, Bosnian, Bulgarian, Catalan, Croatian, Czech, Danish, Dutch, English, Estonian, Finnish, French, German, Greek, Hindi, Hungarian, Indonesian, Italian, Japanese, Kannada, Korean, Latvian, Lithuanian, Macedonian, Malay, Marathi, Norwegian, Polish, Portuguese, Romanian, Russian, Serbian, Slovak, Slovenian, Spanish, Swedish, Tagalog, Tamil, Turkish, Ukrainian, Vietnamese

**Nova-2 (33 languages):** Bulgarian, Catalan, Chinese, Czech, Danish, Dutch, English, Estonian, Finnish, French, German, Greek, Hindi, Hungarian, Indonesian, Italian, Japanese, Korean, Latvian, Lithuanian, Malay, Norwegian, Polish, Portuguese, Romanian, Russian, Slovak, Spanish, Swedish, Thai, Turkish, Ukrainian, Vietnamese

---

## Streaming vs Batch

### Batch Transcription
- Send complete audio file
- Wait for full transcription
- Higher accuracy
- Better for: recordings, file processing, dictation

### Streaming Transcription
- Send audio chunks in real-time
- Get partial results immediately
- Lower latency
- Better for: live captions, voice commands, interactive apps

**Streaming providers:** OpenAI (realtime model), ElevenLabs, Deepgram

---

## Local vs Cloud

### Cloud Providers

**Pros:**
- No setup required
- Always up-to-date models
- Scales automatically
- Professional support

**Cons:**
- Requires internet connection
- API costs
- Data leaves your machine
- Potential latency

### Local (whisper-cpp)

**Pros:**
- Complete privacy
- No API costs
- Works offline
- No network latency
- Your data stays on your machine

**Cons:**
- Requires setup (install whisper-cli)
- Need to download models (75MB-3GB)
- Uses local CPU/GPU resources
- Slower on modest hardware

### When to Choose Local

- Sensitive data (medical, legal, personal)
- Offline environments
- High-volume use (avoiding API costs)
- Privacy-first applications
- Air-gapped systems

### When to Choose Cloud

- Quick setup needed
- Best accuracy required
- Real-time streaming
- Light/occasional use
- Mobile or low-power devices
