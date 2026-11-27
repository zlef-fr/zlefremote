package main

import "strings"

// Windows: PowerShell's Get-Clipboard / Set-Clipboard. `-Raw` keeps the text as
// one string (without it Get-Clipboard returns an array of lines), and reading
// from $input lets us pipe arbitrary text on stdin without quoting issues.
//
// Images go through WinForms (Set-Clipboard has no image parameter). The data
// object carries the picture twice: as a bitmap, which every app understands,
// and as the registered "PNG" format, which keeps transparency for the apps
// that look for it (browsers, Office, Paint.NET…). The clipboard is an OLE
// object, so PowerShell must run single-threaded-apartment (-STA), and
// SetDataObject(…, $true) keeps the data alive after PowerShell exits.
const winSetImage = `$ErrorActionPreference='Stop'
Add-Type -AssemblyName System.Windows.Forms,System.Drawing
$b=[IO.File]::ReadAllBytes($env:ZR_CLIP_PNG)
$img=[Drawing.Image]::FromStream((New-Object IO.MemoryStream(,$b)))
$d=New-Object Windows.Forms.DataObject
$d.SetImage($img)
$d.SetData('PNG',$false,(New-Object IO.MemoryStream(,$b)))
[Windows.Forms.Clipboard]::SetDataObject($d,$true)`

func newClipper() Clipper {
	if have("powershell") {
		return execClipper{
			readCmd:  []string{"powershell", "-NoProfile", "-NonInteractive", "-Command", "Get-Clipboard -Raw"},
			writeCmd: []string{"powershell", "-NoProfile", "-NonInteractive", "-Command", "$input | Set-Clipboard"},
			// one line: the script is a single argv entry, keep it free of
			// anything a command line could split on
			image: pngViaFile("powershell", "-NoProfile", "-NonInteractive", "-STA", "-Command",
				strings.ReplaceAll(winSetImage, "\n", ";")),
		}
	}
	return noClipper{}
}
