# ZlefRemote — Xfce panel plugin

Start a [ZlefRemote](https://remote.zlef.fr) session and show the pairing QR code
straight from your Xfce panel — no terminal. Click the panel button, pick **Local
network** or **Remote**, hit **Start**, and scan the QR with your phone. The phone
becomes a wireless trackpad + keyboard.

![ZlefRemote Xfce panel plugin — start a session and scan from the panel](../media/zlefremote-xfce-demo.gif)

## Install

Grab the release tarball from <https://remote.zlef.fr> (the “Get the Xfce panel
plugin” link under Download), or build from this directory.

```bash
# from the tarball or a repo checkout:
./install.sh            # system-wide (uses sudo) — recommended
./install.sh --user     # per-user, into ~/.local, no root
```

Then reload the panel and add the item:

```bash
xfce4-panel -r
```

Right-click the panel → **Panel → Add New Items…** → search **ZlefRemote**.

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
```

Nothing about the session is interpreted beyond the pairing URL and the QR image
path — the agent does all the crypto and input injection, exactly as it does from
a terminal. The plugin finds the agent via, in order: `$ZLEFREMOTE_AGENT`, your
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
