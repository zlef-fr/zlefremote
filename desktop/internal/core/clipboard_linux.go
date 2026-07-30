package core

import "os"

func NewClipboard() Clipboard {
	if os.Getenv("WAYLAND_DISPLAY") != "" && haveBin("wl-copy") && haveBin("wl-paste") {
		return Clipboard{ReadCmd: []string{"wl-paste", "--no-newline"}, WriteCmd: []string{"wl-copy"}}
	}
	if haveBin("xclip") {
		return Clipboard{
			ReadCmd:  []string{"xclip", "-selection", "Clipboard", "-o"},
			WriteCmd: []string{"xclip", "-selection", "Clipboard", "-i"},
		}
	}
	if haveBin("xsel") {
		return Clipboard{
			ReadCmd:  []string{"xsel", "--Clipboard", "--output"},
			WriteCmd: []string{"xsel", "--Clipboard", "--input"},
		}
	}
	if haveBin("wl-copy") && haveBin("wl-paste") {
		return Clipboard{ReadCmd: []string{"wl-paste", "--no-newline"}, WriteCmd: []string{"wl-copy"}}
	}
	return Clipboard{}
}
