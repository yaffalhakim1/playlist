//go:build windows

package proc

import (
	"os/exec"
	"syscall"
	"testing"
)

// TestHideKeepsTheConsoleClosed checks the two flags that stop Windows from
// drawing a console window for a child process.
//
// This is the bug that put a black rectangle beside the player on every track:
// a GUI app that starts a console program without these flags makes Windows
// open a window for it. The flags are the fix, so they are what the test reads.
func TestHideKeepsTheConsoleClosed(t *testing.T) {
	cmd := exec.Command("cmd", "/c", "exit", "0")
	Hide(cmd)
	if cmd.SysProcAttr == nil {
		t.Fatal("Hide left SysProcAttr nil: the child would open a console")
	}
	if !cmd.SysProcAttr.HideWindow {
		t.Error("HideWindow is not set")
	}
	if cmd.SysProcAttr.CreationFlags&createNoWindow == 0 {
		t.Errorf("CreationFlags = %#x, want CREATE_NO_WINDOW (%#x) set",
			cmd.SysProcAttr.CreationFlags, createNoWindow)
	}
}

// TestHideKeepsTheFlagsItFinds checks that Hide adds to the attributes rather
// than replacing them, so a caller that set a flag of its own does not lose it.
func TestHideKeepsTheFlagsItFinds(t *testing.T) {
	cmd := exec.Command("cmd", "/c", "exit", "0")
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000200} // CREATE_NEW_PROCESS_GROUP
	Hide(cmd)
	if got := cmd.SysProcAttr.CreationFlags; got&0x00000200 == 0 {
		t.Errorf("CreationFlags = %#x, lost the caller's flag", got)
	}
	if got := cmd.SysProcAttr.CreationFlags; got&createNoWindow == 0 {
		t.Errorf("CreationFlags = %#x, want CREATE_NO_WINDOW set", got)
	}
}
