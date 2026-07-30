package main

// Windows: PowerShell's Get-Clipboard / Set-Clipboard. `-Raw` keeps the text as
// one string (without it Get-Clipboard returns an array of lines), and reading
// from stdin ($input) avoids putting the payload on a command line.
func newClipper() Clipper {
	if have("powershell") {
		return execClipper{
			readCmd:  []string{"powershell", "-NoProfile", "-NonInteractive", "-Command", "Get-Clipboard -Raw"},
			writeCmd: []string{"powershell", "-NoProfile", "-NonInteractive", "-Command", "$input | Set-Clipboard"},
		}
	}
	return noClipper{}
}
