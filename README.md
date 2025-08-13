# Hyprvoice

> **Voice‑powered typing for Wayland/Hyprland — press to toggle, speak, instant paste.**
> Streams audio while you talk and **pastes the final text the moment you toggle off** → aims to be the **fastest feel** on Wayland.

**Status:** Early development (expect rough edges)

---

## TL;DR

- **Toggle workflow** (Hyprland‑friendly): press to start, press to stop.
- **Cloud streaming ASR** (MVP) → **single final paste** into the focused window.
- **Daemon with clear states & events**; desktop notifications.
- **Clipboard‑based injection** (save/restore) with **`wtype`**
- **Unixy pipeline** (small pieces, bounded channels).

---

## Requirements

- **Go 1.24.5+** (for building from source)
- Wayland + **Hyprland**
- **PipeWire** (audio capture)
- **systemd --user** (service)
- **wl-clipboard** (clipboard save/restore)
- **libnotify**/`notify-send` (optional notifications)
- `wtype` or `ydotool` (optional text injection fallback)

> Other distros may work, but Arch/Hyprland is the primary target for now.

---

## Install (Arch / Hyprland)

```bash
# AUR
yay -S hyprvoice          # or: yay -S hyprvoice-bin

# Enable user service
systemctl --user enable --now hyprvoice.service

# Hyprland keybind (toggle)
bind = SUPER, R, exec, hyprvoice toggle
```

---

## Usage

### Basic Usage
- Press your **toggle** key to start; press again to stop.
- Audio streams to the cloud ASR while you speak.
- On stop (or VAD endpoint), Hyprvoice **pastes once** into the focused window.
- Injection flow: **save clipboard → copy final text → send Ctrl+V → restore clipboard**.

### CLI Commands
```bash
# Start the daemon
hyprvoice serve

# Toggle recording on/off
hyprvoice toggle

# Check current status
hyprvoice status

# Get protocol version
hyprvoice version

# Stop the daemon
hyprvoice stop
```

---

## Status

| Component                  | State | Notes                                       |
| -------------------------- | ----- | ------------------------------------------- |
| **Daemon (control plane)** | ✅    | State, IPC, worker orchestration            |
| **Recording control**      | ✅    | `hyprvoice toggle`                          |
| **Desktop notifications**  | ✅    | `notify-send` (logs fallback)               |
| **Audio capture**          | 🔄    | PipeWire + VAD                              |
| **ASR backends**           | 🔄    | Cloud **streaming** now; local Whisper next |
| **Text injection**         | 🔄    | Clipboard paste → `wtype` → `ydotool`       |
| **Service management**     | 🔄    | `systemd --user`                            |

Legend: ✅ done · 🔄 in progress · ⏳ planned

---

## How it works

- **Model:** pipeline + central state (daemon = control plane).
- **State machine:** `idle → recording → transcribing → injecting → idle`.
- **Rule:** switch to **`transcribing`**\*\* as soon as the first audio frame is sent\*\* to the ASR.

### ASCII diagram

```
          +-------------------+        Unix socket IPC        +-----------+
CLI cmd → |   Control Daemon  | <---------------------------- |  CLI/Tool |
          |-------------------|                               +-----------+
          | State: idle/rec/  |
          |  transcribing/... |  events        events
          | Event bus (chan)  | -----> [Notifications] -----> notify-send/log
          |                   |
          |  frames    finals |
          +--+-----------+----+
             |           |
      Audio  |           |  Final Text
      Frames v           v
         +--------+   +--------+      text      +-----------+
         | Audio  |-->|  ASR   | -------------->| Injection |
         | Capture|   | Stream |                |  Worker   |
         +--------+   +--------+                +-----------+
             |              ^
             +--------------+
               backpressure via bounded channels

State (daemon):
idle --toggle--> recording --first frame--> transcribing --final--> injecting --done--> idle
```

### Data flow

1. `toggle` → **recording**
2. First frame sent → **transcribing**
3. Cloud ASR returns **final** → **injecting**
4. Paste once → **idle**
5. Notifications at each transition

---

## Build from source

```bash
git clone https://github.com/leonardotrapani/hyprvoice.git
cd hyprvoice

# Build the binary
CGO_ENABLED=1 go build -o hyprvoice ./cmd/hyprvoice

# Run tests (when available)
go test ./...

# Install locally
sudo cp hyprvoice /usr/local/bin/
```

### Dependencies
- **Cobra CLI** - Command-line interface framework
- **Go 1.24.5+** - Programming language runtime

---

## Configuration

### File Locations
- **Socket**: `~/.cache/hyprvoice/control.sock` - IPC communication
- **PID file**: `~/.cache/hyprvoice/hyprvoice.pid` - Process tracking

### Systemd Service
The daemon runs as a user service. To create a systemd service file:

```bash
# Create service file at ~/.config/systemd/user/hyprvoice.service
mkdir -p ~/.config/systemd/user
cat > ~/.config/systemd/user/hyprvoice.service << 'EOF'
[Unit]
Description=Hyprvoice daemon
After=pipewire.service

[Service]
Type=simple
ExecStart=/usr/local/bin/hyprvoice serve
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
EOF

# Enable and start
systemctl --user daemon-reload
systemctl --user enable --now hyprvoice.service
```

---

## Development

### Project Structure
```
hyprvoice/
├── cmd/hyprvoice/         # Main CLI application
├── internal/
│   ├── bus/              # IPC communication (Unix sockets)
│   ├── daemon/           # Main daemon logic and state management
│   ├── notify/           # Desktop notifications
│   └── pipeline/         # Audio processing pipeline
├── go.mod                # Go module definition
└── README.md
```

### State Machine
The daemon operates with these states:
- **idle** → **recording** → **transcribing** → **injecting** → **idle**

### IPC Protocol
Single-character commands over Unix socket:
- `t` - Toggle recording
- `s` - Get status
- `v` - Get protocol version
- `q` - Quit daemon

### Running in Development
```bash
# Terminal 1: Start daemon with logs
go run ./cmd/hyprvoice serve

# Terminal 2: Test commands
go run ./cmd/hyprvoice toggle
go run ./cmd/hyprvoice status
```

---

## Troubleshooting

### Common Issues

**Daemon won't start**
```bash
# Check if already running
hyprvoice status

# Check PID file
ls -la ~/.cache/hyprvoice/

# Remove stale files
rm ~/.cache/hyprvoice/hyprvoice.pid
rm ~/.cache/hyprvoice/control.sock
```

**No notifications**
```bash
# Test notify-send
notify-send "Test notification"

# Check if libnotify is installed
which notify-send
```

**Permission errors**
```bash
# Check socket permissions
ls -la ~/.cache/hyprvoice/control.sock

# Recreate cache directory
rm -rf ~/.cache/hyprvoice
mkdir -p ~/.cache/hyprvoice
```

### Debug Mode
```bash
# Run with verbose logging
hyprvoice serve 2>&1 | tee hyprvoice.log
```

---

## Contributing

- All PRs and issues welcome.
- Follow existing code conventions
- Add tests for new functionality
- Update documentation for user-facing changes

---

## License

MIT — see [LICENSE.md](LICENSE.md)
