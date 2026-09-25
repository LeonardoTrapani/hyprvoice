# AGENTS.md

This repo is a Go CLI + daemon for voice-powered typing on Wayland/Hyprland.

## Build and run
- go mod download
- go build -o hyprvoice ./cmd/hyprvoice
- go run ./cmd/hyprvoice

## Tests
- go test ./... -short
- A few tests are not hermetic: they type into the focused window, raise
  desktop notifications, and open the microphone. They skip unless
  HYPRVOICE_TEST_INTEGRATION is set, so a plain `go test ./...` is safe to run
  on a machine you are using.

## Main structure (short)
- cmd/hyprvoice: CLI entrypoint and commands
- internal/daemon: daemon lifecycle + IPC command handling
- internal/pipeline: recording -> transcription -> processing -> injection state machine
- internal/recording: PipeWire capture
- internal/transcriber: batch/streaming adapters
- internal/llm: post-processing adapters and prompts
- internal/injection: wtype/ydotool/clipboard backends
- internal/provider: provider registry + model metadata
- internal/config: config load/validate + hot reload

## Runtime quick facts
- IPC: unix socket at ~/.cache/hyprvoice/control.sock, single-character commands
- Config: ~/.config/hyprvoice/config.toml (hot reloaded by daemon)

## Configuration
- First-time setup: hyprvoice onboarding (guided flow, no advanced settings)
- Full editor: hyprvoice configure (menu-based, includes advanced settings)

## Docs
- docs/structure.md: code map and entry points
- docs/architecture.md: deeper architecture + adapters/interfaces
- docs/config.md: config reference and paths
- docs/providers.md: provider and model details
- packaging/RELEASE.md: release and AUR workflow
