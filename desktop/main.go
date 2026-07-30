package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/zlef-fr/zlefremote/desktop/internal/core"
)

// version is reported in the handshake's user agent and by -version.
const version = "1.0.0"

const usage = `ZlefRemote desktop — drive another computer from this one.

  zlefremote-desktop [flags] [pairing-link]

The pairing link is what the agent prints (and QR-encodes) on the machine you
want to control, e.g. https://remote.zlef.fr/r/AB12CD#k=<key>. Passing it here
skips the picker; without it the app opens on the machine list.

Inside a session, Right Ctrl is the local menu key: tap it to take or release
control, Right Ctrl + / lists every shortcut.
`

func main() {
	lang := flag.String("lang", "", "interface language: en | fr (default: your system locale)")
	fs := flag.Bool("fullscreen", false, "start fullscreen")
	ver := flag.Bool("version", false, "print version and exit")
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage, "\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *ver {
		fmt.Println("zlefremote-desktop", version)
		return
	}

	// the transport stamps this into its handshake user agent
	core.ClientVersion = version

	store := core.LoadStore()
	if *lang != "" {
		store.Prefs.Lang = *lang
	}
	app := NewApp(store, pickLang(store.Prefs.Lang))

	// A link on the command line is the "one click from the chat message" path:
	// connect straight away instead of showing the picker.
	if arg := flag.Arg(0); arg != "" {
		t, err := core.ParseTarget(arg)
		if err != nil {
			fmt.Fprintln(os.Stderr, "bad pairing link:", err)
			os.Exit(2)
		}
		app.startSession(t)
	}

	ebiten.SetWindowTitle(app.T("app.title"))
	ebiten.SetWindowSize(1280, 800)
	ebiten.SetWindowSizeLimits(720, 480, -1, -1)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	// The picture arrives at 10–20 fps; running the loop at 60 keeps the
	// pointer, HUD and animations smooth without redrawing the frame more than
	// once per arrival.
	ebiten.SetTPS(60)
	ebiten.SetScreenClearedEveryFrame(true)
	if *fs || store.Prefs.Fullscreen {
		ebiten.SetFullscreen(true)
	}

	if err := ebiten.RunGame(app); err != nil && !errors.Is(err, ebiten.Termination) {
		log.Fatal(err)
	}
	app.stopSession()
	_ = store.Save()
}
