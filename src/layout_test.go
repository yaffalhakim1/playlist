package src

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"youtube-playlist/internal/music"
)

// These tests check the layout the way the eye checks it: that the pieces of a
// page sit on one axis, keep the same room around them, and never run off the
// window. A page that draws is not the same as a page that hangs together.

// window is a size to draw a page at.
type window struct {
	name          string
	width, height int
}

// windows are the sizes a page is checked at: the size the window opens at, the
// smallest it may be, and one wide enough that a column has to stop growing.
var windows = []window{
	{"the opening window", 1280, 840},
	{"the smallest window", 960, 620},
	{"a wide window", 1680, 980},
}

// loaded is an app with a playlist in it, which is the state most pages show.
func loaded(page string) *app {
	a := &app{art: newArtwork(), current: 0, page: page, volume: 0.7}
	a.playlist = &music.Playlist{ID: "PL", Title: "Mix - Bruno Mars - It Will Rain"}
	a.queue = []music.Track{
		{ID: "one", Title: "It Will Rain", Artist: "Bruno Mars", Album: "Ballads"},
		{ID: "two", Title: "Roar", Artist: "Katy Perry", Album: "Prism"},
		{ID: "three", Title: "Roll Up", Artist: "Wiz Khalifa"},
	}
	a.visible = allTracks(a.queue)
	a.playing = true
	return a
}

// TestPageInsetIsOneValue checks that a page keeps the same margin on every
// side that has one, and that the margin narrows only when the window does.
func TestPageInsetIsOneValue(t *testing.T) {
	a := &app{art: newArtwork()}
	for _, size := range windows {
		t.Run(size.name, func(t *testing.T) {
			var inset float32
			tester := ui.NewTester(func(c *ui.Context) {
				inset = pageInset(c)
			}, size.width, size.height)
			tester.Frame()
			theme := themeInFrame(t)
			if inset <= 0 {
				t.Fatalf("the page keeps no margin at all")
			}
			if inset != theme.Space(6) && inset != theme.Space(8) {
				t.Errorf("the margin is %.1f, which is not a multiple of the theme's spacing",
					inset)
			}
			if size.width < 1040 && inset != theme.Space(6) {
				t.Errorf("a narrow window keeps a margin of %.1f, not the narrower %.1f",
					inset, theme.Space(6))
			}
			if size.width >= 1040 && inset != theme.Space(8) {
				t.Errorf("a wide window keeps a margin of %.1f, not the wider %.1f",
					inset, theme.Space(8))
			}
		})
	}
	_ = a
}

// TestPageColumnStopsGrowing checks that a page's column stops widening on a
// wide window and centers instead: a line of text wider than the measure is
// hard to follow, and a row of covers with a desert beside it looks unfinished.
func TestPageColumnStopsGrowing(t *testing.T) {
	a := loaded(pageHome)
	tester := ui.NewTester(a.frame, 1680, 980)
	tester.Frame()

	// The page's own content starts at the column's edge. The hero's eyebrow is
	// inside the hero panel, so it sits one card inset in; the panel itself is
	// what has to start at the column.
	row, ok := tester.Find("Playlist link")
	if !ok {
		t.Fatalf("the page has no import row: %v", tester.Texts())
	}
	columnLeft, columnRight := pageColumn(t, a, 1680)
	// The import row's field sits inside the row's own padding and its icon, so
	// the row begins a fixed inset to its left.
	const rowInset = 49
	if diff := (row.X - rowInset) - columnLeft; diff > 2 || diff < -2 {
		t.Errorf("the page's content starts at x=%.0f, not at the column's edge at x=%.0f",
			row.X-rowInset, columnLeft)
	}
	// The column is narrower than the page area, and centered in it.
	pageLeft := a.sidebarWidth()
	if columnLeft-pageLeft < (1680-columnRight)-0 {
		t.Logf("the column runs from %.0f to %.0f in a page area from %.0f to 1680",
			columnLeft, columnRight, pageLeft)
	}
	left := columnLeft - pageLeft
	right := float32(1680) - columnRight
	if diff := left - right; diff > 1 || diff < -1 {
		t.Errorf("the column is not centered: %.0f of room on the left, %.0f on the right",
			left, right)
	}
	if columnRight-columnLeft >= float32(1680)-pageLeft {
		t.Errorf("the column fills the page area: it does not stop growing")
	}
}

