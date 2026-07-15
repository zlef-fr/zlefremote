//go:build robotgo && linux

package main

import (
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xinerama"
)

// displayBounds returns each monitor's rect in RAW root-window (global)
// coordinates, straight from Xinerama. vcaesar/screenshot's GetDisplayBounds
// shifts everything relative to the primary display's origin — fine for its
// own Capture, but Injector.MoveAbs (XWarpPointer/XTEST) works in root
// coordinates, so on layouts where the primary isn't at the root origin the
// shifted bounds would land taps on the wrong spot. Index order matches
// screenshot.CaptureDisplay (both walk xinerama.QueryScreens).
func displayBounds() []DisplayInfo {
	c, err := xgb.NewConn()
	if err != nil {
		return nil
	}
	defer c.Close()

	if err := xinerama.Init(c); err != nil {
		return nil
	}
	reply, err := xinerama.QueryScreens(c).Reply()
	if err != nil {
		return nil
	}

	var out []DisplayInfo
	for _, s := range reply.ScreenInfo {
		if s.Width > 0 && s.Height > 0 {
			out = append(out, DisplayInfo{
				X: int(s.XOrg), Y: int(s.YOrg),
				W: int(s.Width), H: int(s.Height),
			})
		}
	}
	return out
}
