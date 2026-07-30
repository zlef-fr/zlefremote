package core

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// Local Clipboard access, mirroring the agent's approach: shell out to whatever
// Clipboard CLI the platform ships instead of pulling in a cgo dependency. That
// keeps this app buildable with CGO_ENABLED=0 on Windows and avoids a second
// X11 connection on Linux.
type Clipboard struct {
	ReadCmd  []string
	WriteCmd []string
}

func (c Clipboard) Available() bool { return len(c.ReadCmd) > 0 && len(c.WriteCmd) > 0 }

func (c Clipboard) Read() (string, error) {
	if !c.Available() {
		return "", nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, c.ReadCmd[0], c.ReadCmd[1:]...).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (c Clipboard) Write(s string) error {
	if !c.Available() {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, c.WriteCmd[0], c.WriteCmd[1:]...)
	cmd.Stdin = strings.NewReader(s)
	return cmd.Run()
}

func haveBin(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}