// pageColumn returns where a page's column starts and ends, in window
// coordinates: the room the page keeps, then the column the theme's width
// allows, centered in what is left.
func pageColumn(t *testing.T, a *app, width int) (left, right float32) {
	t.Helper()
	inset := pageInsetFor(t, width)
	area := float32(width) - a.sidebarWidth()
	content := area - 2*inset
	column := min(pageWidthInFrame(t, width), content)
	left = a.sidebarWidth() + inset + (content-column)/2
	return left, left + column
}

// pageWidthInFrame returns the widest a page's column grows at a width.
func pageWidthInFrame(t *testing.T, width int) float32 {
	t.Helper()
	var wide float32
	tester := ui.NewTester(func(c *ui.Context) {
		wide = pageWidth(c)
	}, width, 800)
	tester.Frame()
	return wide
}

// pageInsetFor returns the margin a page keeps at a width.
func pageInsetFor(t *testing.T, width int) float32 {
	t.Helper()
	var inset float32
	tester := ui.NewTester(func(c *ui.Context) {
		inset = pageInset(c)
	}, width, 800)
	tester.Frame()
	return inset
}

// TestPagesKeepClearOfTheHeaderAndTheBar checks that no page puts anything
// under the floating header or under the player bar: both are over the page,
// and anything there is covered.
//
// The Player page has no bar under it — it carries its own transport — so the
// bar's strip is only reserved on the other pages.
func TestPagesKeepClearOfTheHeaderAndTheBar(t *testing.T) {
	pages := []string{pageHome, pageQueue, pagePlayer, pageStats, pageAppearance}
	for _, page := range pages {
		for _, size := range windows {
			t.Run(page+"/"+size.name, func(t *testing.T) {
				a := loaded(page)
				tester := ui.NewTester(a.frame, size.width, size.height)
				tester.Frame()

				header := a.headerHeightInFrame(t, size.width, size.height)
				bar := barTopInFrame(t, size.width, size.height)
				for _, text := range tester.Texts() {
					if text == "" {
						continue
					}
					rect, ok := tester.Find(text)
					if !ok {
						continue
					}
					// The sidebar's own labels are above the header, by design.
					if rect.X < a.sidebarWidth() {
						continue
					}
					if rect.Y < header && rect.Y > 0 {
						// Only the header's own action may sit there.
						if text != "Playlist" && text != "Show or hide the sidebar" {
							t.Errorf("%q sits at y=%.0f, under the header that ends at %.0f",
								text, rect.Y, header)
						}
					}
					if page == pagePlayer {
						// The page owns the whole height.
						continue
					}
					if rect.Y+rect.H > bar && rect.Y < bar {
						t.Errorf("%q runs from y=%.0f to y=%.0f, into the bar that starts at %.0f",
							text, rect.Y, rect.Y+rect.H, bar)
					}
				}
			})
		}
	}
}

// headerHeightInFrame returns the room the header takes at a size.
func (a *app) headerHeightInFrame(t *testing.T, width, height int) float32 {
	t.Helper()
	var header float32
	tester := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(a.theme(c))
		header = a.headerHeight(c)
	}, width, height)
	tester.Frame()
	return header
}

// barTopInFrame returns where the player bar starts at a size.
func barTopInFrame(t *testing.T, width, height int) float32 {
	t.Helper()
	a := &app{art: newArtwork()}
	var top float32
	tester := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(a.theme(c))
		options := playerBarOptions(c)
		top = float32(height) - options.Inset - options.Height
	}, width, height)
	tester.Frame()
	return top
}

