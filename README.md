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

- Press your **toggle** key to start; press again to stop.
- Audio streams to the cloud ASR while you speak.
- On stop (or VAD endpoint), Hyprvoice **pastes once** into the focused window.
- Injection flow: **save clipboard → copy final text → send Ctrl+V → restore clipboard**.

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
CGO_ENABLED=1 go build -o hyprvoice ./cmd/hyprvoice
go test ./...
```

---

## Contributing

- All PRs and issues welcome.

---

## License

MIT — see [LICENSE.md](LICENSE.md)
