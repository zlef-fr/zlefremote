package core

func NewClipboard() Clipboard {
	if haveBin("powershell") {
		return Clipboard{
			ReadCmd:  []string{"powershell", "-NoProfile", "-NonInteractive", "-Command", "Get-Clipboard -Raw"},
			WriteCmd: []string{"powershell", "-NoProfile", "-NonInteractive", "-Command", "$input | Set-Clipboard"},
		}
	}
	return Clipboard{}
}
