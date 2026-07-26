// mkico packs the PNG icon set into a single Windows .ico.
//
// Vista and later accept PNG-compressed entries inside an ICO, so the packer is
// a header + directory + the PNG bytes verbatim — no BMP/AND-mask encoding, no
// external tooling in the build. Run from tray/:
//
//	go run ./tools/mkico assets/icon-16.png … -o assets/zlefremote.ico
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image/png"
	"os"
)

func main() {
	var srcs []string
	out := "icon.ico"
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		if args[i] == "-o" && i+1 < len(args) {
			out = args[i+1]
			i++
			continue
		}
		srcs = append(srcs, args[i])
	}
	if len(srcs) == 0 {
		fmt.Fprintln(os.Stderr, "usage: mkico <png…> -o <file.ico>")
		os.Exit(2)
	}

	type entry struct {
		w, h int
		data []byte
	}
	var entries []entry
	for _, s := range srcs {
		data, err := os.ReadFile(s)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		cfg, err := png.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			fmt.Fprintln(os.Stderr, s+":", err)
			os.Exit(1)
		}
		entries = append(entries, entry{cfg.Width, cfg.Height, data})
	}

	var buf bytes.Buffer
	// ICONDIR: reserved, type (1 = icon), count
	binary.Write(&buf, binary.LittleEndian, uint16(0))
	binary.Write(&buf, binary.LittleEndian, uint16(1))
	binary.Write(&buf, binary.LittleEndian, uint16(len(entries)))

	offset := 6 + 16*len(entries)
	for _, e := range entries {
		// 0 in the width/height byte means 256
		bw, bh := byte(e.w), byte(e.h)
		if e.w >= 256 {
			bw = 0
		}
		if e.h >= 256 {
			bh = 0
		}
		buf.WriteByte(bw)
		buf.WriteByte(bh)
		buf.WriteByte(0) // palette entries
		buf.WriteByte(0) // reserved
		binary.Write(&buf, binary.LittleEndian, uint16(1))  // colour planes
		binary.Write(&buf, binary.LittleEndian, uint16(32)) // bits per pixel
		binary.Write(&buf, binary.LittleEndian, uint32(len(e.data)))
		binary.Write(&buf, binary.LittleEndian, uint32(offset))
		offset += len(e.data)
	}
	for _, e := range entries {
		buf.Write(e.data)
	}

	if err := os.WriteFile(out, buf.Bytes(), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("%s — %d entries, %d bytes\n", out, len(entries), buf.Len())
}
