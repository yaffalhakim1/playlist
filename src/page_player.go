package src

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"youtube-playlist/internal/music"
)

// playerPage draws the focused player: the artwork and its transport at the left,
// and what plays next beside it.
//
// The page is two columns on one axis. The left is a poster — the cover, the name
// of the track, where it is, and the controls under it — because the cover is the
// one thing here with no words and the eye lands on it first. The right is the
// queue, so the page answers "what is this" and "what is next" at once.
func (a *app) playerPage(c *ui.Context) {
	t := c.Theme()
	track := a.currentTrack()

	if track.Title == "" && len(a.queue) == 0 {
		a.pageFrame(c, false, func() {
			empty(c, iconDisc, "Nothing playing",
				"Paste a playlist on the home page, and the track that plays takes the stage here.")
		})
		return
	}

	a.pageFrame(c, false, func() {
		ui.Column(c).FillWidth().FillHeight().Gap(t.Space(3)).Children(func() {
			a.backLink(c)
			ui.Row(c).FillWidth().Grow(1).MinHeight(0).Gap(t.Space(10)).Children(func() {
				a.playerStage(c, track)
				a.playerNext(c)
			})
		})
	})
}

// backLink is the breadcrumb the design puts over the stage: the playlist the
// track came from, which is the Queue page.
func (a *app) backLink(c *ui.Context) {
	s := surface(c)
	t := c.Theme()
	b := ui.ButtonBase(c).Padding(t.Space(1), t.Space(2)).Radius(radiusControl(t)).
		Label("Playlist").AlignSelf(ui.Start)
	face := ui.Transparent
	if b.Hovered() {
		face = s.sunken()
	}
	b.Background(b.AnimateColor("face", face, motionColor))
	b.Children(func() {
		ui.Row(c).Gap(t.Space(1)).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconChevronLeft).Size(t.Space(4), t.Space(4)).TextColor(s.muted())
			ui.Text(c, "Playlist").FontSize(t.Rem(remBody)).FontWeight(500).
				TextColor(s.muted())
		})
	})
	if b.Clicked() {
		a.page = pageQueue
	}
}

// playerStage is the artwork, what plays, and the controls under it.
//
// The cover is square and as large as the window leaves room for, so a short
// window shrinks it rather than cutting it off. The transport sits under the name
// on the same axis, as a poster with its caption.
func (a *app) playerStage(c *ui.Context, track music.Track) {
	s := surface(c)
	t := c.Theme()
	_, windowHeight := c.Size()
	// The cover is the subject, so it takes what the page can spare: a square
	// that follows the window's height, between a floor that keeps it readable
	// and a ceiling that stops it crowding the queue.
	art := min(t.Space(84), max(t.Space(44), windowHeight*0.42))

	ui.Column(c).Grow(1).FillHeight().MinWidth(0).AlignItems(ui.Center).
		Justify(ui.Center).Gap(t.Space(6)).Children(func() {
		// The cover, with the design's warm shadow rather than a hairline: it is
		// a photograph, and a line around it would cut it out of the page.
		ui.Box(c).Size(art, art).Shrink(0).Radius(radiusCard(t)+t.Space(1)).
			Clip().Shadow(0, 12, 36, 0, s.shadow()).Children(func() {
			a.picture(c, track.Thumbnail, art, radiusCard(t)+t.Space(1))
		})

		// The name and what is under it, centered on the same axis.
		ui.Column(c).FillWidth().MaxWidth(t.Space(110)).Center().Gap(t.Space(1.5)).
			Children(func() {
			ui.Text(c, track.Title).FontSize(t.Rem(remLead)).FontWeight(700).
				LetterSpacing(trackHeadline(t)).LineHeight(leadHeadline).
				MaxLines(2).Ellipsis("…").TextAlign(ui.Center).TextColor(s.text())
			ui.Text(c, trackLine(track.Title, track.Artist, track.Album)).
				FontSize(t.Rem(remBody)).FontWeight(500).TextColor(s.muted()).
				MaxLines(1).Ellipsis("…").TextAlign(ui.Center)
		})

		// The timeline, then the transport, on the same column as the name.
		ui.Column(c).FillWidth().MaxWidth(t.Space(110)).Gap(t.Space(2)).
			Children(func() {
			ui.Row(c).FillWidth().Gap(t.Space(2.5)).AlignItems(ui.Center).Children(func() {
				mono(c, clock(a.times2())).Width(t.Space(9)).TextAlign(ui.End)
				a.seekSlider(c, a.durationOf())
				mono(c, clock(a.durationOf())).Width(t.Space(9))
			})
			ui.Row(c).FillWidth().Center().Children(func() {
				a.transport(c, a.isPlaying(), true)
			})
		})
	})
}

