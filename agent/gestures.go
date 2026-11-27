package main

// Semantic multi-touch gestures.
//
// A phone can recognise a three-finger swipe; only the computer knows what a
// three-finger swipe MEANS on the desktop it happens to be running. Alt+Tab
// switches windows on Linux and Windows and does nothing useful on a Mac;
// Super+D shows the desktop there and maximises a window on macOS. So the
// clients send an INTENT ("app-next") and this file picks the chord for the
// host OS — one place to be wrong, and old clients that hardcoded chords keep
// working because {t:'key'} is untouched.
type gestureAction struct {
	Key  string
	Mods []string
}

// gestureTable maps runtime.GOOS → intent → chord.
//
// Choices worth their comment:
//   - overview on Linux is the Super key ALONE (GNOME's activities, KDE's
//     launcher). Super+Up would maximise the window instead.
//   - show-desktop on macOS is F11, the Mission Control default; there is no
//     Cmd+D equivalent (that duplicates a file in Finder).
//   - nav-back on macOS is Cmd+[ rather than Cmd+Left: it works in Safari,
//     Chrome and Finder, while Cmd+Left is line-start in every text field.
//   - zoom is a key chord, not Ctrl+wheel, because the wheel's sign is a
//     platform argument we don't need to have. Ctrl+= / Ctrl+- is what every
//     browser, editor and viewer already binds.
var gestureTable = map[string]map[string]gestureAction{
	"linux": {
		"app-next":       {Key: "tab", Mods: []string{"alt"}},
		"app-prev":       {Key: "tab", Mods: []string{"alt", "shift"}},
		"overview":       {Key: "meta"},
		"show-desktop":   {Key: "d", Mods: []string{"meta"}},
		"workspace-next": {Key: "right", Mods: []string{"ctrl", "alt"}},
		"workspace-prev": {Key: "left", Mods: []string{"ctrl", "alt"}},
		"nav-back":       {Key: "left", Mods: []string{"alt"}},
		"nav-forward":    {Key: "right", Mods: []string{"alt"}},
		"zoom-in":        {Key: "=", Mods: []string{"ctrl"}},
		"zoom-out":       {Key: "-", Mods: []string{"ctrl"}},
	},
	"windows": {
		"app-next":       {Key: "tab", Mods: []string{"alt"}},
		"app-prev":       {Key: "tab", Mods: []string{"alt", "shift"}},
		"overview":       {Key: "tab", Mods: []string{"meta"}},
		"show-desktop":   {Key: "d", Mods: []string{"meta"}},
		"workspace-next": {Key: "right", Mods: []string{"ctrl", "meta"}},
		"workspace-prev": {Key: "left", Mods: []string{"ctrl", "meta"}},
		"nav-back":       {Key: "left", Mods: []string{"alt"}},
		"nav-forward":    {Key: "right", Mods: []string{"alt"}},
		"zoom-in":        {Key: "=", Mods: []string{"ctrl"}},
		"zoom-out":       {Key: "-", Mods: []string{"ctrl"}},
	},
	"darwin": {
		"app-next":       {Key: "tab", Mods: []string{"meta"}},
		"app-prev":       {Key: "tab", Mods: []string{"meta", "shift"}},
		"overview":       {Key: "up", Mods: []string{"ctrl"}},
		"show-desktop":   {Key: "f11"},
		"workspace-next": {Key: "right", Mods: []string{"ctrl"}},
		"workspace-prev": {Key: "left", Mods: []string{"ctrl"}},
		"nav-back":       {Key: "[", Mods: []string{"meta"}},
		"nav-forward":    {Key: "]", Mods: []string{"meta"}},
		"zoom-in":        {Key: "=", Mods: []string{"meta"}},
		"zoom-out":       {Key: "-", Mods: []string{"meta"}},
	},
}

// gestureIDs is the vocabulary advertised in the welcome frame, so a client can
// grey out what a given agent doesn't know instead of firing into the void.
var gestureIDs = []string{
	"app-next", "app-prev", "overview", "show-desktop",
	"workspace-next", "workspace-prev", "nav-back", "nav-forward",
	"zoom-in", "zoom-out",
}

// gestureFor resolves an intent for one OS. Unknown OS falls back to the Linux
// table: an X11-ish desktop is the better guess than doing nothing.
func gestureFor(goos, id string) (gestureAction, bool) {
	table, ok := gestureTable[goos]
	if !ok {
		table = gestureTable["linux"]
	}
	a, ok := table[id]
	return a, ok
}
