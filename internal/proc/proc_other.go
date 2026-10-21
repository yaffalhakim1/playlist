//go:build !windows

package proc

import "os/exec"

// hide does nothing: only Windows opens a console for a child process.
func hide(*exec.Cmd) {}
