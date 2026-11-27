# ZlefRemote for Android

The phone client, as a real app. `fr.zlef.remote`, Flutter + a small Kotlin layer.

It replaces the installable PWA that used to live at `remote.zlef.fr/r/`. That web
client still exists — it is the no-install fallback, and the UI the agent embeds
for LAN mode — but it is no longer a PWA, and it is no longer the phone product.

## Why native

Every item here is something the browser refused to do:

| | Web client | This app |
|---|---|---|
| Session with the screen off | dropped (timers frozen, socket closed) | foreground service keeps it |
| Lock-screen transport controls | silent-audio hack that stole audio focus and paused the phone's own music — shipped default-off | real `MediaSession`, no audio focus, default-on |
| QR pairing | `BarcodeDetector`, Chromium only, behind a button | camera opens straight into the scanner |
| Volume rocker → computer volume | impossible | opt-in |
| Where the device keys live | `localStorage` (readable by anything on the origin, wiped by "clear site data") | Android Keystore |
| Opening a pairing link | a browser tab | App Links, verified against `/.well-known/assetlinks.json` |
| LAN mode | the agent's self-signed certificate needed a manual "Advanced → Proceed", and before agent 1.7.1 it silently failed altogether (no `crypto.subtle` outside a secure context) | the certificate is accepted for private addresses only; frames stay sealed either way |
| Zooming the remote screen | none — a 4K desktop letterboxed onto a phone, taps were guesses | pinch to zoom and pan |

## Rule: say it, don't hide it

The web client hid what the computer couldn't do — no screen tab, no brightness
card, no explanation. This app never hides a feature. Every capability is
rendered; the ones that can't run are visible, inert, and carry the reason and
the fix (`lib/core/caps.dart`, `ZCapabilityNotice`, and the full report at the
bottom of the in-session settings sheet).

Agent age is inferred from the shape of the `welcome` frame: `cap.keyhold` and
`cap.clip` both arrived in agent 1.7.0, and Go marshals every key of that map,
so their absence dates the computer's agent. That is what lets the app say
"update the agent" instead of leaving a dead button.

## Layout

```
lib/core/     crypto (AES-256-GCM envelope, shared with agent/crypto.go)
              target (pairing-link parsing) · conn (relay + LAN transport)
              session (state, commands, screen frames) · caps (what/why not)
              devices (Keystore-backed store) · settings · native · updater
lib/ui/       theme (da.zlef.fr tokens) · widgets · screens · panes/
android/      MainActivity (channels, volume keys, APK install)
              ZrSessionService (foreground service + media notification)
tool/         gen_icons.py — legacy launcher PNGs from the same mark
```

## Build

```bash
cd app && flutter build apk --release      # unsigned-ish: falls back to the debug key
bash ../scripts/release.sh                 # signed + published to dist/app/ + manifest
```

`scripts/release.sh` writes `android/key.properties` from `/root/.env.zlefremote`,
builds through `limit` (cgroup scope, so a gradle blow-up can't take the host
down), and publishes the APK plus the `/api/app/release` manifest the in-app
updater reads.

## iOS

There is no iOS build. Compiling and signing one needs macOS and a paid Apple
developer account; this host is Linux and the project has neither. The Dart code
is written to stay portable (no Android-only plugin in `lib/core/`), so the day
a Mac is available it is a build, not a rewrite. Until then the landing page says
so plainly instead of showing a greyed-out App Store badge.

## Gotcha: MIUI (and any OEM battery manager) background-restricts sideloads

Verified on a Redmi Note 8 Pro, Android 10. A sideloaded app lands with
`RUN_IN_BACKGROUND: ignore` / `RUN_ANY_IN_BACKGROUND: ignore`. The foreground
service still runs and the notification stays on screen, but **the socket is
killed the moment the screen goes off** — the session looks alive and answers
nothing. Measured: the link dropped within ~a minute of screen-off; with the
restriction lifted it held two minutes of screen-off with zero disconnects.

The app tries to say so rather than die quietly: a banner on the control screen
and a capability notice in settings, both with a button that opens the right
settings page, gated on `ActivityManager.isBackgroundRestricted()`.

**Caveat — the detection is unverified on MIUI.** With the appops forced to
`ignore` on the Redmi, the banner did *not* appear, i.e. that API returned false
even though the restriction was in force. `isBackgroundRestricted()` is the
AOSP-correct signal (it maps to `OP_RUN_ANY_IN_BACKGROUND != MODE_ALLOWED`), so
it should hold on stock Android, but MIUI appears to enforce its own background
policy without surfacing it there. Treat the banner as best-effort until it has
been seen firing on a real restricted device; a heartbeat-based fallback ("the
link died while the screen was off") would catch what the API misses.

Lifting the restriction by hand for a test device:

    adb shell appops set fr.zlef.remote RUN_IN_BACKGROUND allow
    adb shell appops set fr.zlef.remote RUN_ANY_IN_BACKGROUND allow

## Gotcha: installing on that phone at all

`adb install` fails with `INSTALL_FAILED_USER_RESTRICTED` (MIUI's "install via
USB" wall, which wants a Mi account + SIM). The path that works is `adb push`
to `/sdcard/Download` then tapping the file in the Downloads UI. Play Protect
then fails its own verification with `Finsky: Could not find valid auth token`
and reports a bare "Application non installée"; clear it for the install and
put it back afterwards:

    adb shell settings put global package_verifier_enable 0
    adb shell settings put global package_verifier_user_consent -1
    # … install …
    adb shell settings put global package_verifier_enable 1
    adb shell settings put global package_verifier_user_consent 1
