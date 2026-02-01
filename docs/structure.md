# Code Structure

This doc explains how the CLI, daemon, and pipeline fit together and where to start reading the code.

## Top-level layout
- cmd/hyprvoice: CLI entrypoint and commands
- internal/: core packages
- docs/: user and developer docs
- packaging/: AUR and systemd packaging
- .github/workflows/: CI and release workflows

## Control flow (high level)
1. CLI command sends a single-character IPC command over a unix socket.
2. Daemon receives the command and owns lifecycle and state transitions.
3. Pipeline runs: recording -> transcribing -> processing -> injecting.
4. Notifications reflect state changes and errors.

State machine: idle -> recording -> transcribing -> processing -> injecting -> idle

## Key packages
- internal/bus: unix socket IPC, pid file, and client helpers
- internal/daemon: command handling, lifecycle, pipeline ownership
- internal/config: load/save/validate config and hot reload
- internal/pipeline: state machine coordinating recording/transcriber/llm/injection
- internal/recording: PipeWire audio capture
- internal/transcriber: batch and streaming provider adapters
- internal/llm: post-processing adapters and prompts
- internal/injection: wtype/ydotool/clipboard injection
- internal/notify: desktop notifications
- internal/provider: provider registry and model metadata
- internal/models/whisper: local whisper model registry and downloads
- internal/language: language metadata and compatibility rules
- internal/deps: dependency detection (ffmpeg, whisper-cli, etc.)
- internal/tui: interactive configuration wizard
- internal/testutil: shared test helpers

## Entry points and key files
- cmd/hyprvoice/main.go: CLI entrypoint and command wiring
- internal/daemon/daemon.go: daemon lifecycle and command handling
- internal/config/manager.go: config manager and hot reload
- internal/pipeline/: pipeline orchestration and state machine
- internal/recording/: audio capture implementation
- internal/transcriber/: provider-specific adapters

## IPC protocol (daemon control)
- Socket: ~/.cache/hyprvoice/control.sock
- Commands: t=toggle, c=cancel, s=status, v=version, q=quit

## Data and config locations
- Config: ~/.config/hyprvoice/config.toml
- Models: ~/.local/share/hyprvoice/models/whisper/
- PID file: ~/.cache/hyprvoice/hyprvoice.pid

## Suggested reading order
1. cmd/hyprvoice/main.go for CLI command flow.
2. internal/daemon/daemon.go for lifecycle and IPC handling.
3. internal/pipeline for state transitions and orchestration.
4. internal/recording and internal/transcriber for audio and STT.
5. internal/llm and internal/injection for text cleanup and output.
