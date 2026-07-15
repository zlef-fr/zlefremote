package main

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// runOut runs a command with a hard timeout and returns its trimmed stdout.
// Brightness tools are interactive-less one-shots; anything hanging past the
// timeout is killed so a wedged tool can't stall the brightness worker.
func runOut(timeout time.Duration, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).Output()
	return strings.TrimSpace(string(out)), err
}

// noBright is the "unsupported" brightener (no usable backend was found).
type noBright struct{}

func (noBright) Available() bool         { return false }
func (noBright) Screens() []BrightScreen { return nil }
func (noBright) Set(int, int)            {}
