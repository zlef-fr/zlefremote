package core

func NewClipboard() Clipboard {
	if haveBin("pbcopy") && haveBin("pbpaste") {
		return Clipboard{ReadCmd: []string{"pbpaste"}, WriteCmd: []string{"pbcopy"}}
	}
	return Clipboard{}
}
