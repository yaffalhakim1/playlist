package src

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"youtube-playlist/internal/music"
)

func TestVisibleTracks(t *testing.T) {
	queue := []music.Track{
		{Title: "Bruno Mars - It Will Rain", Artist: "Bruno Mars", Album: "Ballads"},
		{Title: "Katy Perry - Roar", Artist: "Katy Perry"},
		{Title: "Wiz Khalifa - Roll Up", Artist: "Wiz Khalifa"},
	}
	cases := []struct {
		filter string
		want   []int
	}{
		{filter: "", want: []int{0, 1, 2}},
		{filter: "bruno", want: []int{0}},
		{filter: "KATY", want: []int{1}},
		{filter: "ballads", want: []int{0}},
		{filter: "nothing here", want: nil},
	}
	for _, tc := range cases {
		got := visibleTracks(queue, tc.filter)
		if len(got) != len(tc.want) {
			t.Errorf("visibleTracks(%q) = %v, want %v", tc.filter, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("visibleTracks(%q) = %v, want %v", tc.filter, got, tc.want)
				break
			}
		}
	}
}

func TestClock(t *testing.T) {
	cases := []struct {
		in   time.Duration
		want string
	}{
		{in: 0, want: "0:00"},
		{in: 42 * time.Second, want: "0:42"},
		{in: 3*time.Minute + 42*time.Second, want: "3:42"},
		{in: time.Hour + 2*time.Minute + 3*time.Second, want: "1:02:03"},
	}
	for _, tc := range cases {
		if got := clock(tc.in); got != tc.want {
			t.Errorf("clock(%s) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestFrameDrawsTheQueue draws the window with a playlist, which shows that
// the view holds together.
func TestFrameDrawsTheQueue(t *testing.T) {
	a := &app{art: newArtwork(), current: -1, page: pagePlaylist}
	a.playlist = &music.Playlist{ID: "PL", Title: "A playlist"}
	a.queue = []music.Track{
		{ID: "one", Title: "It Will Rain", Artist: "Bruno Mars", Duration: 3*time.Minute + 42*time.Second},
		{ID: "two", Title: "Roar", Artist: "Katy Perry", Duration: 3 * time.Minute},
	}
	a.visible = visibleTracks(a.queue, "")

	tester := ui.NewTester(a.frame, 1240, 820)
	tester.Frame()

	for _, want := range []string{"It Will Rain", "Bruno Mars", "3:42"} {
		if !tester.HasText(want) {
			t.Errorf("the list does not show %q: %v", want, tester.Texts())
		}
	}
}

// TestFrameDrawsTheHomePage draws the window with nothing to play.
func TestFrameDrawsTheHomePage(t *testing.T) {
	a := &app{art: newArtwork(), current: -1, page: pageHome}
	tester := ui.NewTester(a.frame, 1240, 820)
	tester.Frame()

	if !tester.HasText("Playlist link") {
		t.Errorf("the home page does not ask for a link: %v", tester.Texts())
	}
	for _, item := range destinations {
		if !tester.HasText(item.label) {
			t.Errorf("the sidebar does not show %q: %v", item.label, tester.Texts())
		}
	}
}
