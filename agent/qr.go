package main

import (
	"fmt"
	"os"
	"path/filepath"

	qrcode "github.com/skip2/go-qrcode"
)

// printQR renders a scannable QR to the terminal (half-block) and also drops a
// PNG next to the binary in case the terminal font can't show it. It returns
// the PNG path (empty if it could not be written) so callers can advertise it
// to a front-end over the machine protocol.
func printQR(url string) (pngPath string) {
	q, err := qrcode.New(url, qrcode.Medium)
	if err != nil {
		fmt.Println("(could not render QR)", err)
		fmt.Println(url)
		return ""
	}
	if !machineMode {
		fmt.Println(q.ToSmallString(false))
	}

	png := filepath.Join(os.TempDir(), "zlefremote-qr.png")
	if err := q.WriteFile(256, png); err != nil {
		return ""
	}
	if !machineMode {
		fmt.Printf("  QR also saved to: %s\n", png)
	}
	return png
}
