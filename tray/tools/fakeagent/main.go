// fakeagent replays the agent's `-machine` protocol without touching the
// network, the relay or the input stack. It exists so the tray UI can be
// developed and verified (including under Wine, headless) without a phone:
//
//	go build -o /tmp/fakeagent.exe ./tools/fakeagent
//	ZLEFREMOTE_AGENT=/tmp/fakeagent.exe zlefremote-tray.exe -window
//
// Timings are compressed: a phone "arrives" a few seconds in, and leaves again
// so the roster/disconnect paths get exercised too.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	machine := flag.Bool("machine", false, "emit machine-readable '@zr key=value' lines")
	mode := flag.String("mode", "lan", "connection mode: lan | remote")
	remember := flag.Bool("remember", false, "persist this computer's encryption key locally")
	pair := flag.Duration("pair", 4*time.Second, "delay before a fake phone pairs")
	leave := flag.Duration("leave", 0, "if set, the fake phone disconnects after this long")
	ver := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *ver {
		fmt.Println("zlefremote-agent 1.6.0-fake")
		return
	}

	emit := func(k, v string) {
		if *machine {
			fmt.Printf("@zr %s=%s\n", k, v)
			os.Stdout.Sync()
		}
	}

	emit("host", "DESKTOP-FAKE")
	emit("mode", *mode)
	if *remember {
		emit("persistent", "1")
	}
	url := "https://remote.zlef.fr/r/K7X2QM#k=Zm9vYmFyYmF6cXV1eGNvcmdlZ3JhdWx0Z2FycGx5"
	if *mode == "lan" {
		url = "http://192.168.1.24:9783/r/#k=Zm9vYmFyYmF6cXV1eGNvcmdlZ3JhdWx0Z2FycGx5"
	}
	emit("url", url)
	emit("qr", os.TempDir()+`\zlefremote-qr.png`)
	emit("status", "waiting")

	time.Sleep(*pair)
	emit("peer", "join 1 192.168.1.42")
	emit("clients", "1")
	emit("event", "paired")

	if *leave > 0 {
		time.Sleep(*leave)
		emit("peer", "leave 1")
		emit("clients", "0")
		emit("event", "disconnect")
	}
	// Stay up until the tray stops us, like the real agent. (A bare `select {}`
	// would trip Go's all-goroutines-asleep detector and exit(2) instead.)
	for {
		time.Sleep(time.Hour)
	}
}
