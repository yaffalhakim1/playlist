package src

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// TestDrawLoading draws the home page while a playlist is being read, so the
// spinner and the word beside it can be looked at: a slow playlist is the one
// moment the app has nothing else to show.
func TestDrawLoading(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "playlist-pages")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("make the folder: %v", err)
	}
	a := loaded(pageHome)
	a.loading = true
	a.link = "https://music.youtube.com/playlist?list=PLFgquLnL59alCl_2TQvOiD5Vgm1hCaGSI"

	tester := ui.NewTester(a.frame, 1280, 840)
	tester.Frame()
	// The spinner turns and the busy button settles: the shot is of the wait as
	// it looks a moment in, not of its first frame.
	tester.Advance(300 * time.Millisecond)
	tester.Frame()
	path := filepath.Join(dir, "home-loading.png")
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

// TestLoadingSaysWhatItIsDoing checks the words, not the picture: while a
// playlist is read, the page has to say so and disable the field.
func TestLoadingSaysWhatItIsDoing(t *testing.T) {
	a := &app{art: newArtwork(), current: -1, page: pageHome}
	a.loading = true

	tester := ui.NewTester(a.frame, 1280, 840)
	tester.Frame()

	if !tester.HasText("Reading the playlist. A long one takes a few seconds.") {
		t.Errorf("the page does not say it is reading: %v", tester.Texts())
	}
	if !tester.HasText("Reading the playlist…") {
		t.Errorf("the button does not say it is reading: %v", tester.Texts())
	}
}

// TestLoadingShowsOneButton checks that the Play button is gone while a
// playlist is read, rather than being joined by a second one: building the
// Play button and then swapping the variable left both in the row.
//
// The busy word appears twice in the text list, as every button's does: once
// as what it draws and once as its accessible name.
func TestLoadingShowsOneButton(t *testing.T) {
	a := &app{art: newArtwork(), current: -1, page: pageHome}
	a.loading = true

	tester := ui.NewTester(a.frame, 1280, 840)
	tester.Frame()

	play := 0
	for _, text := range tester.Texts() {
		if text == "Play" {
			play++
		}
	}
	if play != 0 {
		t.Errorf("the Play button is still drawn %d times while reading", play)
	}
	if !tester.HasText("Reading the playlist…") {
		t.Errorf("the busy button is not drawn: %v", tester.Texts())
	}
}

// TestLoadingClearsTheOldStatus checks the fix for a duplicated line: load()
// used to set a status of "Reading the playlist…" as well as the spinner, so
// two lines said the same thing one under the other.
//
// The spinner line is the only thing that speaks while a load runs; the status
// line is for what went wrong.
func TestLoadingClearsTheOldStatus(t *testing.T) {
	a := &app{art: newArtwork(), current: -1, page: pageHome, store: openStore(t)}
	a.status = "an old failure"

	// A link that is not cached goes to the network, which is the path that
	// sets the loading state. The request itself is not waited for.
	a.link = "https://music.youtube.com/playlist?list=PLnothing"
	a.load()

	a.mu.Lock()
	loading, status := a.loading, a.status
	a.mu.Unlock()

	if !loading {
		t.Fatal("a load of an uncached playlist did not start")
	}
	if status != "" {
		t.Errorf("the status line still says %q while the spinner is up", status)
	}
}

// TestAFailureStopsTheSpinner checks that a load that failed shows the error
// and not the spinner: the two are alternatives, not a stack.
func TestAFailureStopsTheSpinner(t *testing.T) {
	a := &app{art: newArtwork(), current: -1, page: pageHome}
	a.loading = false
	a.status = "That playlist would not open"

	tester := ui.NewTester(a.frame, 1280, 840)
	tester.Frame()

	if tester.HasText("Reading the playlist. A long one takes a few seconds.") {
		t.Error("the spinner line is drawn after a load failed")
	}
	if !tester.HasText("That playlist would not open") {
		t.Errorf("the failure is not shown: %v", tester.Texts())
	}
}

// TestTheFormClearsOnceAPlaylistOpens checks that a successful load empties the
// field: the link is in the sidebar now, and leaving it there invites a second
// paste of the same playlist.
func TestTheFormClearsOnceAPlaylistOpens(t *testing.T) {
	a := &app{art: newArtwork(), current: -1, page: pageHome, store: openStore(t)}
	a.link = "https://music.youtube.com/playlist?list=PLabc"

	a.showPlaylist("https://music.youtube.com/playlist?list=PLabc", samplePlaylist())

	if a.link != "" {
		t.Errorf("the field still holds %q after the playlist opened", a.link)
	}
	if a.openLink != "https://music.youtube.com/playlist?list=PLabc" {
		t.Errorf("the open link is %q, not the one that was read", a.openLink)
	}
	if a.loading {
		t.Error("the page still says it is reading after the playlist opened")
	}
}

// TestAFailedLoadKeepsTheField checks the other half: a link that would not open
// stays in the field, so the user can fix a typo rather than paste it again.
func TestAFailedLoadKeepsTheField(t *testing.T) {
	a := &app{art: newArtwork(), current: -1, page: pageHome, store: openStore(t)}
	a.link = "https://music.youtube.com/playlist?list=broken"
	a.status = "That playlist would not open"

	tester := ui.NewTester(a.frame, 1280, 840)
	tester.Frame()

	if a.link == "" {
		t.Error("a failed load cleared the field, so the link is gone")
	}
}