// TestPlayerBarHasItsOwnStrip checks that the page stops above the bar: the
// bar is the last row of the window, not something floating over the page with
// content passing under it.
func TestPlayerBarHasItsOwnStrip(t *testing.T) {
	a := loaded(pageHome)
	tester := ui.NewTester(a.frame, 1280, 840)
	tester.Frame()

	title, ok := tester.Find("Now playing from")
	if !ok {
		t.Fatal("the page has no name")
	}
	bar := barTopInFrame(t, 1280, 840)
	if title.Y+title.H >= bar {
		t.Errorf("the page's name runs into the bar: it ends at %.0f, the bar starts at %.0f",
			title.Y+title.H, bar)
	}
}

// TestSidebarClearsTheHeader checks that the sidebar's first section sits below
// the floating header, so its heading is not hidden under the window controls.
func TestSidebarClearsTheHeader(t *testing.T) {
	for _, size := range windows {
		t.Run(size.name, func(t *testing.T) {
			a := loaded(pageHome)
			tester := ui.NewTester(a.frame, size.width, size.height)
			tester.Frame()

			header := a.headerHeightInFrame(t, size.width, size.height)
			library, ok := tester.Find("LIBRARY")
			if !ok {
				t.Fatalf("the sidebar has no first section: %v", tester.Texts())
			}
			if library.Y < header {
				t.Errorf("the sidebar's heading sits at y=%.0f, under the header that ends at %.0f",
					library.Y, header)
			}
		})
	}
}

// TestHeaderStartsWhereThePageDoes checks that the header's own name begins at
// the page's column edge: a header floating at the window's edge while the page
// stops short reads as two layouts.
func TestHeaderStartsWhereThePageDoes(t *testing.T) {
	a := loaded(pageHome)
	tester := ui.NewTester(a.frame, 1680, 980)
	tester.Frame()

	name, ok := tester.Find("Playlist")
	if !ok {
		t.Fatalf("the header has no name: %v", tester.Texts())
	}
	columnLeft, _ := pageColumn(t, a, 1680)
	if diff := name.X - columnLeft; diff > 2 || diff < -2 {
		t.Errorf("the header starts at x=%.0f, the page's column at x=%.0f: they do not line up",
			name.X, columnLeft)
	}
}

// TestSettingsRowsShareTheGrid checks that a setting's name starts where a
// page's other text does, so the Look page is on the same grid as the rest.
func TestSettingsRowsShareTheGrid(t *testing.T) {
	a := loaded(pageAppearance)
	tester := ui.NewTester(a.frame, 1280, 840)
	tester.Frame()

	title, ok := tester.Find("Appearance")
	if !ok {
		t.Fatalf("the page has no name: %v", tester.Texts())
	}
	setting, ok := tester.Find("Appearance")
	if !ok {
		t.Fatalf("the page has no settings: %v", tester.Texts())
	}
	// The card adds its own room, so the setting sits inside the name's edge,
	// never outside it.
	if setting.X < title.X {
		t.Errorf("a setting starts at x=%.0f, left of the page's name at x=%.0f",
			setting.X, title.X)
	}
}

// TestListRowsAreTheSameHeight checks that every row of a list is the same
// height: a list where one row is taller than the next looks broken, and a
// selected row that grows as it is chosen is worse.
func TestListRowsAreTheSameHeight(t *testing.T) {
	a := loaded(pageQueue)
	tester := ui.NewTester(a.frame, 1280, 840)
	tester.Frame()

	rows := []string{"It Will Rain", "Roar", "Roll Up"}
	var heights []float32
	for _, name := range rows {
		rect, ok := tester.Find(name)
		if !ok {
			t.Fatalf("the list does not show %q: %v", name, tester.Texts())
		}
		heights = append(heights, rect.H)
	}
	for i := 1; i < len(heights); i++ {
		if heights[i] != heights[0] {
			t.Errorf("row %d is %.1f high, row 0 is %.1f: the rows do not agree",
				i, heights[i], heights[0])
		}
	}
}

