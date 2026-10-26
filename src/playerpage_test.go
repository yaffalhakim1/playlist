package src

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// The footer is the one place the app says what the playback is doing: a track
// being found, a track that failed, a track that plays. A page without it is a
// page that cannot say a track is loading.
//
// It was once left off the Player page, on the argument that the page already had
// a transport. That was wrong twice: the page's transport cannot say what the
// footer says, and a page that hides the only status line is a page that looks
// frozen when a track takes a moment.

// TestEveryPageCarriesTheFooter checks the rule: no page hides the bar.
func TestEveryPageCarriesTheFooter(t *testing.T) {
	for _, page := range []string{pageHome, pageQueue, pagePlayer, pageStats, pageAppearance} {
		t.Run(page, func(t *testing.T) {
			a := loaded(page)
			tester := ui.NewTester(a.frame, 1280, 840)
			tester.Frame()

			// The volume knob is the footer's own, and no page draws another.
			if !tester.HasText("Volume") {
				t.Errorf("the page %q has no footer: %v", page, tester.Texts())
			}
		})
	}
}

// TestThePlayerPageStillSaysWhenATrackFails checks the case the footer exists for:
// the page has its own transport, so a track that failed has to be said by the
// footer or by nothing.
func TestThePlayerPageStillSaysWhenATrackFails(t *testing.T) {
	a := loaded(pagePlayer)
	a.playing = false
	a.status = "yt-dlp found no audio stream"
	tester := ui.NewTester(a.frame, 1280, 840)
	tester.Frame()

	if !tester.HasText("That track will not play") {
		t.Errorf("the player page does not say the track failed: %v", tester.Texts())
	}
}

// TestThePlayerPageNamesTheTrack checks the two things the page is for: the track
// that plays, and what plays after it.
func TestThePlayerPageNamesTheTrack(t *testing.T) {
	a := loaded(pagePlayer)
	tester := ui.NewTester(a.frame, 1280, 840)
	tester.Frame()

	for _, want := range []string{"It Will Rain", "Up next", "Playlist"} {
		if !tester.HasText(want) {
			t.Errorf("the player page does not show %q: %v", want, tester.Texts())
		}
	}
}
