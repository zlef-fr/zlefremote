package main

import (
	"reflect"
	"testing"
)

// Every OS must answer every advertised intent — a client greys out what the
// welcome frame omits, so a hole here is a gesture that silently does nothing.
func TestEveryOSAnswersEveryAdvertisedGesture(t *testing.T) {
	for goos := range gestureTable {
		for _, id := range gestureIDs {
			a, ok := gestureFor(goos, id)
			if !ok {
				t.Errorf("%s: no chord for %q", goos, id)
				continue
			}
			if a.Key == "" {
				t.Errorf("%s/%s: empty key", goos, id)
			}
		}
	}
}

func TestGestureChordsAreOSSpecific(t *testing.T) {
	cases := []struct {
		goos, id string
		want     gestureAction
	}{
		{"linux", "app-next", gestureAction{Key: "tab", Mods: []string{"alt"}}},
		{"linux", "app-prev", gestureAction{Key: "tab", Mods: []string{"alt", "shift"}}},
		{"linux", "overview", gestureAction{Key: "meta"}},
		{"windows", "overview", gestureAction{Key: "tab", Mods: []string{"meta"}}},
		// the whole reason this table exists: Alt+Tab is not app switching on a Mac
		{"darwin", "app-next", gestureAction{Key: "tab", Mods: []string{"meta"}}},
		{"darwin", "show-desktop", gestureAction{Key: "f11"}},
		{"darwin", "nav-back", gestureAction{Key: "[", Mods: []string{"meta"}}},
	}
	for _, c := range cases {
		got, ok := gestureFor(c.goos, c.id)
		if !ok || !reflect.DeepEqual(got, c.want) {
			t.Errorf("gestureFor(%q,%q) = %+v,%v; want %+v", c.goos, c.id, got, ok, c.want)
		}
	}
}

// An OS we've never seen still gets a working desktop's chords rather than
// silence, and an intent we don't know stays silent rather than guessing.
func TestUnknownOSFallsBackAndUnknownIntentIsDropped(t *testing.T) {
	got, ok := gestureFor("plan9", "app-next")
	if !ok || got.Key != "tab" {
		t.Errorf("unknown OS should fall back to the linux table, got %+v,%v", got, ok)
	}
	if _, ok := gestureFor("linux", "make-coffee"); ok {
		t.Error("unknown intent resolved to a chord")
	}
}

func TestGestureCommandInjectsHostChord(t *testing.T) {
	h := newHarness(t)
	h.inj.goos = "darwin"
	h.sendCmd(t, map[string]any{"t": "gesture", "g": "app-next"})
	h.sendCmd(t, map[string]any{"t": "gesture", "g": "nope"})

	taps := h.inj.keyTaps()
	if len(taps) != 1 {
		t.Fatalf("want exactly the one known gesture injected, got %+v", taps)
	}
	if taps[0].key != "tab" || !reflect.DeepEqual(taps[0].mods, []string{"meta"}) {
		t.Errorf("darwin app-next injected %+v; want Cmd+Tab", taps[0])
	}
}

func TestWelcomeAdvertisesGestureVocabulary(t *testing.T) {
	h := newHarness(t)
	h.sendCmd(t, map[string]any{"t": "hello"})
	ws := h.frames("welcome")
	if len(ws) != 1 {
		t.Fatalf("want 1 welcome, got %d", len(ws))
	}
	w := ws[0]

	cap, _ := w["cap"].(map[string]any)
	if cap["gesture"] != true {
		t.Error("welcome must advertise cap.gesture so clients stop hardcoding chords")
	}
	list, _ := w["gestures"].([]any)
	if len(list) != len(gestureIDs) {
		t.Fatalf("welcome listed %d gestures, agent knows %d", len(list), len(gestureIDs))
	}
}