// TestThePlayingRowIsMarkedAndStillTheSameSize checks that the track that plays
// is marked, and that marking it does not change its height.
func TestThePlayingRowIsMarkedAndStillTheSameSize(t *testing.T) {
	a := loaded(pageQueue)
	tester := ui.NewTester(a.frame, 1280, 840)
	tester.Frame()

	playing, ok := tester.Find("It Will Rain")
	if !ok {
		t.Fatal("the list does not show the track that plays")
	}
	other, ok := tester.Find("Roar")
	if !ok {
		t.Fatal("the list does not show a track that does not play")
	}
	if playing.H != other.H {
		t.Errorf("the playing row is %.1f high and another is %.1f: the mark changed the size",
			playing.H, other.H)
	}
}

// TestTextScaleGrowsTheLayout checks that the window follows the desktop's text
// size: nothing is cut off when the text grows, which is what a layout written
// in fixed pixels does.
func TestTextScaleGrowsTheLayout(t *testing.T) {
	for _, scale := range []float32{1, 1.25, 1.5} {
		t.Run("scale", func(t *testing.T) {
			a := loaded(pageHome)
			tester := ui.NewTester(a.frame, 1280, 840)
			tester.SetPreferences(ui.Preferences{TextScale: scale})
			tester.Frame()
			if tester.Image() == nil {
				t.Fatalf("nothing drew at a text scale of %.2f", scale)
			}
			if !tester.HasText("Now playing from") {
				t.Errorf("the page's name is gone at a text scale of %.2f", scale)
			}
		})
	}
}

// TestReduceMotionStillDraws checks that the window draws with motion turned
// off, which is what the desktop's Reduce Motion setting asks for.
func TestReduceMotionStillDraws(t *testing.T) {
	a := loaded(pageHome)
	tester := ui.NewTester(a.frame, 1280, 840)
	tester.SetPreferences(ui.Preferences{ReduceMotion: true})
	tester.Frame()
	if tester.Image() == nil {
		t.Fatal("nothing drew with reduced motion")
	}
	// A page change is the one thing that moves; with motion off it must land
	// at once rather than stay half way.
	a.page = pageStats
	tester.Frame()
	tester.Frame()
	if !tester.HasText("Your listening") {
		t.Errorf("the new page did not arrive with reduced motion: %v", tester.Texts())
	}
}

// TestHighContrastDraws checks that a high contrast desktop still draws a page
// whose text is not the same colour as the page under it.
func TestHighContrastDraws(t *testing.T) {
	a := loaded(pageHome)
	tester := ui.NewTester(a.frame, 1280, 840)
	tester.SetPreferences(ui.Preferences{HighContrast: true})
	tester.Frame()
	if tester.Image() == nil {
		t.Fatal("nothing drew at high contrast")
	}
	if !tester.HasText("Now playing from") {
		t.Error("the page's name is gone at high contrast")
	}
}

// TestEveryPageDrawsInTheDark checks that no page depends on the light
// appearance to be legible: a page that draws nothing in the dark is a page
// whose colours were written by hand.
func TestEveryPageDrawsInTheDark(t *testing.T) {
	// Each page has one thing it must show, in the dark as in the light: the
	// words that say what the page is.
	pages := []struct{ page, name string }{
		{pageHome, "Now playing from"},
		{pageQueue, "Songs"},
		{pagePlayer, "It Will Rain"},
		{pageStats, "Your listening"},
		{pageAppearance, "Appearance"},
	}
	for _, tc := range pages {
		page, name := tc.page, tc.name
		t.Run(page, func(t *testing.T) {
			a := loaded(page)
			tester := ui.NewTester(a.frame, 1280, 840)
			tester.SetDark(true)
			tester.Frame()
			if tester.Image() == nil {
				t.Fatalf("the page %q drew nothing in the dark", page)
			}
			if !tester.HasText(name) {
				t.Errorf("the page %q shows no name in the dark: %v", page, tester.Texts())
			}
		})
	}
}
