package main

// macOS always ships pbcopy/pbpaste.
func newClipper() Clipper {
	if have("pbcopy") && have("pbpaste") {
		return execClipper{
			readCmd:  []string{"pbpaste"},
			writeCmd: []string{"pbcopy"},
		}
	}
	return noClipper{}
}
