//go:build windows

// clipboard.go — "Copy link" support. The pairing URL carries the session key
// in its fragment, so this is the one piece of the app that moves a secret; it
// only ever goes to the clipboard the user explicitly asked for.
package main

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

func copyToClipboard(hwnd uintptr, s string) bool {
	if s == "" {
		return false
	}
	if ok, _, _ := pOpenClipboard.Call(hwnd); ok == 0 {
		return false
	}
	defer pCloseClipboard.Call()
	pEmptyClipboard.Call()

	utf16 := windows.StringToUTF16(s)
	size := uintptr(len(utf16) * 2)
	mem, _, _ := pGlobalAlloc.Call(gmemMoveable, size)
	if mem == 0 {
		return false
	}
	ptr, _, _ := pGlobalLock.Call(mem)
	if ptr == 0 {
		return false
	}
	// Copy through RtlMoveMemory rather than reinterpreting the locked address
	// as a Go pointer: the block isn't Go-managed, and this keeps the unsafe
	// surface to a single documented call.
	pRtlMoveMemory.Call(ptr, uintptr(unsafe.Pointer(&utf16[0])), size)
	pGlobalUnlock.Call(mem)

	// ownership of `mem` passes to the clipboard on success
	if h, _, _ := pSetClipboardData.Call(cfUnicodeText, mem); h == 0 {
		return false
	}
	return true
}
