package src

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"youtube-playlist/internal/library"
	"youtube-playlist/internal/music"
)

// Deleting a playlist is the one thing in the app that cannot be undone, so it
// asks first, and it has to clear everything the app remembers about it: the
// sidebar entry, the cached tracks, and the session.

// TestForgetPlaylistRemovesEverything checks the store side: the recent list,
// the cache and the session all let go of the link, and nothing else does.
func TestForgetPlaylistRemovesEverything(t *testing.T) {
	store := openStore(t)
	keep := playlistNamed("keep")
	drop := playlistNamed("drop")
	store.CachePlaylist("link-keep", keep)
	store.CachePlaylist("link-drop", drop)
	store.AddRecent("link-keep", "Keep", "")
	store.AddRecent("link-drop", "Drop", "")
	store.SetSession(library.Session{Link: "link-drop", TrackID: "two"})

	store.ForgetPlaylist("link-drop")

	if _, ok := store.CachedPlaylist("link-drop"); ok {
		t.Error("the deleted playlist is still cached")
	}
	if _, ok := store.CachedPlaylist("link-keep"); !ok {
		t.Error("deleting one playlist dropped another")
	}
	for _, recent := range store.Recents() {
		if recent.Link == "link-drop" {
			t.Error("the deleted playlist is still in the sidebar's list")
		}
	}
	if got := len(store.Recents()); got != 1 {
		t.Errorf("the recent list has %d playlists, not the 1 that was kept", got)
	}
	if session := store.Session(); session.Link != "" {
		t.Errorf("the session still points at the deleted playlist: %+v", session)
	}
}

// TestForgetPlaylistKeepsTheSessionOfAnother checks that deleting a playlist
// that is not the one open leaves the session alone: the app should still
// resume where it was.
func TestForgetPlaylistKeepsTheSessionOfAnother(t *testing.T) {
	store := openStore(t)
	store.SetSession(library.Session{Link: "link-keep", TrackID: "one"})
	store.CachePlaylist("link-drop", playlistNamed("drop"))

	store.ForgetPlaylist("link-drop")

	if session := store.Session(); session.Link != "link-keep" {
		t.Errorf("deleting another playlist cleared the session: %+v", session)
	}
}

// TestForgetPlaylistKeepsThePlays checks that the stats survive: forgetting a
// playlist is not a reason to forget that it was heard.
func TestForgetPlaylistKeepsThePlays(t *testing.T) {
	store := openStore(t)
	store.AddPlay(library.Play{TrackID: "one", Title: "It Will Rain", At: time.Now()})
	store.CachePlaylist("link-drop", playlistNamed("drop"))

	store.ForgetPlaylist("link-drop")

	if got := len(store.Plays()); got != 1 {
		t.Errorf("the plays are %d, not 1: deleting a playlist should not touch the stats", got)
	}
}

// TestDeletingTheOpenPlaylistGoesHome checks what the window does when the
// playlist on screen is deleted: it cannot keep showing it.
func TestDeletingTheOpenPlaylistGoesHome(t *testing.T) {
	a := &app{art: newArtwork(), current: 0, page: pagePlaylist, store: openStore(t)}
	list := samplePlaylist()
	a.link = "link-a"
	a.playlist = &list
	a.queue = list.Tracks
	a.visible = visibleTracks(list.Tracks, "")

	a.deletePlaylist("link-a")

	if a.page != pageHome {
		t.Errorf("the app is on %q, not home", a.page)
	}
	if a.playlist != nil {
		t.Error("the deleted playlist is still on the page")
	}
	if len(a.queue) != 0 {
		t.Errorf("the queue still holds %d tracks", len(a.queue))
	}
	if a.link != "" {
		t.Errorf("the link field still holds %q", a.link)
	}
	if a.current != -1 {
		t.Errorf("the current track is %d, not -1", a.current)
	}
}

