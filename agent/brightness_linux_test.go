//go:build linux

package main

import "testing"

// stubBackend is a fake Brightener with a fixed screen list, recording Set calls.
type stubBackend struct {
	screens []BrightScreen
	sets    [][2]int
}

func (s *stubBackend) Available() bool         { return true }
func (s *stubBackend) Screens() []BrightScreen { return s.screens }
func (s *stubBackend) Set(display, pct int)    { s.sets = append(s.sets, [2]int{display, pct}) }

func newSwitch(active int) (*switchBright, *stubBackend, *stubBackend) {
	hw := &stubBackend{screens: []BrightScreen{{Name: "amdgpu_bl0", Pct: 30}}}
	sw := &stubBackend{screens: []BrightScreen{{Name: "eDP", Pct: 47}, {Name: "DP-0", Pct: 100}}}
	s := &switchBright{
		avail: []BrightBackend{
			{ID: "sysfs", Label: "Backlight (sysfs)", Kind: "hardware"},
			{ID: "xrandr", Label: "Software dimming (xrandr)", Kind: "software"},
		},
		impl:   map[string]Brightener{"sysfs": hw, "xrandr": sw},
		active: active,
	}
	return s, hw, sw
}

// switchBright must satisfy both the core Brightener and the optional chooser.
var (
	_ Brightener     = (*switchBright)(nil)
	_ BackendChooser = (*switchBright)(nil)
)

func TestSwitchBrightDelegatesToActiveBackend(t *testing.T) {
	s, hw, sw := newSwitch(0)

	if s.Active() != "sysfs" {
		t.Fatalf("default Active: want sysfs (best), got %q", s.Active())
	}
	// delegates Screens/Set to the hardware backend while it's active
	if got := s.Screens(); len(got) != 1 || got[0].Name != "amdgpu_bl0" {
		t.Fatalf("Screens on sysfs: got %#v", got)
	}
	s.Set(-1, 20)
	if len(hw.sets) != 1 || hw.sets[0] != [2]int{-1, 20} {
		t.Fatalf("Set should hit the hardware backend, got %v", hw.sets)
	}
	if len(sw.sets) != 0 {
		t.Fatalf("inactive backend got a Set: %v", sw.sets)
	}

	// switch to software gamma → Screens/Set now target xrandr's outputs
	if !s.Select("xrandr") {
		t.Fatal("Select(xrandr) returned false")
	}
	if s.Active() != "xrandr" {
		t.Fatalf("after switch Active: want xrandr, got %q", s.Active())
	}
	if got := s.Screens(); len(got) != 2 || got[1].Name != "DP-0" {
		t.Fatalf("Screens on xrandr: got %#v", got)
	}
	s.Set(1, 60)
	if len(sw.sets) != 1 || sw.sets[0] != [2]int{1, 60} {
		t.Fatalf("Set should hit the xrandr backend, got %v", sw.sets)
	}
	if len(hw.sets) != 1 {
		t.Fatalf("hardware backend got an extra Set after switch: %v", hw.sets)
	}
}

func TestSwitchBrightRejectsUnknownBackend(t *testing.T) {
	s, _, _ := newSwitch(0)
	if s.Select("does-not-exist") {
		t.Fatal("Select of an unknown id should return false")
	}
	if s.Active() != "sysfs" {
		t.Fatalf("unknown Select must not change the active backend, got %q", s.Active())
	}
}

func TestSwitchBrightBackendsMetadata(t *testing.T) {
	s, _, _ := newSwitch(0)
	bs := s.Backends()
	if len(bs) != 2 {
		t.Fatalf("Backends: want 2, got %d", len(bs))
	}
	if bs[0].ID != "sysfs" || bs[0].Kind != "hardware" {
		t.Fatalf("Backends[0]: got %#v", bs[0])
	}
	if bs[1].ID != "xrandr" || bs[1].Kind != "software" {
		t.Fatalf("Backends[1]: got %#v", bs[1])
	}
}

func TestPickActiveBackend(t *testing.T) {
	avail := []BrightBackend{{ID: "sysfs"}, {ID: "xrandr"}}
	cases := []struct {
		want string
		idx  int
	}{
		{"", 0},         // no override → best default
		{"sysfs", 0},    // explicit best
		{"xrandr", 1},   // pin the software backend
		{"nonsense", 0}, // unknown → best default, never out of range
	}
	for _, c := range cases {
		if got := pickActiveBackend(avail, c.want); got != c.idx {
			t.Fatalf("pickActiveBackend(%q): want %d, got %d", c.want, c.idx, got)
		}
	}
}
