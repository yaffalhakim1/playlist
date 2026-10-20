package src

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"youtube-playlist/internal/music"
)

// TestPlayerPageGeometry checks that the player and the queue beside it land
// inside the window, on a window of any size.
//
// The window is drawn through the shell's own layout, so this catches what
// the window shows, not what a simpler layout would.
func TestPlayerPageGeometry(t *testing.T) {
	for _, size := range []struct {
		name          string
		width, height int
	}{
		{"a window", 1240, 820},
		{"a small window", 1000, 620},
	} {
		t.Run(size.name, func(t *testing.T) {
			a := &app{art: newArtwork(), current: 0, page: pagePlayer}
			a.queue = []music.Track{
				{ID: "one", Title: "It Will Rain", Artist: "Bruno Mars"},
				{ID: "two", Title: "Roar", Artist: "Katy Perry"},
			}
			a.playing = true

			tester := ui.NewTester(a.frame, size.width, size.height)
			tester.Frame()

			// The track is the title of the page: there is no heading above
			// it, because the track is what the page is.
			track, ok := tester.Find("It Will Rain")
			if !ok {
				t.Fatalf("the player does not name the track: %v", tester.Texts())
			}
			if track.Y < 0 {
				t.Errorf("the track name sits at y=%.0f, above the window", track.Y)
			}

			next, ok := tester.Find("Up next")
			if !ok {
				t.Fatalf("the player has no queue: %v", tester.Texts())
			}
			if next.X <= track.X+track.W {
				t.Errorf("the queue sits at x=%.0f, not beside the track name ending at x=%.0f",
					next.X, track.X+track.W)
			}
			if next.X+next.W > float32(size.width) {
				t.Errorf("the queue runs off the window, to x=%.0f of %d",
					next.X+next.W, size.width)
			}
			if next.Y < 0 {
				t.Errorf("the queue sits at y=%.0f, above the window", next.Y)
			}
		})
	}
}

// TestPagesFitTheWindow draws every page on a small window, which shows that
// none of them asks for more room than it has.
func TestPagesFitTheWindow(t *testing.T) {
	pages := []string{pageHome, pagePlaylist, pagePlayer, pageQueue, pageStats, pageAppearance}
	for _, page := range pages {
		t.Run(page, func(t *testing.T) {
			a := &app{art: newArtwork(), current: 0, page: page}
			a.playlist = &music.Playlist{ID: "PL", Title: "A playlist"}
			a.queue = []music.Track{
				{ID: "one", Title: "It Will Rain", Artist: "Bruno Mars"},
				{ID: "two", Title: "Roar", Artist: "Katy Perry"},
			}
			a.visible = visibleTracks(a.queue, "")

			tester := ui.NewTester(a.frame, 1000, 620)
			tester.Frame()
			if tester.Image() == nil {
				t.Fatalf("the page %q drew nothing", page)
			}
		})
	}
}
