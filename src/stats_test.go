package src

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"youtube-playlist/internal/library"
)

// The stats page is drawn on every frame, so anything it computes has to come
// out the same way every time it is asked. A list that reorders itself between
// frames flickers while the user looks at it, and no amount of care in the
// drawing code fixes a sort that is not a total order.

// TestMostPlayedIsStableAcrossCalls checks that the same plays always give the
// same list, in the same order.
//
// The tracks are played the same number of times on purpose: that is the case
// that used to flicker, because the tie was broken by Go's map iteration, which
// is randomized on every pass.
func TestMostPlayedIsStableAcrossCalls(t *testing.T) {
	plays := []libraryPlay{
		{TrackID: "a", Title: "It Will Rain"},
		{TrackID: "b", Title: "Roar"},
		{TrackID: "c", Title: "Roll Up"},
		{TrackID: "d", Title: "Locked Out of Heaven"},
		{TrackID: "e", Title: "Grenade"},
	}
	// Every track played once, so every one of them ties for the top place.
	first := mostPlayed(plays, 5)
	for range 200 {
		got := mostPlayed(plays, 5)
		if len(got) != len(first) {
			t.Fatalf("the list changed length: %d then %d", len(first), len(got))
		}
		for i := range got {
			if got[i] != first[i] {
				t.Fatalf("the list reordered itself between frames: position %d was %+v, then %+v\nfirst: %+v\nnow:   %+v",
					i, first[i], got[i], first, got)
			}
		}
	}
}

// TestMostPlayedBreaksTiesByTitle checks the tiebreak itself, so the order is
// not merely stable by accident but decided: equal plays come out by name.
func TestMostPlayedBreaksTiesByTitle(t *testing.T) {
	plays := []libraryPlay{
		{TrackID: "z", Title: "Zebra"},
		{TrackID: "a", Title: "Apple"},
		{TrackID: "m", Title: "Mango"},
	}
	got := mostPlayed(plays, 3)
	want := []string{"Apple", "Mango", "Zebra"}
	for i := range want {
		if got[i].title != want[i] {
			t.Fatalf("tied tracks are not in name order: got %v, want %v",
				titlesOf(got), want)
		}
	}
}

// TestMostPlayedPutsTheMostPlayedFirst checks the ranking itself: a track
// played more often comes before one played less, whatever the names are.
func TestMostPlayedPutsTheMostPlayedFirst(t *testing.T) {
	plays := []libraryPlay{
		{TrackID: "a", Title: "Apple"},
		{TrackID: "z", Title: "Zebra"},
		{TrackID: "z", Title: "Zebra"},
		{TrackID: "z", Title: "Zebra"},
		{TrackID: "m", Title: "Mango"},
		{TrackID: "m", Title: "Mango"},
	}
	got := mostPlayed(plays, 3)
	want := []struct {
		title string
		count int
	}{{"Zebra", 3}, {"Mango", 2}, {"Apple", 1}}
	if len(got) != len(want) {
		t.Fatalf("got %d tracks, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].title != want[i].title || got[i].count != want[i].count {
			t.Fatalf("position %d is %+v, want %+v (list: %+v)",
				i, got[i], want[i], got)
		}
	}
}

// TestMostPlayedRespectsTheLimit checks that the list stops at the limit, and
// keeps the tracks that earned their place.
func TestMostPlayedRespectsTheLimit(t *testing.T) {
	var plays []libraryPlay
	for i := range 10 {
		// Track i played i+1 times, so the last one is the most played.
		for range i + 1 {
			plays = append(plays, libraryPlay{TrackID: string(rune('a' + i)), Title: string(rune('A' + i))})
		}
	}
	got := mostPlayed(plays, 3)
	if len(got) != 3 {
		t.Fatalf("the list has %d tracks, not the limit of 3", len(got))
	}
	want := []string{"J", "I", "H"}
	for i := range want {
		if got[i].title != want[i] {
			t.Fatalf("the top three are %v, want %v", titlesOf(got), want)
		}
	}
}

// TestMostPlayedHandlesNoPlays checks the empty case, which the stats page hits
// on a fresh install.
func TestMostPlayedHandlesNoPlays(t *testing.T) {
	if got := mostPlayed(nil, 5); len(got) != 0 {
		t.Errorf("nothing played gives %+v, want nothing", got)
	}
}

// TestMostPlayedIsDrawnInTheSameOrderOnEveryFrame checks the flicker where the
// user saw it: the drawn page, not the function.
//
// Two frames of the same state must place the same name at the same height. A
// list that follows map iteration puts a different track first each frame.
func TestMostPlayedIsDrawnInTheSameOrderOnEveryFrame(t *testing.T) {
	store, err := library.Open(filepath.Join(t.TempDir(), "settings.json"))
	if err != nil {
		t.Fatalf("open the store: %v", err)
	}
	// Four tracks, each played the same number of times: every one ties.
	for _, title := range []string{"Grenade", "Roar", "It Will Rain", "Roll Up"} {
		store.AddPlay(playRecord(title, title))
	}

	a := &app{art: newArtwork(), current: -1, page: pageStats, store: store}
	var first float32
	for frame := range 12 {
		tester := ui.NewTester(a.frame, 1280, 840)
		tester.Frame()
		rect, ok := tester.Find("Grenade")
		if !ok {
			t.Fatalf("frame %d does not show the track: %v", frame, tester.Texts())
		}
		if frame == 0 {
			first = rect.Y
			continue
		}
		if rect.Y != first {
			t.Fatalf("frame %d draws Grenade at y=%.0f, the first frame drew it at y=%.0f: the list reorders itself while the page is looked at",
				frame, rect.Y, first)
		}
	}
}

// playRecord makes a play of a track.
func playRecord(id, title string) libraryPlay {
	return libraryPlay{TrackID: id, Title: title, At: time.Now()}
}

// titlesOf returns the names of a list, for a failure message.
func titlesOf(list []played) []string {
	out := make([]string, len(list))
	for i, item := range list {
		out[i] = item.title
	}
	return out
}
