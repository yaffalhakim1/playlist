package src

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// shortcuts binds the keys of the player.
//
// The space plays and holds the track, the arrows move in it and change the
// volume, and the keys with control jump from track to track.
func (a *app) shortcuts(c *ui.Context) {
	c.OnShortcut(0, ui.KeySpace, func() {
		a.engine.Toggle()
	})
	c.OnShortcut(0, ui.KeyRight, func() {
		a.engine.Seek(a.times2() + 5*time.Second)
	})
	c.OnShortcut(0, ui.KeyLeft, func() {
		a.engine.Seek(a.times2() - 5*time.Second)
	})
	c.OnShortcut(0, ui.KeyUp, func() {
		a.volumeBy(0.05)
	})
	c.OnShortcut(0, ui.KeyDown, func() {
		a.volumeBy(-0.05)
	})
	c.OnShortcut(ui.Cmd, ui.KeyRight, func() {
		a.engine.Next()
	})
	c.OnShortcut(ui.Cmd, ui.KeyLeft, func() {
		a.engine.Previous()
	})
}

// volumeBy changes the volume of the player.
func (a *app) volumeBy(step float64) {
	a.mu.Lock()
	volume := a.volume + step
	if volume < 0 {
		volume = 0
	}
	if volume > 1 {
		volume = 1
	}
	a.volume = volume
	a.mu.Unlock()
	a.engine.SetVolume(volume)
	a.rememberVolume(volume)
}
