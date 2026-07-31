//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

// DDC/CI brightness for EXTERNAL monitors on Windows.
//
// The WMI path (brightness_windows.go) only ever reaches a laptop's own panel:
// `WmiMonitorBrightness` is exposed by the integrated display's driver, and a
// desktop with a monitor on HDMI/DisplayPort has no such instance — which is
// why brightness used to be reported as simply unavailable on those machines.
//
// Every monitor of the last fifteen years instead answers DDC/CI over the video
// cable, and Windows exposes exactly that through dxva2.dll's Monitor
// Configuration API. This file drives it with plain syscalls: no cgo, so the
// agent still cross-compiles for Windows from this Linux host.
//
// Caveats worth knowing: DDC/CI can be disabled in a monitor's own OSD menu, KVM
// switches and some docks drop the I²C lines, and a handful of panels answer the
// capability query and then ignore writes. Each call is therefore best-effort
// and a monitor that fails the initial read is left out of the list rather than
// offered as a control that does nothing.

var (
	dxva2                           = syscall.NewLazyDLL("dxva2.dll")
	user32                          = syscall.NewLazyDLL("user32.dll")
	procGetNumberOfPhysicalMonitors = dxva2.NewProc("GetNumberOfPhysicalMonitorsFromHMONITOR")
	procGetPhysicalMonitors         = dxva2.NewProc("GetPhysicalMonitorsFromHMONITOR")
	procDestroyPhysicalMonitors     = dxva2.NewProc("DestroyPhysicalMonitors")
	procGetMonitorBrightness        = dxva2.NewProc("GetMonitorBrightness")
	procSetMonitorBrightness        = dxva2.NewProc("SetMonitorBrightness")
	procEnumDisplayMonitors         = user32.NewProc("EnumDisplayMonitors")
	procGetMonitorInfoW             = user32.NewProc("GetMonitorInfoW")
)

const physicalMonitorDescriptionSize = 128

// PHYSICAL_MONITOR: a handle plus a human description ("Generic PnP Monitor",
// or the model name on better drivers).
type physicalMonitor struct {
	handle      syscall.Handle
	description [physicalMonitorDescriptionSize]uint16
}

type monitorInfoEx struct {
	cbSize    uint32
	rcMonitor [4]int32
	rcWork    [4]int32
	dwFlags   uint32
	szDevice  [32]uint16
}

// ddcMonitor is one DDC/CI-controllable display.
type ddcMonitor struct {
	handle  syscall.Handle
	name    string
	minimum uint32
	maximum uint32
}

// enumerateDDC opens every physical monitor Windows knows about and keeps the
// ones that answer a brightness read.
func enumerateDDC() []ddcMonitor {
	var handles []syscall.Handle
	var names []string

	callback := syscall.NewCallback(func(hMonitor syscall.Handle, _ syscall.Handle, _ uintptr, _ uintptr) uintptr {
		var count uint32
		r, _, _ := procGetNumberOfPhysicalMonitors.Call(
			uintptr(hMonitor), uintptr(unsafe.Pointer(&count)))
		if r == 0 || count == 0 {
			return 1 // keep enumerating
		}
		monitors := make([]physicalMonitor, count)
		r, _, _ = procGetPhysicalMonitors.Call(
			uintptr(hMonitor), uintptr(count), uintptr(unsafe.Pointer(&monitors[0])))
		if r == 0 {
			return 1
		}

		// prefer the adapter's device name when the description is the useless
		// generic one, so two identical panels are still tellable apart
		var info monitorInfoEx
		info.cbSize = uint32(unsafe.Sizeof(info))
		device := ""
		if r, _, _ := procGetMonitorInfoW.Call(uintptr(hMonitor), uintptr(unsafe.Pointer(&info))); r != 0 {
			device = syscall.UTF16ToString(info.szDevice[:])
		}

		for i := range monitors {
			label := syscall.UTF16ToString(monitors[i].description[:])
			if label == "" || label == "Generic PnP Monitor" {
				if device != "" {
					label = trimDisplayDevice(device)
				}
			}
			handles = append(handles, monitors[i].handle)
			names = append(names, label)
		}
		return 1
	})
	procEnumDisplayMonitors.Call(0, 0, callback, 0)

	var found []ddcMonitor
	for i, h := range handles {
		var minimum, current, maximum uint32
		r, _, _ := procGetMonitorBrightness.Call(
			uintptr(h),
			uintptr(unsafe.Pointer(&minimum)),
			uintptr(unsafe.Pointer(&current)),
			uintptr(unsafe.Pointer(&maximum)),
		)
		if r == 0 || maximum <= minimum {
			// no DDC/CI (or it is switched off in the monitor's menu)
			procDestroyPhysicalMonitors.Call(1, uintptr(unsafe.Pointer(&physicalMonitor{handle: h})))
			continue
		}
		found = append(found, ddcMonitor{handle: h, name: names[i], minimum: minimum, maximum: maximum})
	}
	return found
}

// trimDisplayDevice turns `\\.\DISPLAY2` into `Display 2`.
func trimDisplayDevice(device string) string {
	for i := len(device) - 1; i >= 0; i-- {
		if device[i] == '\\' {
			return "Monitor " + device[i+1:]
		}
	}
	return device
}

// ddcBright is the Brightener backed by DDC/CI.
type ddcBright struct{ monitors []ddcMonitor }

func newDDCBrightener() (ddcBright, bool) {
	m := enumerateDDC()
	return ddcBright{monitors: m}, len(m) > 0
}

func (d ddcBright) Available() bool { return len(d.monitors) > 0 }

func (d ddcBright) Screens() []BrightScreen {
	screens := make([]BrightScreen, 0, len(d.monitors))
	for i, m := range d.monitors {
		pct := -1
		var minimum, current, maximum uint32
		r, _, _ := procGetMonitorBrightness.Call(
			uintptr(m.handle),
			uintptr(unsafe.Pointer(&minimum)),
			uintptr(unsafe.Pointer(&current)),
			uintptr(unsafe.Pointer(&maximum)),
		)
		if r != 0 && maximum > minimum {
			pct = int(float64(current-minimum) / float64(maximum-minimum) * 100)
		}
		name := m.name
		if name == "" {
			name = fmt.Sprintf("Monitor %d", i+1)
		}
		screens = append(screens, BrightScreen{Name: name, Pct: pct})
	}
	return screens
}

func (d ddcBright) Set(display, pct int) {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	apply := func(m ddcMonitor) {
		// monitors report their own scale; 0..100 is our wire unit, not theirs
		value := m.minimum + uint32(float64(m.maximum-m.minimum)*float64(pct)/100)
		procSetMonitorBrightness.Call(uintptr(m.handle), uintptr(value))
	}
	if display < 0 {
		for _, m := range d.monitors {
			apply(m)
		}
		return
	}
	if display < len(d.monitors) {
		apply(d.monitors[display])
	}
}
