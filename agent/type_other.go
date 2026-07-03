//go:build robotgo && !linux

package main

import "github.com/go-vgo/robotgo"

// On Windows (KEYEVENTF_UNICODE) and macOS (Unicode key events) robotgo's
// TypeStr is already layout-independent, so use it directly. The custom X11
// path (type_linux.go) only exists because robotgo's Linux typing is not.
func typeText(s string) { robotgo.TypeStr(s) }
