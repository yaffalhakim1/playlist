package src

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// destination is a page of the app.
type destination struct {
	id    string
	label string
	icon  *ui.SVG
}

// destinations are the pages, in the order the rail shows them.
//
// Two groups, as the design has them: the two places that are about what the app
// does, then the library. A playlist is not a page — it is an item in the rail's
// own list — so opening one shows its tracks on the Queue page.
var (
	navDestinations = []destination{
		{pageHome, "Home", iconHome},
		{pagePlayer, "Player", iconDisc},
	}
	libraryDestinations = []destination{
		{pageQueue, "Listen Now", iconPlayCircle},
		{pageStats, "History", iconHistory},
		{pageAppearance, "Appearance", iconPalette},
	}
)

// railWidth is how wide the sidebar is drawn: the design's 240 DIPs.
const railWidth = 240

// railMotion is how long the rail takes to open or close.
var railMotion = 220 * time.Millisecond

// sidebar draws the rail: the window's own dots, the two groups of pages, the
// playlists that were played before, and what is playing now.
//
// The header floats over the window, so the rail keeps its height clear above its
// first row and scrolls under it.
func (a *app) sidebar(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(a.headerHeight(c), t.Space(2.5), t.Space(2.5), t.Space(2.5)).
		Children(func() {
			ui.Scroll(c).Fill().ScrollbarInsets(t.Space(2)).Children(func() {
				ui.Column(c).FillWidth().Children(func() {
					a.navGroup(c, navDestinations)
					a.libraryGroup(c)
					if len(a.recents) > 0 {
						a.playlistGroup(c)
					}
				})
			})
		})
}

// navGroup draws the two pages at the top, with no heading over them.
func (a *app) navGroup(c *ui.Context, items []destination) {
	t := c.Theme()
	ui.Column(c).FillWidth().Gap(t.Space(0.5)).Children(func() {
		for _, item := range items {
			a.destination(c, item)
		}
	})
}

// libraryGroup draws the Library heading and the pages under it.
func (a *app) libraryGroup(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).FillWidth().Gap(t.Space(0.5)).Padding(t.Space(5), 0, 0, 0).Children(func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Padding(0, t.Space(2.5), t.Space(1.5), t.Space(2.5)).
			Children(func() {
			ui.Text(c, "LIBRARY").FontSize(t.Rem(remCaption)).FontWeight(600).
				LetterSpacing(0.06 * t.Rem(remCaption)).TextColor(surface(c).muted())
		})
		for _, item := range libraryDestinations {
			a.destination(c, item)
		}
	})
}

// playlistGroup draws the playlists that were played before, as names under a
// heading, which is how the design has them: a rail of text rather than a column
// of covers, so the page keeps the pictures.
func (a *app) playlistGroup(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).FillWidth().Gap(t.Space(0.5)).Padding(t.Space(5), 0, 0, 0).Children(func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Padding(0, t.Space(2.5), t.Space(1.5), t.Space(2.5)).
			Children(func() {
			ui.Text(c, "PLAYLISTS").FontSize(t.Rem(remCaption)).FontWeight(600).
				LetterSpacing(0.06 * t.Rem(remCaption)).TextColor(surface(c).muted())
		})
		for _, recent := range a.recents {
			a.recentItem(c, recent.Link, recent.Title)
		}
	})
}

// destination draws one page of the rail: a soft pill that turns white when it is
// the page that shows.
func (a *app) destination(c *ui.Context, item destination) {
	s := surface(c)
	t := c.Theme()
	selected := a.page == item.id
	b := ui.ButtonBase(c).Key("nav-"+item.id).FillWidth().Gap(t.Space(2.5)).
		Padding(t.Space(1.5), t.Space(3)).Radius(radiusControl(t)).
		AlignItems(ui.Center).Label(item.label)
	face := ui.Transparent
	switch {
	case selected:
		face = s.card()
	case b.Hovered():
		face = s.card().Alpha(0.6)
	}
	b.Background(b.AnimateColor("face", face, motionColor))
	if selected {
		b.Shadow(0, 1, 3, 0, s.shadow())
	}
	ink := s.muted()
	if selected {
		ink = s.text()
	}
	b.Children(func() {
		ui.Row(c).FillWidth().Gap(t.Space(2.5)).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, item.icon).Size(t.Space(4.5), t.Space(4.5)).TextColor(ink)
			ui.Text(c, item.label).FontSize(t.Rem(remBody)).FontWeight(500).
				TextColor(ink).Grow(1).MaxLines(1).Ellipsis("…")
		})
	})
	if b.Clicked() {
		a.page = item.id
	}
}

// recentItem draws one of the playlists that were played before, as a name.
//
// A right click offers to forget it, which asks first: deleting is the one thing
// here that cannot be undone.
func (a *app) recentItem(c *ui.Context, link, title string) {
	s := surface(c)
	t := c.Theme()
	if title == "" {
		title = link
	}
	selected := a.openLink == link && a.page == pageQueue
	b := ui.ButtonBase(c).Key("recent-"+link).FillWidth().
		Padding(t.Space(1.5), t.Space(3)).Radius(radiusControl(t)).
		AlignItems(ui.Center).Label(title)
	face := ui.Transparent
	if b.Hovered() {
		face = s.card().Alpha(0.6)
	}
	b.Background(b.AnimateColor("face", face, motionColor))
	ink := s.muted()
	if selected {
		ink = s.text()
	}
	b.Children(func() {
		ui.Text(c, title).FontSize(t.Rem(remBody)).TextColor(ink).
			MaxLines(1).Ellipsis("…")
	})
	b.ContextMenu(func(m *ui.Menu) {
		if m.Item("Delete playlist").Chosen() {
			a.askDelete(link, title)
		}
	})
	if b.Clicked() {
		a.openRecent(link)
	}
}

// miniPlayer is the card at the foot of the rail: what plays, its cover, and one
// button that plays or holds it.
//
// It is the rail's answer to the bar at the bottom of the window, for the pages
// that have one: the Player page has its own transport, and this one is small
// enough to sit under the playlists without taking their room.
func (a *app) miniPlayer(c *ui.Context) {
	s := surface(c)
	t := c.Theme()
	track := a.currentTrack()
	ui.Box(c).FillWidth().Shrink(0).Padding(t.Space(2.5)).Radius(radiusControl(t)+t.Space(1)).
		Background(s.card()).Border(1, s.hairline()).
		Shadow(0, 1, 4, 0, s.shadow()).Children(func() {
		ui.Row(c).FillWidth().Gap(t.Space(2.5)).AlignItems(ui.Center).Children(func() {
			a.picture(c, track.Thumbnail, t.Space(10), radiusControl(t))
			ui.Column(c).Grow(1).MinWidth(0).Gap(t.Space(0.25)).Children(func() {
				if track.Title == "" {
					ui.Text(c, "Nothing playing").FontSize(t.Rem(remCaption)).
						TextColor(s.muted()).MaxLines(1).Ellipsis("…")
					return
				}
				ui.Text(c, track.Title).FontSize(t.Rem(remCaption)).FontWeight(600).
					TextColor(s.text()).MaxLines(1).Ellipsis("…")
				line := trackLine(track.Title, track.Artist, track.Album)
				if line != track.Title {
					ui.Text(c, line).FontSize(t.Rem(remCaption)).TextColor(s.muted()).
						MaxLines(1).Ellipsis("…")
				}
			})
			if len(a.queue) > 0 {
				if b := iconButton(c, iconHeart, "Like"); b.Clicked() {
					// There is no favourites store yet; the button is drawn for
					// the design and does nothing until there is one.
				}
			}
		})
	})
}
