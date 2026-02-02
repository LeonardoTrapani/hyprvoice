# Integration Testing

This doc covers `test-models`, the e2e test command for validating provider APIs work correctly.

## When to Run

Run `test-models` when:
- adding a new provider or model
- updating provider adapters
- debugging API connectivity issues
- verifying API keys are valid
- before releases (CI runs this automatically)

## Quick Start

```bash
# test all configured providers (requires API keys)
hyprvoice test-models

# output results to json
hyprvoice test-models --output results.json
```

## What It Tests

The command validates:
1. **Transcription providers**: sends a sample audio file through each model and verifies a transcription is returned
2. **LLM providers**: sends a test phrase through each model and verifies post-processing works

For each model, it reports:
- pass: API responded with valid output
- fail: API error or timeout
- skip: missing API key or dependency (e.g. whisper-cli not installed)

## API Keys

Keys are resolved from config or environment variables:
- `OPENAI_API_KEY`
- `GROQ_API_KEY`
- `DEEPGRAM_API_KEY`
- `ELEVENLABS_API_KEY`
- `MISTRAL_API_KEY`

Models without a valid key are skipped (not failed).

## Options

| Flag | Default | Description |
|------|---------|-------------|
| `--audio` | (downloaded sample) | custom WAV file to use |
| `--record-seconds` | 0 | record mic instead of using a file (e.g. `5s`) |
| `--timeout` | 45s | per-model timeout |
| `--output` | (none) | write JSON report to file |
| `--realtime` | true | pace streaming chunks in real time |
| `--both-modes` | true | test batch+streaming models in both modes |
| `--local-model` | (smallest) | whisper-cpp model to test |
| `--download-local` | false | download local model if missing |
| `--language` | en | language code for tests |
| `--keywords` | Hyprvoice,transcription,dictation | keywords for provider hints |
| `--no-keywords` | false | skip keyword hints |
| `--no-language` | false | use auto-detect instead of explicit language |

## Examples

```bash
# basic run - uses downloaded sample audio
hyprvoice test-models

# use your own audio file
hyprvoice test-models --audio ~/voice-sample.wav

# record 5 seconds from mic
hyprvoice test-models --record-seconds 5s

# longer timeout for slow connections
hyprvoice test-models --timeout 90s

# test local whisper-cpp with specific model
hyprvoice test-models --local-model base.en --download-local

# json report for CI
hyprvoice test-models --output test-results.json
```

## Output

Terminal output shows pass/fail/skip for each model:

```
test-models: total=25 pass=18 fail=2 skip=5
audio: /home/user/.cache/hyprvoice/testaudio.wav
pass openai/whisper-1 batch 1234ms output="She had your dark suit..."
pass openai/gpt-4o-transcribe batch 2156ms output="She had your dark suit..."
pass groq-transcription/whisper-large-v3 batch 456ms output="She had your dark suit..."
skip deepgram/nova-3 batch error=missing api key
fail mistral-transcription/voxtral-mini-latest batch 45000ms error=context deadline exceeded
pass openai/gpt-4o-mini llm 892ms output="I want to test Hyprvoice..."
```

JSON report (`--output`) includes full details:

```json
{
  "started_at": "2024-01-15T10:30:00Z",
  "audio_src": "/home/user/.cache/hyprvoice/testaudio.wav",
  "results": [
    {
      "provider": "openai",
      "model": "whisper-1",
      "type": "transcription",
      "mode": "batch",
      "local": false,
      "status": "pass",
      "duration_ms": 1234,
      "output": "She had your dark suit...",
      "output_chars": 45
    }
  ],
  "pass_count": 18,
  "fail_count": 2,
  "skip_count": 5,
  "total_count": 25
}
```

## CI Integration

The repo includes a GitHub Actions workflow (`.github/workflows/e2e.yml`) that runs `test-models` on demand:

```yaml
# triggered manually via workflow_dispatch
./hyprvoice test-models \
  --timeout=60s \
  --output=test-models-report.json
```

Secrets required in repo settings:
- `OPENAI_API_KEY`
- `GROQ_API_KEY`
- `DEEPGRAM_API_KEY`
- `ELEVENLABS_API_KEY`
- `MISTRAL_API_KEY`

## Adding a New Provider

When adding a new provider:

1. implement the adapter in `internal/transcriber/` or `internal/llm/`
2. register models in `internal/provider/`
3. add env var mapping if needed
4. run `test-models` to verify:
   ```bash
   hyprvoice test-models --output before-merge.json
   ```
5. add the API key to CI secrets

## Local Model Testing

whisper-cpp models require:
- `whisper-cli` binary installed
- model downloaded (`hyprvoice model download <model>`)

Use `--download-local` to auto-download during test:

```bash
hyprvoice test-models --local-model tiny.en --download-local
```

Only the smallest local model is tested by default to save time. If it works, larger models should work too.

## Troubleshooting

**All models skipped**: check API keys are set in env or config

**Timeouts**: increase `--timeout`, check network connectivity

**whisper-cpp skipped**: install `whisper-cli` and download a model

**Streaming failures**: some providers have separate streaming endpoints - check provider docs
