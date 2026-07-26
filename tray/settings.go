//go:build windows

// settings.go — the handful of preferences the tray remembers, plus the
// "start with Windows" switch. Everything lives under HKCU: the tray never
// needs admin rights (the installer does, for Program Files, but the app
// itself must run fine as a plain user).
package main

import (
	"os"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const (
	regApp = `Software\ZlefRemote`
	regRun = `Software\Microsoft\Windows\CurrentVersion\Run`
	runKey = "ZlefRemote"
)

// Settings is what survives a restart.
type Settings struct {
	Remote   bool // last chosen mode (false = LAN)
	Remember bool // "remember this computer" ticked
}

func loadSettings() Settings {
	s := Settings{}
	k, err := registry.OpenKey(registry.CURRENT_USER, regApp, registry.QUERY_VALUE)
	if err != nil {
		return s
	}
	defer k.Close()
	if v, _, err := k.GetStringValue("Mode"); err == nil {
		s.Remote = v == "remote"
	}
	if v, _, err := k.GetIntegerValue("Remember"); err == nil {
		s.Remember = v != 0
	}
	return s
}

func (s Settings) save() {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, regApp, registry.SET_VALUE)
	if err != nil {
		return
	}
	defer k.Close()
	mode := "lan"
	if s.Remote {
		mode = "remote"
	}
	k.SetStringValue("Mode", mode)
	var rem uint32
	if s.Remember {
		rem = 1
	}
	k.SetDWordValue("Remember", rem)
}

// ── autostart ───────────────────────────────────────────────────────────────

func autostartEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, regRun, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	v, _, err := k.GetStringValue(runKey)
	if err != nil {
		return false
	}
	self, err := os.Executable()
	if err != nil {
		return v != ""
	}
	// tolerate the quoted form the value is written in
	return strings.EqualFold(strings.Trim(v, `"`), self)
}

func setAutostart(on bool) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, regRun, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if !on {
		err := k.DeleteValue(runKey)
		if err == registry.ErrNotExist {
			return nil
		}
		return err
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	return k.SetStringValue(runKey, `"`+self+`"`)
}
