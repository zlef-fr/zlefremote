# ZlefRemote

**Your phone is the trackpad.** Control your computer's mouse and keyboard from
any phone — over your local Wi-Fi, or end-to-end encrypted from anywhere through
[remote.zlef.fr](https://remote.zlef.fr). No app store, no account.

```
phone browser ──(AES-256-GCM)──▶ relay (sees only ciphertext) ──▶ agent ──▶ your OS
                       or, on LAN: phone browser ──────────────▶ agent ──▶ your OS
```

Two pieces:

- **The agent** — a single portable binary you run on the computer you want to
  control (Linux / Windows / macOS). It injects mouse & keyboard events.
- **The remote** — a web page that runs in any phone browser. Nothing to install
  on the phone; you open it by scanning the agent's QR code.

## How it works

1. Run the agent. Pick **Local network** or **Remote**.
2. It prints a QR code (and a URL). Scan it with your phone.
3. The phone becomes a trackpad + keyboard + media remote.

## Security model

- The agent generates a random **256-bit key** on every run.
- That key is placed **only in the QR code's URL fragment** (`#k=…`) — the part
  browsers never transmit to a server. The phone reads it locally.
- Every command is sealed with **AES-256-GCM** before it touches the wire.
- In **Remote** mode the relay (`remote.zlef.fr`) only ever sees a room code and
  opaque ciphertext. It holds no key and cannot read a single keystroke.
- In **LAN** mode traffic never leaves your network; the key still gates access
  (a client without it cannot produce a frame that decrypts → cannot inject).

The browser crypto (WebCrypto `AES-GCM`) and the Go crypto (`crypto/cipher` GCM)
are wire-compatible: `base64url(iv) + "." + base64url(ciphertext)`.

## Run the agent

```bash
./zlefremote-agent                 # interactive: choose LAN or Remote
./zlefremote-agent --mode lan      # local network, default port 9783
./zlefremote-agent --mode remote   # pair through remote.zlef.fr (E2EE)
./zlefremote-agent --mode lan --port 8080
./zlefremote-agent --mode remote --relay remote.zlef.fr
```

## Build from source

Real OS input control uses [`robotgo`](https://github.com/go-vgo/robotgo), which
is CGO and builds best **natively per-OS**:

| OS | Prerequisites |
|----|---------------|
| Linux | `gcc libc6-dev libx11-dev libxtst-dev libxkbcommon-dev xorg-dev libxext-dev` |
| macOS | `xcode-select --install` |
| Windows | a gcc (mingw-w64 / msys2) |

```bash
cd agent
./build.sh          # real agent for the current OS  → ../dist/
./build.sh stub     # portable, CGO-free stub (logs input; for testing transport)
```

Prebuilt binaries for all three platforms are produced by GitHub Actions
(`.github/workflows/build.yml`).

## The relay (this repo's web service)

Node + `ws`, stateless blind relay. Modular:

- `server.js` — HTTP (landing + phone client + downloads) and the `/ws` relay.
- `lib/rooms.js` — in-memory rooms; forwards opaque frames host ↔ clients.
- `lib/pages.js`, `lib/i18n.js` — SSR landing (EN/FR).
- `public/app/` — the phone client (also embedded into the agent for LAN mode).

```bash
npm install && npm start    # PORT=10067
```

A zlef.fr project · EN/FR.
