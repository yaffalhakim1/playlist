package src

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"

	"youtube-playlist/internal/library"
	"youtube-playlist/internal/music"
)

// These tests check what the redesign changed in the app's behaviour rather than
// in its look: the one appearance setting, and what a page says when there is
// nothing to show.

// TestAppearanceIsRemembered checks that the choice survives the run: the store
// is what makes it a setting rather than a mood.
func TestAppearanceIsRemembered(t *testing.T) {
	store := openStore(t)
	a := &app{art: newArtwork(), store: store}
	a.setMode("Dark")

	mode, _, _, _ := store.Appearance()
	if mode != "dark" {
		t.Errorf("the appearance came back as %q, not dark", mode)
	}
}

// TestTheThemeIsPaperAndInk checks the one thing the design insists on: the
// window has no accent hue. The accent is the near-black of a filled button, so
// a list of covers is the only colour on the page.
func TestTheThemeIsPaperAndInk(t *testing.T) {
	a := &app{art: newArtwork()}
	theme := a.themeInFrame(t)

	if theme.Background != ui.Hex(paperPage) {
		t.Errorf("the page is not the paper: %v", theme.Background)
	}
	if theme.Surface != ui.Hex(paperCard) {
		t.Errorf("a card is not white: %v", theme.Surface)
	}
	if theme.Text != ui.Hex(paperInk) {
		t.Errorf("the ink is not the design's: %v", theme.Text)
	}
	// The accent is a near-black, not a hue: its channels are within a few
	// points of each other.
	if spread := channelSpread(theme.Accent); spread > 12 {
		t.Errorf("the accent has a hue (channel spread %d): the window is paper and ink", spread)
	}
}

// TestTheDarkAppearanceIsTheSameScheme checks that the dark mode is the paper
// turned over, not a second design.
func TestTheDarkAppearanceIsTheSameScheme(t *testing.T) {
	a := &app{art: newArtwork()}
	var theme *ui.Theme
	tester := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(a.theme(c))
		theme = c.Theme()
	}, 40, 40)
	tester.SetDark(true)
	tester.Frame()

	if !theme.Dark {
		t.Fatal("the dark appearance did not reach the theme")
	}
	if theme.Background != ui.Hex(inkPage) {
		t.Errorf("the dark page is not the design's: %v", theme.Background)
	}
	if theme.Text != ui.Hex(inkText) {
		t.Errorf("the dark ink is not the design's: %v", theme.Text)
	}
}

// channelSpread is how far apart a colour's channels are: 0 is a grey, and a
// large spread is a hue.
func channelSpread(c ui.Color) int {
	lo, hi := 255, 0
	for _, v := range []int{int(c.R), int(c.G), int(c.B)} {
		if v < lo {
			lo = v
		}
		if v > hi {
			hi = v
		}
	}
	return hi - lo
}

// themeInFrame returns the theme the app installs at a frame.
func (a *app) themeInFrame(t *testing.T) *ui.Theme {
	t.Helper()
	var theme *ui.Theme
	tester := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(a.theme(c))
		theme = c.Theme()
	}, 40, 40)
	tester.Frame()
	if theme == nil {
		t.Fatal("the frame installed no theme")
	}
	return theme
}

// TestEveryPageHasAnEmptyState checks that a page with nothing in it still says
// what it is and what would fill it: an empty window with one line of text in a
// corner reads as broken.
func TestEveryPageHasAnEmptyState(t *testing.T) {
	cases := []struct {
		page string
		want string
	}{
		{pageQueue, "The queue is empty"},
		{pagePlayer, "Nothing playing"},
	}
	for _, tc := range cases {
		t.Run(tc.page, func(t *testing.T) {
			a := &app{art: newArtwork(), current: -1, page: tc.page}
			tester := ui.NewTester(a.frame, 1280, 840)
			tester.Frame()
			if !tester.HasText(tc.want) {
				t.Errorf("the page does not say %q: %v", tc.want, tester.Texts())
			}
		})
	}
}