// playerNext is what plays after this: the queue, in a panel beside the player.
//
// It opens on the track that plays, not on the first of the queue: a playlist of a
// thousand tracks would otherwise need a scroll to find what is on. When the track
// changes it follows, and the reader can still scroll anywhere after that.
func (a *app) playerNext(c *ui.Context) {
	s := surface(c)
	t := c.Theme()
	ui.Column(c).Grow(1).MinWidth(0).FillHeight().MinHeight(0).Children(func() {
		ui.Column(c).FillWidth().FillHeight().Gap(t.Space(3)).Padding(t.Space(5)).
			Radius(radiusCard(t)+t.Space(1)).Background(s.card()).
			Border(1, s.hairline()).Shadow(0, 2, 12, 0, s.shadow()).Children(func() {
			// The panel's head: what it is, and how much of it there is.
			ui.Row(c).FillWidth().Gap(t.Space(4)).AlignItems(ui.Center).
				Padding(0, t.Space(2)).Children(func() {
				ui.Text(c, "Up next").FontSize(t.Rem(remTitle)).FontWeight(700).
					TextColor(s.text())
				ui.Spacer(c)
				mono(c, fmt.Sprintf("%d", len(a.queue)))
			})
			if len(a.queue) == 0 {
				ui.Box(c).Grow(1).MinHeight(t.Space(20)).Center().Children(func() {
					ui.Text(c, "The queue is empty.").FontSize(t.Rem(remBody)).
						TextColor(s.muted())
				})
				return
			}
			a.followPlaying(c, &a.queueL)
			ui.List(c, &a.queueL, len(a.queue), func(i int) {
				a.upNextRow(c, i)
			}).Grow(1).MinHeight(t.Space(30)).Label("Up next")
		})
	})
}

// followPlaying scrolls a list to the track that plays, once, when it changes.
//
// "Once" matters: scrolling on every frame would fight the reader, who may want to
// look further down the queue while the music plays.
func (a *app) followPlaying(c *ui.Context, list *ui.ListState) {
	current := a.currentIndex()
	if current < 0 || current == a.followedTrack {
		return
	}
	a.followedTrack = current
	// The list lays out after this frame, so the scroll is asked for then.
	c.After(0)
	list.ScrollTo(current, ui.Start)
}

// upNextRow draws one track of the panel: its cover, its name, and its length.
//
// The row that plays is the one the design marks: a soft pill, and the bars in
// place of nothing, so it is found without reading.
func (a *app) upNextRow(c *ui.Context, i int) {
	s := surface(c)
	t := c.Theme()
	if i < 0 || i >= len(a.queue) {
		return
	}
	track := a.queue[i]
	playing := i == a.currentIndex()
	button := ui.ButtonBase(c).Key("upnext-"+track.ID).Grow(1).FillWidth().
		Radius(radiusCard(t)).Padding(t.Space(2.5), t.Space(2))
	face := ui.Transparent
	switch {
	case playing:
		face = s.sunken()
	case button.Hovered():
		face = s.chosen().Alpha(0.5)
	}
	button.Background(button.AnimateColor("face", face, motionColor))
	button.Children(func() {
		ui.Row(c).FillWidth().Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			a.picture(c, track.Thumbnail, t.Space(10), radiusControl(t))
			ui.Column(c).Grow(1).MinWidth(0).Gap(t.Space(0.25)).Children(func() {
				ui.Text(c, track.Title).FontSize(t.Rem(remCaption)).FontWeight(600).
					TextColor(s.text()).MaxLines(1).Ellipsis("…")
				line := trackLine(track.Title, track.Artist, track.Album)
				if line != track.Title {
					ui.Text(c, line).FontSize(t.Rem(remCaption)).TextColor(s.muted()).
						MaxLines(1).Ellipsis("…")
				}
			})
			if playing {
				a.nowPlayingBars(c, t.Space(4), s.text())
			}
			mono(c, clock(track.Duration)).Shrink(0)
		})
	})
	if button.Clicked() {
		a.play(i)
	}
}
