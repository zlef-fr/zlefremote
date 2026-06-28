# ZlefRemote — Xfce panel plugin

Start a [ZlefRemote](https://remote.zlef.fr) session and show the pairing QR code
straight from your Xfce panel — no terminal. Click the panel button, pick **Local
network** or **Remote**, hit **Start**, and scan the QR with your phone. The phone
becomes a wireless trackpad + keyboard.

![ZlefRemote Xfce panel plugin — start a session and scan from the panel](../media/zlefremote-xfce-demo.gif)

## Install

### Debian / Ubuntu / Mint / Xubuntu — apt (recommended, auto-updates)

One repo, then `apt upgrade` keeps it current:

```bash
curl -fsSL https://remote.zlef.fr/apt/zlefremote.gpg | sudo tee /usr/share/keyrings/zlefremote.gpg >/dev/null
echo "deb [signed-by=/usr/share/keyrings/zlefremote.gpg] https://remote.zlef.fr/apt stable main" \
  | sudo tee /etc/apt/sources.list.d/zlefremote.list
sudo apt update && sudo apt install zlefremote-xfce-plugin
xfce4-panel -r
```

### Arch — PKGBUILD

```bash
makepkg -si    # uses packaging/arch/PKGBUILD (fetches the release tarball)
```

### Any distro — tarball (build from source)

Grab the release tarball from <https://remote.zlef.fr> (the “Get the Xfce panel
plugin” link under Download), or build from this directory.

```bash
./install.sh            # system-wide (uses sudo) — recommended
./install.sh --user     # per-user, into ~/.local, no root
./install.sh --update   # later: fetch the latest release and reinstall
xfce4-panel -r
```

Right-click the panel → **Panel → Add New Items…** → search **ZlefRemote**.

### Updating

- **apt install** → `sudo apt update && sudo apt upgrade` (automatic).
- **tarball install** → `./install.sh --update`.
- **the bundled agent** can also update itself: `zlefremote-agent -update`
  (checks `https://remote.zlef.fr/api/agent/version`, verifies the SHA-256, and
  swaps the binary in place). apt-managed installs should use `apt upgrade`.

### Build dependencies

| Distro | Packages |
|--------|----------|
| Debian/Ubuntu | `build-essential pkg-config libgtk-3-dev libxfce4panel-2.0-dev libxfce4util-dev` |
| Fedora | `gcc make pkgconf-pkg-config gtk3-devel xfce4-panel-devel` |
| Arch | `base-devel gtk3 xfce4-panel` |

## How it works

The plugin is a thin GTK3 front-end. It launches the `zlefremote-agent` binary in
machine mode (`zlefremote-agent -machine -mode lan|remote`) and reads its
line-oriented protocol from stdout:

```
@zr mode=lan
@zr url=http://192.168.1.20:9783/#k=…
@zr qr=/tmp/zlefremote-qr.png
@zr status=waiting
@zr event=paired
@zr peer=join 1 203.0.113.45
@zr clients=1
@zr peer=leave 1
@zr clients=0
```

Nothing about the session is interpreted beyond the pairing URL, the QR image
path, and the connected-clients roster — the agent does all the crypto and input
injection, exactly as it does from a terminal. The popup shows a live **“N phones
connected”** count plus each client's IP (the phone's real address — remote mode
gets it from the relay, LAN mode from the socket), so you can always see who is
controlling the machine. The same roster prints in the terminal when you run the
agent directly. The plugin finds the agent via, in order: `$ZLEFREMOTE_AGENT`, your
`$PATH`, then `~/.local/share/zlefremote/` and `/usr/local/lib/zlefremote/`.

## Files

| File | Role |
|------|------|
| `zlefremote-ui.c` | UI + agent process control (GTK3 only) |
| `zlefremote-plugin.c` | libxfce4panel glue: panel button + popup |
| `zlefremote-standalone.c` | standalone window for dev/screenshot testing |
| `zlefremote.desktop.in` | panel plugin descriptor |
| `icons/zlefremote.svg` | panel icon |

```bash
make            # build libzlefremote.so
make standalone # build the standalone UI harness (no panel needed)
make install    # honours PREFIX / DESTDIR / LIBDIR / DATADIR
```

EN/FR, auto-detected from your locale. A zlef.fr project.