// TestDeletingAnotherPlaylistLeavesThePageAlone checks that deleting a playlist
// that is not on screen changes nothing the user is looking at.
func TestDeletingAnotherPlaylistLeavesThePageAlone(t *testing.T) {
	a := &app{art: newArtwork(), current: 0, page: pagePlaylist, store: openStore(t)}
	list := samplePlaylist()
	a.link = "link-open"
	a.playlist = &list
	a.queue = list.Tracks

	a.deletePlaylist("link-other")

	if a.page != pagePlaylist {
		t.Errorf("the app left the page it was on: %q", a.page)
	}
	if a.playlist == nil {
		t.Error("the playlist on screen was cleared by deleting another")
	}
	if a.link != "link-open" {
		t.Errorf("the link field changed to %q", a.link)
	}
}

// TestDeleteAsksFirst checks that nothing is deleted until the question is
// answered: a right click that deleted at once would be a trap.
func TestDeleteAsksFirst(t *testing.T) {
	a := &app{art: newArtwork(), current: -1, page: pageHome, store: openStore(t)}
	a.store.CachePlaylist("link-a", samplePlaylist())
	a.store.AddRecent("link-a", "A playlist", "")
	a.recents = a.store.Recents()

	a.askDelete("link-a", "A playlist")

	if !a.deleting.open {
		t.Fatal("no question was asked")
	}
	if _, ok := a.store.CachedPlaylist("link-a"); !ok {
		t.Error("the playlist was deleted before the question was answered")
	}
}

// TestConfirmDeleteRemovesItWhenAnswered checks the yes: the answer the dialog
// reports as the destructive button deletes the playlist.
func TestConfirmDeleteRemovesItWhenAnswered(t *testing.T) {
	a := &app{art: newArtwork(), current: -1, page: pageHome, store: openStore(t)}
	a.store.CachePlaylist("link-a", samplePlaylist())
	a.store.AddRecent("link-a", "A playlist", "")
	a.recents = a.store.Recents()
	a.askDelete("link-a", "A playlist")

	tester := ui.NewTester(a.frame, 1280, 900)
	tester.Frame()

	// The dialog's buttons, in the order confirmDelete passes them.
	if err := tester.Click("Delete"); err != nil {
		t.Fatalf("the question has no Delete button: %v (%v)", err, tester.Texts())
	}
	tester.Frame()

	if _, ok := a.store.CachedPlaylist("link-a"); ok {
		t.Error("the playlist was not deleted after the question was answered")
	}
	if a.deleting.open {
		t.Error("the question is still open after it was answered")
	}
}

// TestCancelKeepsThePlaylist checks the no: the playlist survives.
func TestCancelKeepsThePlaylist(t *testing.T) {
	a := &app{art: newArtwork(), current: -1, page: pageHome, store: openStore(t)}
	a.store.CachePlaylist("link-a", samplePlaylist())
	a.store.AddRecent("link-a", "A playlist", "")
	a.recents = a.store.Recents()
	a.askDelete("link-a", "A playlist")

	tester := ui.NewTester(a.frame, 1280, 900)
	tester.Frame()

	if err := tester.Click("Cancel"); err != nil {
		t.Fatalf("the question has no Cancel button: %v (%v)", err, tester.Texts())
	}
	tester.Frame()

	if _, ok := a.store.CachedPlaylist("link-a"); !ok {
		t.Error("the playlist was deleted even though the answer was Cancel")
	}
	if a.deleting.open {
		t.Error("the question is still open after it was answered")
	}
}

// TestNoQuestionWhenNothingIsPending checks that the dialog draws nothing when
// there is nothing to ask about, so it cannot be dismissed by accident.
func TestNoQuestionWhenNothingIsPending(t *testing.T) {
	a := &app{art: newArtwork(), current: -1, page: pageHome, store: openStore(t)}
	tester := ui.NewTester(a.frame, 1280, 900)
	tester.Frame()
	if tester.HasText("Delete this playlist?") {
		t.Error("a question was asked with nothing to delete")
	}
}

// playlistNamed is a playlist with one track, named for the test.
func playlistNamed(name string) music.Playlist {
	return music.Playlist{Title: name, Tracks: []music.Track{{ID: name, Title: name}}}
}
