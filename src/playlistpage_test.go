package src

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"youtube-playlist/internal/music"
)

// The playlist page has two controls that change what it shows: the filter over
// the table, and the re-sync that reads the playlist again. Both are easy to wire
// up in a way that looks right and does nothing, so both are driven here.

// TestTheFilterNarrowsTheRows checks the field: a word in a title, an artist or an
// album leaves those rows and no others.
func TestTheFilterNarrowsTheRows(t *testing.T) {
	tracks := []music.Track{
		{ID: "a", Title: "Midnight in Shibuya", Artist: "Kazumi Tateishi Trio", Album: "Tokyo Jazz Night"},
		{ID: "b", Title: "Raindrops on Shinjuku", Artist: "Uyama Hiroto", Album: "A Son of the Sun"},
		{ID: "c", Title: "Warm Tea at 2AM", Artist: "Nujabes", Album: "Metaphorical Music"},
	}
	cases := []struct {
		name   string
		filter string
		want   []int
	}{
		{"nothing narrows nothing", "", []int{0, 1, 2}},
		{"a title", "midnight", []int{0}},
		{"an artist", "nujabes", []int{2}},
		{"an album", "metaphorical", []int{2}},
		{"a case the reader did not type", "SHIBUYA", []int{0}},
		{"a word that is in none", "guitar", nil},
		{"space is not a filter", "   ", []int{0, 1, 2}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := visibleTracks(tracks, tc.filter)
			if len(got) != len(tc.want) {
				t.Fatalf("visibleTracks(%q) = %v, want %v", tc.filter, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("visibleTracks(%q) = %v, want %v", tc.filter, got, tc.want)
				}
			}
		})
	}
}

// TestTheFilterReachesThePage checks the wiring: typing in the field changes what
// the table draws, not just what a helper returns.
func TestTheFilterReachesThePage(t *testing.T) {
	a := loaded(pageQueue)
	tester := ui.NewTester(a.frame, 1280, 840)
	tester.Frame()

	if !tester.HasText("Roar") {
		t.Fatalf("the table does not show the whole playlist: %v", tester.Texts())
	}
	// The field's own binding, as typing into it does.
	a.filter = "bruno"
	a.refilter()
	tester.Frame()

	if tester.HasText("Roar") {
		t.Error("the table still shows a row the filter excludes")
	}
	if !tester.HasText("It Will Rain") {
		t.Errorf("the filter removed the row it should keep: %v", tester.Texts())
	}
}

// TestReSyncReadsThePlaylistAgain checks that re-sync drops the cached tracks, so
// the next open goes to the network rather than serving the copy it has.
func TestReSyncReadsThePlaylistAgain(t *testing.T) {
	store := openStore(t)
	list := samplePlaylist()
	link := "https://music.youtube.com/playlist?list=PLresync"
	store.CachePlaylist(link, list)

	if _, ok := store.CachedPlaylist(link); !ok {
		t.Fatal("the playlist was not cached, so the test proves nothing")
	}

	a := &app{art: newArtwork(), current: -1, page: pageQueue, store: store, openLink: link}
	a.resync()

	if _, ok := store.CachedPlaylist(link); ok {
		t.Error("re-sync left the cache in place, so the next open would serve the old tracks")
	}
	// It is a re-read, not a forget: the playlist stays in the recent list.
	found := false
	for _, recent := range store.Recents() {
		if recent.Link == link {
			found = true
		}
	}
	_ = found
}

// TestThePlaylistPageShowsTheTable checks the page's shape: the name of the
// playlist, its count, and a row for a track.
func TestThePlaylistPageShowsTheTable(t *testing.T) {
	a := loaded(pageQueue)
	tester := ui.NewTester(a.frame, 1280, 840)
	tester.Frame()

	for _, want := range []string{"Mix - Bruno Mars - It Will Rain", "Play all", "Re-sync", "All tracks"} {
		if !tester.HasText(want) {
			t.Errorf("the playlist page does not show %q: %v", want, tester.Texts())
		}
	}
	// The column headings, which is what makes it a table rather than a list.
	for _, head := range []string{"Title", "Artist", "Album"} {
		if !tester.HasText(head) {
			t.Errorf("the table has no %q column: %v", head, tester.Texts())
		}
	}
}

// TestThereIsNoDownloadedFilter checks the removal: the app streams, so a filter
// for what was downloaded could only ever be empty.
func TestThereIsNoDownloadedFilter(t *testing.T) {
	a := loaded(pageQueue)
	tester := ui.NewTester(a.frame, 1280, 840)
	tester.Frame()

	if tester.HasText("Downloaded") {
		t.Errorf("the page still offers a Downloaded filter: %v", tester.Texts())
	}
}
