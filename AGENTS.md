# AGENTS.md

This repo is a Go CLI + daemon for voice-powered typing on Wayland/Hyprland.

## Build and run
- go mod download
- go build -o hyprvoice ./cmd/hyprvoice
- go run ./cmd/hyprvoice

## Where to look
- docs/structure.md: architecture and code map
- docs/config.md: config reference and paths
- docs/providers.md: provider and model details
- packaging/RELEASE.md: release and AUR workflow
