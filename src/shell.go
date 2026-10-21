package src

import (
	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/icons/lucide"
)

// destination is a page of the app.
type destination struct {
	id    string
	label string
	icon  func() *ui.SVG
}

// destinations are the pages, in the order the sidebar shows them.
var destinations = []destination{
	{pageHome, "Home", lucide.House},
	{pagePlaylist, "Playlist", lucide.ListMusic},
	{pagePlayer, "Player", lucide.Disc3},
	{pageQueue, "Queue", lucide.ListOrdered},
	{pageStats, "Stats", lucide.ChartBar},
	{pageAppearance, "Look", lucide.Palette},
}

// sidebar draws the navigation of the window: the pages, and the playlists
// that were played before, under their artwork.
//
// The shell floats its header over the sidebar and puts its collapse control
// at the top left of it, so the sidebar keeps the header's height clear above
// its first section and lets the rest scroll under the bar below.
func (a *app) sidebar(c *ui.Context) {
	ui.Column(c).Fill().Padding(a.headerHeight(c)+c.Theme().Space(1), 0, 0).Children(func() {
		components.ScrollView(c, components.ScrollViewOptions{}).Fill().Children(func() {
			components.Sidebar(c).Children(func() {
				components.SidebarSection(c, "Library", func() {
					for _, item := range destinations {
						a.destination(c, item)
					}
				})
				if len(a.recents) > 0 {
					components.SidebarSection(c, "Playlists", func() {
						for _, recent := range a.recents {
							a.recentItem(c, recent.Link, recent.Title, recent.Thumbnail)
						}
					})
				}
			})
		})
	})
}

// destination draws one destination of the sidebar.
func (a *app) destination(c *ui.Context, item destination) {
	selected := a.page == item.id
	button := components.SidebarItem(c, item.label, item.icon(), selected).Key("nav-" + item.id)
	if button.Clicked() {
		a.page = item.id
	}
}

// recentItem draws one of the playlists that were played before, under the
// artwork it came with.
//
// A right click on it offers to forget it, which removes it from the sidebar,
// from the cache, and from the session if it was the one open. Deleting is the
// one thing here that cannot be undone, so it asks first.
func (a *app) recentItem(c *ui.Context, link, title, thumbnail string) {
	if title == "" {
		title = link
	}
	var item ui.Element
	if bitmap := a.art.get(thumbnail); bitmap != nil {
		item = components.SidebarImageItem(c, title, bitmap, false)
	} else {
		a.art.want(thumbnail)
		item = components.SidebarItem(c, title, lucide.ListMusic(), false)
	}
	item.Key("recent-" + link)
	item.ContextMenu(func(m *ui.Menu) {
		if m.Item("Delete playlist").Chosen() {
			a.askDelete(link, title)
		}
	})
	if item.Clicked() {
		a.link = link
		a.load()
	}
}

// content draws the page, with the player bar under it.
//
// The page keeps clear of the bar, so nothing rests behind it and nothing
// scrolls under it: the bar is the last row of the window, not a thing
// floating over the page. A page that replaces another arrives from a little
// below and fades in, keyed by which page it is, so the change reads as one
// surface replacing another rather than as a flicker.
func (a *app) content(c *ui.Context) {
	ui.Box(c).Fill().Clip().Children(func() {
		ui.Box(c).Key(a.page).Fill().Transition(motionReveal).Children(func() {
			a.pageContent(c)
		})
		a.playerbar(c)
	})
}

// pageTitle returns the name of the page that shows.
func (a *app) pageTitle() string {
	switch a.page {
	case pagePlaylist:
		if a.playlist != nil && a.playlist.Title != "" {
			return a.playlist.Title
		}
		return "Playlist"
	case pagePlayer:
		return "Now playing"
	case pageQueue:
		return "Queue"
	case pageStats:
		return "Listening stats"
	case pageAppearance:
		return "Appearance"
	default:
		return "Home"
	}
}

// pageContent draws the page that shows.
func (a *app) pageContent(c *ui.Context) {
	switch a.page {
	case pagePlaylist:
		a.playlistPage(c)
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

// pageFrame gives a page the room it lives in: the same margins on every
// page, clear of the header above and of the bar below, and never wider than
// a line of text can be followed.
//
// The bar keeps its own strip at the bottom of the window, outside the page,
// so nothing the page holds ever passes under it: a row of covers is cut by
// the edge of its own space rather than by something floating over it.
//
// A page that scrolls puts its content in one scroll view; a page that holds
// a list gives the list the height that is left, so there is never a scroll
// bar inside a scroll bar.
func (a *app) pageFrame(c *ui.Context, scrolling bool, build func()) {
	t := c.Theme()
	top := a.headerHeight(c) + t.Space(6)
	side := pageInset(c)
	bottom := components.ScrollEdgeBarExtent(c, playerBarOptions(c)) + t.Space(2)
	column := func() {
		ui.Column(c).FillWidth().MaxWidth(pageWidth(c)).AlignSelf(ui.Center).Children(build)
	}
	if scrolling {
		ui.Box(c).Fill().Padding(top, side, bottom, side).Children(func() {
			components.ScrollView(c, components.ScrollViewOptions{}).Fill().Children(column)
		})
		return
	}
	ui.Box(c).Fill().Padding(top, side, bottom, side).Children(func() {
		ui.Column(c).Fill().MaxWidth(pageWidth(c)).AlignSelf(ui.Center).Children(build)
	})
}

// pageWidth is how wide a page grows before it stops: past it a line of text
// is hard to follow and a row of covers leaves a desert of empty page beside
// it, so the column centers in the window instead.
func pageWidth(c *ui.Context) float32 { return c.Theme().Space(238) }

// pageInset is the margin a page keeps from the window, and the one the
// header's own actions keep, so an action ends where the page does. A narrow
// window keeps less of it, so the page still has room for what it holds.
func pageInset(c *ui.Context) float32 {
	t := c.Theme()
	if width, _ := c.Size(); width < 1040 {
		return t.Space(6)
	}
	return t.Space(8)
}

// playerBarOptions describes the bar that floats over the page.
func playerBarOptions(c *ui.Context) components.ScrollEdgeBarOptions {
	t := c.Theme()
	return components.ScrollEdgeBarOptions{
		Style:  components.ScrollEdgeSoft,
		Bottom: true,
		Height: t.Space(22),
		Inset:  t.Space(4),
	}
}
