# ZlefRemote desktop client

Drive another computer from a laptop or desktop: its screen in a window, your
own mouse, keyboard and clipboard. The phone client turns a phone into a
trackpad; this is the other half — the machine-to-machine one.

It speaks exactly the same protocol as the phone client (same rooms, same
AES-256-GCM envelope, same relay), so **any agent works with it** and nothing
new is exposed on the network.

```
this app ──(AES-256-GCM)──▶ relay (sees only ciphertext) ──▶ agent ──▶ that machine's OS
                    or, on LAN: this app ──────────────────▶ agent ──▶ that machine's OS
```

## Use it

```bash
zlefremote-desktop                                     # pick a machine in the app
zlefremote-desktop 'https://remote.zlef.fr/r/AB12CD#k=…'   # or connect straight away
```

The link is the one the agent prints (and QR-encodes) on the machine you want to
control — copy it whole: everything after `#` is the encryption key, and it never
reaches the relay. Agents started with `--remember` keep a stable address, so the
app offers to save them; a one-shot session is not saved because its link dies
with it.

### Right Ctrl is the local menu key

While Right Ctrl is held, nothing is forwarded — it drives this app instead.

| Chord | What it does |
|---|---|
| `Right Ctrl` (tap) | take / release control |
| `Right Ctrl + F` | fullscreen |
| `Right Ctrl + P` | cycle image quality (fast / balanced / sharp) |
| `Right Ctrl + M` | next monitor |
| `Right Ctrl + K` | raw keyboard (see below) |
| `Right Ctrl + H` | hide the HUD |
| `Right Ctrl + C` | push this machine's clipboard to the host |
| `Right Ctrl + Del` | send Ctrl+Alt+Del |
| `Right Ctrl + Tab` | send Alt+Tab |
| `Right Ctrl + S` | send the Windows/Super key |
| `Right Ctrl + Esc` | send Escape |
| `Right Ctrl + Q` | disconnect |
| `Right Ctrl + /` | show the shortcut list |

Clicking inside the picture also takes control, and puts the remote pointer
exactly where you clicked.

## How the input is made "transparent"

- **Pointer** — while you have control the local cursor is *captured*, so it can
  never escape the window mid-drag. The app keeps its own pointer position and
  sends it as an absolute, normalized coordinate: a dropped packet cannot leave
  the two cursors permanently offset the way relative motion would. The remote
  cursor is drawn by the client, because screen capture doesn't include it.
- **Typing across layouts** — characters are composed by *your* keyboard (AltGr,
  dead keys, accents included) and sent as text, which the agent injects
  layout-correctly. Physical key events are reserved for keys that produce no
  character (Enter, arrows, F-keys…) and for chords. `Right Ctrl + K` switches to
  **raw keyboard**, which forwards physical keys instead — right for games, and
  for a host whose layout matches yours.
- **Modifiers are lazy** — a modifier is pressed on the host only right before a
  chord or a click needs it. Holding Shift or AltGr on the host while text is
  being injected would corrupt what you typed.
- **Nothing stays stuck** — releasing control, losing window focus or dropping the
  connection lifts every key and button the app is holding on the other machine.
- **Clipboard** — copy on either machine, paste on the other (agent ≥ 1.7.0, and a
  clipboard tool present: `xclip`/`xsel`/`wl-clipboard`, `pbcopy`, PowerShell).

## Protocol additions (agent ≥ 1.7.0)

The phone client only ever taps keys; a desktop client needs more. The agent
gained, all capability-gated so older agents still work:

| Verb | Meaning |
|---|---|
| `{"t":"kdown","k":…}` / `{"t":"kup","k":…}` | hold / release one key (`cap.keyhold`) |
| `{"t":"clip","s":…}` | set the host clipboard (`cap.clip`) |
| `{"t":"clipget"}` / `{"t":"clipwatch","on":…}` | read / follow the host clipboard |
| `{"t":"ping","i":n}` → `{"t":"pong","i":n}` | round-trip time for the HUD |

Against an older agent the client falls back to `{"t":"key",…}` taps with the
live modifier list, and simply hides clipboard sharing and the latency readout.

## Build

```bash
./build.sh                 # current OS/arch
./build.sh windows amd64   # cross-build from Linux (Windows needs no cgo)
go test ./internal/...     # crypto vector, link parsing, frame reassembly
```

Ebitengine needs cgo and GL/X headers on Linux and macOS:

```bash
sudo apt-get install gcc libc6-dev libgl1-mesa-dev libx11-dev libxcursor-dev \
                     libxi-dev libxinerama-dev libxrandr-dev libxxf86vm-dev
```

Layout: `internal/core` holds everything that doesn't touch the window
(transport, E2EE, pairing links, saved machines, frame reassembly, clipboard) so
it is testable without a display; the package root is the Ebitengine app (UI,
input pump, key map, theme, i18n).

State lives in `~/.config/zlefremote/desktop.json` (0600 — it holds room keys).