// TestEmptyStatsSaysWhatWouldFillIt checks that a chart with no plays says so
// rather than drawing an axis with no bars on it, which reads as a broken chart.
func TestEmptyStatsSaysWhatWouldFillIt(t *testing.T) {
	a := &app{art: newArtwork(), current: -1, page: pageStats}
	tester := ui.NewTester(a.frame, 1280, 840)
	tester.Frame()
	if !tester.HasText("Nothing in this period.") {
		t.Errorf("the empty chart does not say what would fill it: %v", tester.Texts())
	}
}

// TestEmptyPlayerLeavesNoDeadControls checks that the player's transport is
// unavailable with no queue rather than looking live and doing nothing.
func TestEmptyPlayerLeavesNoDeadControls(t *testing.T) {
	a := &app{art: newArtwork(), current: -1, page: pageHome}
	tester := ui.NewTester(a.frame, 1280, 840)
	tester.Frame()
	if !tester.HasText("Nothing playing") {
		t.Errorf("the rail does not say that nothing plays: %v", tester.Texts())
	}
}

// TestHomeShowsThePlaylistThatIsOpen checks the hero: the page leads with the
// playlist the app has loaded, not with a name for the page.
func TestHomeShowsThePlaylistThatIsOpen(t *testing.T) {
	a := loaded(pageHome)
	tester := ui.NewTester(a.frame, 1280, 840)
	tester.Frame()

	if !tester.HasText("Mix - Bruno Mars - It Will Rain") {
		t.Errorf("the page does not name the playlist: %v", tester.Texts())
	}
	if !tester.HasText("It Will Rain") {
		t.Errorf("the page does not list the playlist's tracks: %v", tester.Texts())
	}
	if !tester.HasText("Now playing from") {
		t.Errorf("the page does not say what the hero is: %v", tester.Texts())
	}
}

// TestHomePointsAtTheQueueForTheRest checks that the table does not try to show
// a thousand rows: it shows a handful and says where the rest are.
func TestHomePointsAtTheQueueForTheRest(t *testing.T) {
	a := loaded(pageHome)
	a.queue = a.queue[:0]
	for i := range 40 {
		a.queue = append(a.queue, music.Track{ID: fmt.Sprintf("t%d", i), Title: fmt.Sprintf("Track %d", i)})
	}
	a.visible = allTracks(a.queue)
	tester := ui.NewTester(a.frame, 1280, 840)
	tester.Frame()

	if !tester.HasText("All 40 tracks") {
		t.Errorf("the table does not point at the queue for the rest: %v", tester.Texts())
	}
	if err := tester.Click("All 40 tracks"); err != nil {
		t.Fatalf("open the queue: %v", err)
	}
	tester.Frame()
	if a.page != pageQueue {
		t.Errorf("the action left the page at %q, not on the queue", a.page)
	}
}

// TestPickingAPlaylistLoadsIt checks that a playlist of the shelf is played from
// its card, which is what the shelf is for.
func TestPickingAPlaylistLoadsIt(t *testing.T) {
	a := loaded(pageHome)
	a.link = ""
	a.recents = []library.Recent{
		{Link: "https://music.youtube.com/playlist?list=PLabc", Title: "Mix - Bruno Mars"},
	}
	tester := ui.NewTester(a.frame, 1280, 840)
	tester.Frame()

	if err := tester.Click("Mix - Bruno Mars"); err != nil {
		t.Fatalf("click the recent playlist: %v", err)
	}
	if a.link != "https://music.youtube.com/playlist?list=PLabc" {
		t.Errorf("the link was not taken from the card: %q", a.link)
	}
}

// TestTotalLengthAddsUp checks the length the playlist's head reports.
func TestTotalLengthAddsUp(t *testing.T) {
	tracks := []music.Track{
		{Duration: 3 * 60 * 1e9},
		{Duration: 2 * 60 * 1e9},
	}
	if got, want := totalLength(tracks), "5:00"; got != want {
		t.Errorf("totalLength = %q, want %q", got, want)
	}
	if got := totalLength(nil); got != "unknown length" {
		t.Errorf("totalLength of nothing = %q, want %q", got, "unknown length")
	}
}
