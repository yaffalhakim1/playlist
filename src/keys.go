package src

import (
	"github.com/egoist/mygo"
)

// keys are the shortcuts of the system: the media keys of the keyboard, and
// a key to bring the window back.
//
// They work while another window has the focus, which is what makes the
// player usable from anywhere.
var keys = []struct {
	accelerator string
	action      string
}{
	{"MediaPlayPause", "toggle"},
	{"MediaNextTrack", "next"},
	{"MediaPreviousTrack", "previous"},
	{"MediaStop", "stop"},
	{"CmdOrCtrl+Shift+Space", "show"},
}

// registerKeys binds the shortcuts of the system.
//
// A key another app already holds is skipped: it is not an error, and the
// app works without it.
func (a *app) registerKeys() {
	for _, key := range keys {
		action := key.action
		_ = mygo.GlobalShortcut.Register(key.accelerator, func() {
			a.onKey(action)
		})
	}
}

// onKey does what a shortcut of the system asks.
func (a *app) onKey(action string) {
	switch action {
	case "toggle":
		a.engine.Toggle()
	case "next":
		a.engine.Next()
	case "previous":
		a.engine.Previous()
	case "stop":
		a.engine.Pause()
		a.engine.Seek(0)
	case "show":
		a.update(func() {
			a.win.Show()
			a.win.Focus()
		})
	}
}
