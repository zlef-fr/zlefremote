package main

// macOS always ships pbcopy/pbpaste — text only, though: pbcopy can't take a
// picture. osascript can, by reading the PNG from disk as «class PNGf» (the
// pasteboard's PNG flavour), which every app that accepts a pasted image reads.
func newClipper() Clipper {
	if have("pbcopy") && have("pbpaste") {
		c := execClipper{
			readCmd:  []string{"pbpaste"},
			writeCmd: []string{"pbcopy"},
		}
		if have("osascript") {
			c.image = pngViaFile("osascript",
				"-e", `set p to system attribute "ZR_CLIP_PNG"`,
				"-e", `set the clipboard to (read (POSIX file p) as «class PNGf»)`)
		}
		return c
	}
	return noClipper{}
}
