package src

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"youtube-playlist/internal/library"
	"youtube-playlist/internal/music"
)

// TestDrawPages draws every page of the app, in the light and in the dark, on
// a window and on a small one, and writes the pictures in the temporary
// folder, so the look of each page can be looked at.
//
//	go test -run TestDrawPages ./src/
func TestDrawPages(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "playlist-pages")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("make the folder: %v", err)
	}
	queue := []music.Track{
		{ID: "one", Title: "It Will Rain (Lyrics) | Wiz Khalifa, Ed Sheeran", Artist: "Bruno Mars", Album: "Ballads", Duration: 3*time.Minute + 42*time.Second},
		{ID: "two", Title: "Roar", Artist: "Katy Perry", Album: "Prism", Duration: 3*time.Minute + 21*time.Second},
		{ID: "three", Title: "Roll Up", Artist: "Wiz Khalifa", Duration: 2*time.Minute + 58*time.Second},
		{ID: "four", Title: "Locked Out of Heaven", Artist: "Bruno Mars", Album: "Unorthodox Jukebox", Duration: 3*time.Minute + 53*time.Second},
	}
	shapes := []struct {
		name          string
		width, height int
		dark          bool
	}{
		{"", 1280, 840, false},
		{"-dark", 1280, 840, true},
		{"-small", 1000, 620, false},
		{"-wide", 1680, 980, false},
		// A page taller than the window is only seen whole by a tall one, which
		// is how the lower groups, such as Last.fm, get looked at.
		{"-tall", 1280, 1600, false},
	}
	for _, page := range []string{pageHome, pageQueue, pagePlayer, pageQueue, pageStats, pageAppearance} {
		for _, shape := range shapes {
			a := &app{art: newArtwork(), current: 0, page: page}
			// The Look page draws the Last.fm card from the scrobbler's state, so
			// the shot needs one: without it the card is left out.
			a.scrobble = newScrobbler()
			a.playlist = &music.Playlist{ID: "PL", Title: "Mix - Bruno Mars - It Will Rain"}
			a.queue = queue
			a.visible = allTracks(queue)
			a.volume = 0.7
			a.duration = 3*time.Minute + 42*time.Second
			a.position = 97 * time.Second
			a.playing = true
			a.recents = []library.Recent{
				{Link: "https://music.youtube.com/playlist?list=PLabc", Title: "Mix - Bruno Mars"},
				{Link: "https://music.youtube.com/playlist?list=PLdef", Title: "Late night drive"},
				{Link: "https://music.youtube.com/playlist?list=PLghi", Title: "Prism, in full"},
			}

			tester := ui.NewTester(a.frame, shape.width, shape.height)
			tester.SetDark(shape.dark)
			tester.Frame()
			// The picture is of the window at rest, not of a colour still on
			// its way: a face animates over a moment, and a shot taken on the
			// first frame shows where it started.
			tester.Advance(time.Second)
			tester.Frame()
			image := tester.Image()
			if image == nil {
				t.Fatalf("the page %q drew nothing", page)
			}
			path := filepath.Join(dir, page+shape.name+".png")
			file, err := os.Create(path)
			if err != nil {
				t.Fatalf("make the file: %v", err)
			}
			if err := png.Encode(file, image); err != nil {
				t.Fatalf("write the picture: %v", err)
			}
			file.Close()
			t.Logf("wrote %s", path)
		}
	}
}

// TestDrawStats draws the stats page with plays in it, which is the only way
// to see the chart: an empty chart says so instead of drawing an axis.
func TestDrawStats(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "playlist-pages")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("make the folder: %v", err)
	}
	store, err := library.Open(filepath.Join(t.TempDir(), "settings.json"))
	if err != nil {
		t.Fatalf("open the store: %v", err)
	}
	a := &app{art: newArtwork(), current: -1, page: pageStats, store: store}
	titles := []struct{ id, title, artist string }{
		{"one", "It Will Rain", "Bruno Mars"},
		{"two", "Roar", "Katy Perry"},
		{"three", "Roll Up", "Wiz Khalifa"},
		{"four", "Locked Out of Heaven", "Bruno Mars"},
	}
	now := time.Now()
	for day := 0; day < statsDays; day++ {
		for i := 0; i < (day*3)%7; i++ {
			pick := titles[i%len(titles)]
			store.AddPlay(library.Play{
				TrackID:  pick.id,
				Title:    pick.title,
				Artist:   pick.artist,
				At:       now.AddDate(0, 0, -day),
				Listened: 3 * time.Minute,
			})
		}
	}
	tester := ui.NewTester(a.frame, 1280, 840)
	tester.Frame()
	tester.Advance(time.Second)
	tester.Frame()
	path := filepath.Join(dir, "stats-data.png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("make the file: %v", err)
	}
	if err := png.Encode(file, tester.Image()); err != nil {
		t.Fatalf("write the picture: %v", err)
	}
	file.Close()
	t.Logf("wrote %s", path)
}

// TestDrawEmptyPages draws the pages with nothing to show, which is what the
// app looks like the first time it opens: an empty state has to carry the page
// on its own.
func TestDrawEmptyPages(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "playlist-pages")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("make the folder: %v", err)
	}
	for _, page := range []string{pageHome, pageQueue, pagePlayer, pageQueue, pageStats} {
		a := &app{art: newArtwork(), current: -1, page: page}
		tester := ui.NewTester(a.frame, 1280, 840)
		tester.Frame()
		tester.Advance(time.Second)
		tester.Frame()
		image := tester.Image()
		if image == nil {
			t.Fatalf("the page %q drew nothing", page)
		}
		path := filepath.Join(dir, page+"-empty.png")
		file, err := os.Create(path)
		if err != nil {
			t.Fatalf("make the file: %v", err)
		}
		if err := png.Encode(file, image); err != nil {
			t.Fatalf("write the picture: %v", err)
		}
		file.Close()
		t.Logf("wrote %s", path)
	}
}
