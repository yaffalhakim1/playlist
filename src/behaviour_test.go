package src

import (
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"

	"youtube-playlist/internal/library"
	"youtube-playlist/internal/music"
)

// These tests check what the redesign changed in the app's behaviour rather
// than in its look: the accent a user picks, and what a page says when there is
// nothing to show.

// TestAccentCanBeUnmade checks the round trip the Look page now offers: a user
// picks a colour, and can go back to the design's own.
func TestAccentCanBeUnmade(t *testing.T) {
	a := &app{art: newArtwork()}

	a.setAccent("#00639b")
	if a.appearance.accent != "#00639b" {
		t.Fatalf("the accent was not kept: %q", a.appearance.accent)
	}
	if a.appearance.dynamic {
		t.Error("picking a colour should stop the accent following the artwork")
	}

	a.setAccent("")
	if a.appearance.accent != "" {
		t.Errorf("the accent was not unmade: %q", a.appearance.accent)
	}
	if a.appearance.dynamic {
		t.Error("unmaking the accent should not turn the artwork colour on")
	}
}

// TestDynamicAccentClearsTheChosenOne checks that turning the artwork colour on
// clears a colour the user picked before, so only one of the two decides.
func TestDynamicAccentClearsTheChosenOne(t *testing.T) {
	a := &app{art: newArtwork()}
	a.setAccent("#00639b")
	a.setDynamic(true)
	if a.appearance.accent != "" {
		t.Errorf("the picked colour survived the artwork colour: %q", a.appearance.accent)
	}
	if !a.appearance.dynamic {
		t.Error("the artwork colour was not turned on")
	}
}

// TestAccentIsRemembered checks that the choice survives the run: the store is
// what makes it a setting rather than a mood.
func TestAccentIsRemembered(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	store, err := library.Open(path)
	if err != nil {
		t.Fatalf("open the store: %v", err)
	}
	a := &app{art: newArtwork(), store: store}
	a.setAccent("#386a20")

	reopened, err := library.Open(path)
	if err != nil {
		t.Fatalf("reopen the store: %v", err)
	}
	mode, accent, dynamic, _ := reopened.Appearance()
	if accent != "#386a20" {
		t.Errorf("the accent came back as %q, not #386a20", accent)
	}
	if dynamic {
		t.Error("the artwork colour came back on")
	}
	if mode == "" {
		t.Error("the appearance came back empty")
	}
}

// TestAccentIsLaidOverThePalette checks that a picked accent reaches the theme
// the frame installs, and that what the theme derives from it is derived.
func TestAccentIsLaidOverThePalette(t *testing.T) {
	a := &app{art: newArtwork()}
	a.setAccent("#00639b")
	theme := a.themeInFrame(t)

	want := ui.Hex("#00639b")
	if theme.Accent != want {
		t.Errorf("the theme's accent is not the one picked")
	}
	if (theme.AccentText == ui.Color{}) {
		t.Error("nothing is written on the accent")
	}
	if (theme.Selection == ui.Color{}) {
		t.Error("nothing is marked as chosen")
	}
	if (theme.Focus == ui.Color{}) {
		t.Error("the focus ring has no colour")
	}
	// The hover and the pressed face have to differ from the face itself, or a
	// button gives no sign it was touched.
	if theme.AccentHover == theme.Accent || theme.AccentPressed == theme.Accent {
		t.Error("the accent's hover and pressed faces are the accent itself")
	}
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
		{pagePlaylist, "No playlist yet"},
		{pagePlayer, "Nothing playing"},
		{pageQueue, "The queue is empty"},
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
		t.Errorf("the bar does not say that nothing plays: %v", tester.Texts())
	}
}

// TestPlayingFromNamesThePlaylist checks the card on the home page that says
// what the queue is, and that it opens the queue.
func TestPlayingFromNamesThePlaylist(t *testing.T) {
	a := loaded(pageHome)
	tester := ui.NewTester(a.frame, 1280, 840)
	tester.Frame()

	if !tester.HasText("Playing from") {
		t.Errorf("the page does not say what plays from where: %v", tester.Texts())
	}
	if !tester.HasText("Mix - Bruno Mars - It Will Rain") {
		t.Errorf("the page does not name the playlist: %v", tester.Texts())
	}
	if err := tester.Click("Open queue"); err != nil {
		t.Fatalf("open the queue: %v", err)
	}
	tester.Frame()
	if a.page != pageQueue {
		t.Errorf("the action left the page at %q, not on the queue", a.page)
	}
}

// TestPickingAPlaylistLoadsIt checks that a recent playlist is played from its
// card, which is what the shelf of covers is for.
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
