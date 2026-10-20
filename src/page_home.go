package src

import (
	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/icons/lucide"

	"youtube-playlist/internal/library"
)

// homePage draws the page one lands on: the link to play, and what was
// played before.
//
// It has one job, so it says it once and leaves the rest of the page quiet:
// a name, a line about what the app does, the field, and the playlists that
// were played before as pictures you recognise before you read.
func (a *app) homePage(c *ui.Context) {
	t := c.Theme()
	a.pageFrame(c, true, func() {
		ui.Column(c).FillWidth().Gap(t.Space(10)).Children(func() {
			a.homeHero(c)
			if len(a.recents) > 0 {
				a.homeRecents(c)
			}
			if len(a.queue) > 0 {
				a.homePlaying(c)
			}
		})
	})
}

// homeHero is what the app does, and the field that does it.
//
// No label sits above the headline: the sentence is the label, and a tracked
// caption over it would only repeat the word "playlist" twice before saying
// anything.
func (a *app) homeHero(c *ui.Context) {
	s := surface(c)
	t := c.Theme()
	ui.Column(c).FillWidth().Gap(t.Space(3)).Padding(0, 0, t.Space(4)).Children(func() {
		components.Text(c, "Paste a link. Hear the music.").
			FontSize(t.Rem(remDisplay)).FontWeight(600).
			LetterSpacing(trackDisplay(t)).LineHeight(leadDisplay).
			MaxLines(2).MaxWidth(t.Space(120))
		body(c, "A playlist of YouTube Music, or any playlist of YouTube. The music plays; the video never shows.").
			MaxLines(2)

		ui.Row(c).FillWidth().Gap(t.Space(3)).AlignItems(ui.Center).
			Padding(0, t.Space(1), 0, 0).Children(func() {
			box := components.TextBox(c, &a.link, components.TextBoxOptions{LeadingIcon: lucide.Link()})
			box.Grow(1).Radius(radiusPill(t)).Height(t.Space(11)).
				Label("Playlist link").
				Placeholder("https://music.youtube.com/playlist?list=…").
				OnSubmit(func() { a.load() })
			play := primaryButton(c, lucide.Play(), "Play")
			if a.loading {
				play = primaryButton(c, lucide.Play(), "Reading…")
				play.Disabled(true)
			}
			if play.Clicked() {
				a.load()
			}
		})
		if a.status != "" {
			ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				components.Icon(c, lucide.AudioLines()).Size(t.Space(4), t.Space(4)).
					TextColor(s.accent())
				components.Text(c, a.status).FontSize(t.Rem(remCaption)).
					TextColor(t.Danger).MaxLines(2).Ellipsis("…")
			})
		}
	})
}

// homeRecents draws the playlists that were played before, as a shelf that
// starts at the page's own left edge and runs off the right: the covers are
// what is recognised, so they are large, and the row is scrolled sideways
// rather than wrapping into a second row of smaller ones.
func (a *app) homeRecents(c *ui.Context) {
	t := c.Theme()
	cover := t.Space(40)
	// The shelf is as tall as a card: the cover, the name under it, and the
	// word that says what it is, with the card's own room around them.
	shelf := cover + t.Rem(remBody)*leadTitle + t.Rem(remCaption)*1.6 + t.Space(7)
	ui.Column(c).FillWidth().Gap(t.Space(3)).Children(func() {
		listHead(c, "Listen again", "")
		components.ScrollView(c, components.ScrollViewOptions{Axes: components.ScrollHorizontal}).
			FillWidth().Height(shelf).
			ScrollbarInsets(0, 0, t.Space(2), 0).Children(func() {
			ui.Row(c).Gap(t.Space(3)).Children(func() {
				for _, recent := range a.recents {
					a.recentCard(c, recent, cover)
				}
			})
		})
	})
}

// homePlaying says what the queue is, when there is one, and opens it.
func (a *app) homePlaying(c *ui.Context) {
	t := c.Theme()
	var open ui.Element
	card(c, func() {
		ui.Row(c).FillWidth().Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Gap(t.Space(0.5)).Children(func() {
				caption(c, "Playing from")
				cardName(c, a.playlistTitle()).MaxLines(1).Ellipsis("…")
			})
			open = components.Button(c, "", components.ButtonGhost).Label("Open queue")
			open.Children(func() {
				ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
					components.Text(c, "Open queue").FontSize(t.Rem(remControl)).
						FontWeight(500).TextColor(surface(c).accent())
					components.Icon(c, lucide.ChevronRight()).Size(t.Space(4), t.Space(4)).
						TextColor(surface(c).accent())
				})
			})
		})
	})
	if open.Clicked() {
		a.page = pageQueue
	}
}

// recentCard draws one of the playlists that were played before.
//
// A width of zero takes the width of the slide the card sits in.
func (a *app) recentCard(c *ui.Context, recent library.Recent, width float32) {
	t := c.Theme()
	title := recent.Title
	if title == "" {
		title = recent.Link
	}
	button := ui.ButtonBase(c).Key("recent-" + recent.Link).Padding(t.Space(1)).
		Radius(radiusCard(t)).Gap(t.Space(2)).Shrink(0)
	if width > 0 {
		button.Width(width)
	} else {
		button.FillWidth()
	}
	button.Children(func() {
		ui.Column(c).FillWidth().Gap(t.Space(2)).Children(func() {
			art := width
			if art == 0 {
				art = t.Space(38)
			}
			a.picture(c, recent.Thumbnail, art, radiusControl(t))
			ui.Column(c).FillWidth().Gap(t.Space(0.25)).Padding(t.Space(1), 0).Children(func() {
				components.Text(c, title).FontSize(t.Rem(remBody)).FontWeight(500).
					LineHeight(leadTitle).MaxLines(1).Ellipsis("…")
				caption(c, "Playlist")
			})
		})
	})
	if button.Clicked() {
		a.link = recent.Link
		a.load()
	}
}

// playlistTitle returns the name of the playlist that plays.
func (a *app) playlistTitle() string {
	if a.playlist == nil {
		return ""
	}
	return a.playlist.Title
}
