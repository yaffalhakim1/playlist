//go:build windows

package proc

import (
	"os/exec"
	"syscall"
)

// createNoWindow gives the child a console that is never shown. It is the flag
// Windows documents for exactly this: a console program run from a GUI app.
const createNoWindow = 0x08000000

// hide sets both of the flags that keep a console from appearing.
//
// CREATE_NO_WINDOW is the one that matters: the child gets a console of its own
// that is never drawn, so its standard streams are real files rather than the
// absent ones a detached process would have. HideWindow asks for the same
// thing in the older way, which costs nothing and covers a launcher that
// ignores the newer flag.
func hide(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
	cmd.SysProcAttr.CreationFlags |= createNoWindow
}
