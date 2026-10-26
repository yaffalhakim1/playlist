package src

import (
	"github.com/egoist/mygo/ui"
)

// content draws the page, with the player bar under it.
//
// The page keeps clear of the bar, so nothing rests behind it and nothing scrolls
// under it: the bar is the last row of the window, not a thing floating over the
// page. A page that replaces another arrives from a little below and fades in,
// keyed by which page it is, so the change reads as one surface replacing another
// rather than as a flicker.
//
// Every page carries the bar, the player's included: it is the one place the app
// says what the playback is doing, so a page without it would be a page that
// cannot say a track is loading or broken.
func (a *app) content(c *ui.Context) {
	ui.Box(c).Fill().Clip().Children(func() {
		ui.Box(c).Key(a.page).Fill().Transition(motionReveal).Children(func() {
			a.pageContent(c)
		})
		a.playerbar(c)
	})
}

// pageContent draws the page that shows.
func (a *app) pageContent(c *ui.Context) {
	switch a.page {
	case pagePlayer:
		a.playerPage(c)
	case pageQueue:
		a.queuePage(c)
	case pageStats:
		a.statsPage(c)
	case pageAppearance:
		a.appearancePage(c)
	default:
		a.homePage(c)
	}
}

// pageFrame gives a page the room it lives in: the same margins on every page,
// clear of the header above and of the bar below, and never wider than a line of
// text can be followed.
//
// The bar keeps its own strip at the bottom of the window, outside the page, so
// nothing the page holds ever passes under it.
//
// A page that scrolls puts its content in one scroll view; a page that holds a
// list gives the list the height that is left, so there is never a scroll bar
// inside a scroll bar.
func (a *app) pageFrame(c *ui.Context, scrolling bool, build func()) {
	t := c.Theme()
	top := a.headerHeight(c) + t.Space(4)
	side := pageInset(c)
	bottom := playerBarHeight(c) + t.Space(2)
	column := func() {
		ui.Column(c).FillWidth().MaxWidth(pageWidth(c)).AlignSelf(ui.Center).Children(build)
	}
	if scrolling {
		ui.Box(c).Fill().Padding(top, side, bottom, side).Children(func() {
			ui.Scroll(c).Fill().ScrollbarInsets(t.Space(2)).Children(column)
		})
		return
	}
	ui.Box(c).Fill().Padding(top, side, bottom, side).Children(func() {
		ui.Column(c).Fill().MaxWidth(pageWidth(c)).AlignSelf(ui.Center).Children(build)
	})
}

// pageWidth is how wide a page grows before it stops: past it a line of text is
// hard to follow and a row of covers leaves a desert of empty page beside it, so
// the column centers in the window instead.
func pageWidth(c *ui.Context) float32 { return c.Theme().Space(238) }

// pageInset is the margin a page keeps from the window, and the one the header's
// own row keeps, so the search sits where the page does. A narrow window keeps
// less of it, so the page still has room for what it holds.
func pageInset(c *ui.Context) float32 {
	t := c.Theme()
	if width, _ := c.Size(); width < 1040 {
		return t.Space(6)
	}
	return t.Space(8)
}

// barOptions is the strip the player bar keeps at the bottom of the window.
type barOptions struct {
	// Height is how tall the bar is, and Inset how far it stands from the
	// window's edge.
	Height float32
	Inset  float32
}

// playerBarOptions describes the strip the bar keeps at the bottom of the page.
func playerBarOptions(c *ui.Context) barOptions {
	t := c.Theme()
	return barOptions{Height: t.Space(20), Inset: 0}
}

// playerBarHeight is the room the page leaves for the bar, inset and all.
func playerBarHeight(c *ui.Context) float32 {
	o := playerBarOptions(c)
	return o.Height + 2*o.Inset
}

// sidebarWidth returns the width the rail is drawn at, which is nothing while it
// is closed.
func (a *app) sidebarWidth() float32 {
	if a.sidebarPixels > 0 {
		return a.sidebarPixels
	}
	return railWidth
}
