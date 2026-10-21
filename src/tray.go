package src

import (
	_ "embed"

	"github.com/egoist/mygo"

	"youtube-playlist/internal/music"
)

// tray keeps the music going when the window closes.
//
// Closing the window hides it and leaves the app in the notification area,
// where a click brings it back and the menu plays, holds and skips. Quitting
// from the menu ends the app for good.
type tray struct {
	app  *app
	item *mygo.Tray
	menu *mygo.Menu
	// tooltip is what the tray last said. It is kept even when there is no
	// icon to show it on, so what the tray would say can be read without a
	// desktop, and so a missing icon is not a missing state.
	tooltip string
}

// newTray adds the icon to the notification area.
func (a *app) newTray() {
	t := &tray{app: a}
	icon, err := trayIcon(false)
	if err != nil {
		return // no icon, no tray: the window simply closes as usual
	}
	item, err := mygo.NewTray(mygo.TrayOptions{
		Icon:    icon,
		ToolTip: "Playlist",
	})
	if err != nil {
		return
	}
	t.item = item
	t.menu = mygo.NewMenu([]*mygo.MenuItem{
		{ID: "open", Label: "Open Playlist", Click: t.show},
		mygo.Separator(),
		{ID: "toggle", Label: "Play", Click: t.toggle},
		{ID: "next", Label: "Next", Click: t.next},
		{ID: "previous", Label: "Previous", Click: t.previous},
		mygo.Separator(),
		{ID: "quit", Label: "Quit", Click: t.quit},
	})
	item.SetMenu(t.menu)
	item.OnClick(func() { t.show(nil, nil) })
	a.tray = t
}

// show brings the window back.
func (t *tray) show(_ *mygo.MenuItem, _ *mygo.Window) {
	t.app.update(func() {
		t.app.win.Show()
		t.app.win.Focus()
	})
}

// toggle plays or holds the track.
func (t *tray) toggle(_ *mygo.MenuItem, _ *mygo.Window) {
	t.app.engine.Toggle()
}

// next plays the track after this one.
func (t *tray) next(_ *mygo.MenuItem, _ *mygo.Window) {
	t.app.engine.Next()
}

// previous plays the track before this one.
func (t *tray) previous(_ *mygo.MenuItem, _ *mygo.Window) {
	t.app.engine.Previous()
}

// quit ends the app.
func (t *tray) quit(_ *mygo.MenuItem, _ *mygo.Window) {
	t.app.engine.Close()
	mygo.App.Quit()
}

// refresh writes the track that plays in the tooltip and the menu, so the
// notification area says what is on.
//
// The state it reads is handed in by the caller, which holds the app's lock:
// asking the app for it again here would lock the lock the caller holds, and
// the thread that draws would wait on itself. The tooltip is set off the
// thread that draws, because a tray call reaches the desktop and may wait for
// it.
func (t *tray) refresh(track music.Track, playing bool) {
	if t == nil {
		return
	}
	t.tooltip = tooltipFor(track, playing)
	if t.item == nil {
		return
	}
	tip := t.tooltip
	icon, err := trayIcon(playing)
	// The icon says whether the music runs, so it changes with the state.
	// Both calls reach the desktop and may wait for it, so they go off the
	// thread that draws.
	go func() {
		if err == nil {
			_ = t.item.SetIcon(icon, false)
		}
		t.item.SetToolTip(tip)
	}()
	if t.menu == nil {
		return
	}
	label := "Play"
	if playing {
		label = "Pause"
	}
	if item := t.menu.ItemByID("toggle"); item != nil {
		item.Label = label
	}
}

// tooltipFor writes what the notification area shows when the pointer rests
// on the icon.
func tooltipFor(track music.Track, playing bool) string {
	if track.Title == "" {
		return "Playlist"
	}
	if !playing {
		return "Playlist — paused"
	}
	return "Playlist — " + track.Title
}

//go:embed assets/tray-playing.png
var trayPlaying []byte

//go:embed assets/tray-paused.png
var trayPaused []byte

// trayIcon returns the icon of the notification area: the app's mark, white on
// transparent, so it reads on either desktop's tray.
//
// Playing and paused are two pictures of the same mark, the paused one dimmed.
// A tray icon cannot animate on Windows or Linux, so swapping between two is
// how it says whether the music runs.
//
// The pictures are rendered from the same geometry as logo/b-small.svg by
// scripts/render-tray.ps1.
func trayIcon(playing bool) ([]byte, error) {
	if playing {
		return trayPlaying, nil
	}
	return trayPaused, nil
}
