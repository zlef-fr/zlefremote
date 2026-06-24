package main

import (
	"fmt"
	"os"
	"path/filepath"

	qrcode "github.com/skip2/go-qrcode"
)

// printQR renders a scannable QR to the terminal (half-block) and also drops a
// PNG next to the binary in case the terminal font can't show it.
func printQR(url string) {
	q, err := qrcode.New(url, qrcode.Medium)
	if err != nil {
		fmt.Println("(could not render QR)", err)
		fmt.Println(url)
		return
	}
	fmt.Println(q.ToSmallString(false))

	png := filepath.Join(os.TempDir(), "zlefremote-qr.png")
	if err := q.WriteFile(256, png); err == nil {
		fmt.Printf("  QR also saved to: %s\n", png)
	}
}
