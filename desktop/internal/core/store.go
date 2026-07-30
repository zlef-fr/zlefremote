package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Store is the on-disk state: the machines you have paired with and your
// preferences. It lives next to the agent's own identity file
// (~/.config/zlefremote/) and holds E2EE keys, so it is written 0600.
//
// Saving a device is only offered for agents started with --remember: those
// derive a stable room from a stable key, so the same link works tomorrow. A
// one-shot session's link would just fail on the next launch.
type Store struct {
	path    string
	Devices []Device `json:"devices"`
	Prefs   Prefs    `json:"prefs"`
}

type Device struct {
	Target
	LastUsed time.Time `json:"lastUsed"`
}

// Prefs are sticky UI choices. Quality is a Preset name (see Presets in
// stream.go); RawKeyboard sends physical keys instead of composed text.
type Prefs struct {
	Quality     string  `json:"quality"`
	RawKeyboard bool    `json:"rawKeyboard"`
	ClipSync    bool    `json:"clipSync"`
	ScrollSpeed float64 `json:"scrollSpeed"`
	Natural     bool    `json:"naturalScroll"`
	Sensitivity float64 `json:"sensitivity"`
	Lang        string  `json:"lang"`
	Fullscreen  bool    `json:"fullscreen"`
}

func defaultPrefs() Prefs {
	return Prefs{
		Quality:     "balanced",
		ClipSync:    true,
		ScrollSpeed: 1,
		Sensitivity: 1,
	}
}

func storePath() string {
	if dir, err := os.UserConfigDir(); err == nil && dir != "" {
		return filepath.Join(dir, "zlefremote", "desktop.json")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "zlefremote", "desktop.json")
}

func LoadStore() *Store {
	s := &Store{path: storePath(), Prefs: defaultPrefs()}
	b, err := os.ReadFile(s.path)
	if err != nil {
		return s
	}
	// A corrupt file must never stop the app from starting — fall back to
	// defaults and let the next save overwrite it.
	var on Store
	if json.Unmarshal(b, &on) != nil {
		return s
	}
	s.Devices = on.Devices
	s.Prefs = on.Prefs
	if s.Prefs.Quality == "" {
		s.Prefs.Quality = "balanced"
	}
	if s.Prefs.ScrollSpeed <= 0 {
		s.Prefs.ScrollSpeed = 1
	}
	if s.Prefs.Sensitivity <= 0 {
		s.Prefs.Sensitivity = 1
	}
	return s
}

func (s *Store) Save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// Remember stores (or refreshes) a device and bubbles it to the top of the list.
func (s *Store) Remember(t Target, label string) {
	if label != "" {
		t.Label = label
	}
	for i := range s.Devices {
		if s.Devices[i].Target.Same(t) {
			s.Devices[i].Target = t
			s.Devices[i].LastUsed = time.Now()
			s.sort()
			_ = s.Save()
			return
		}
	}
	s.Devices = append(s.Devices, Device{Target: t, LastUsed: time.Now()})
	s.sort()
	_ = s.Save()
}

func (s *Store) Forget(i int) {
	if i < 0 || i >= len(s.Devices) {
		return
	}
	s.Devices = append(s.Devices[:i], s.Devices[i+1:]...)
	_ = s.Save()
}

func (s *Store) sort() {
	sort.SliceStable(s.Devices, func(a, b int) bool {
		return s.Devices[a].LastUsed.After(s.Devices[b].LastUsed)
	})
}
