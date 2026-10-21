package src

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/icons/lucide"

	"youtube-playlist/internal/music"
)

// playerPage draws the full player: the artwork, the controls, and what plays
// next.
//
// The artwork is the one thing on the page with no words, so it takes the
// room the page can spare and the eye lands on it first; everything else is
// centered under it on one axis, as a poster with its caption.
func (a *app) playerPage(c *ui.Context) {
	t := c.Theme()
	track := a.currentTrack()

	if track.Title == "" && len(a.queue) == 0 {
		a.pageFrame(c, false, func() {
			empty(c, lucide.Disc3(), "Nothing playing",
				"Paste a playlist on the home page, and the track that plays takes the stage here.")
		})
		return
	}

	a.pageFrame(c, false, func() {
		ui.Row(c).Fill().Gap(t.Space(8)).Children(func() {
			a.playerStage(c, track)
			a.playerNext(c)
		})
	})
}

// mixButtonHeight is the shortest window that has room for the mix button under
// the transport. Below it the stage fills the window, and the button would sit
// under the player bar.
const mixButtonHeight = 700

// playerStage is the artwork, what plays, and the controls under it.
func (a *app) playerStage(c *ui.Context, track music.Track) {
	s := surface(c)
	t := c.Theme()
	// The artwork is square, and as large as the window leaves room for once
	// the words and the controls have theirs: a short window shrinks it
	// instead of cutting it off.
	_, windowHeight := c.Size()
	art := min(t.Space(80), windowHeight*0.4)

	ui.Column(c).Grow(1).FillHeight().MinWidth(0).AlignItems(ui.Center).
		Justify(ui.Center).Gap(t.Space(5)).Children(func() {
		a.picture(c, track.Thumbnail, art, radiusCard(t))
		ui.Column(c).FillWidth().Gap(t.Space(0.5)).Center().Children(func() {
			leadName(c, track.Title).TextAlign(ui.Center)
			line := trackLine(track.Title, track.Artist, track.Album)
			if line != track.Title {
				components.Text(c, line).FontSize(t.Rem(remBody)).
					TextColor(s.muted()).MaxLines(1).Ellipsis("…").
					TextAlign(ui.Center)
			}
		})

		ui.Column(c).FillWidth().MaxWidth(t.Space(130)).Gap(t.Space(2)).Children(func() {
			ui.Row(c).FillWidth().Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
				caption(c, clock(a.times2())).Width(t.Space(11)).TextAlign(ui.End)
				a.seekSlider(c, a.durationOf())
				caption(c, clock(a.durationOf())).Width(t.Space(11))
			})
			ui.Row(c).FillWidth().Center().Children(func() {
				a.transport(c, a.isPlaying(), true)
			})
			// A mix is built around the track that is playing, so this is the
			// one place it can be started from, and the one mix that needs no
			// account.
			//
			// The smallest window has no room for it: the stage already fills
			// it, and a button drawn there would sit under the player bar,
			// where it cannot be reached. It comes back as the window grows.
			if track.ID != "" && windowHeight >= mixButtonHeight {
				ui.Row(c).FillWidth().Center().Children(func() {
					if secondaryButton(c, lucide.Radio(), "Start a mix").Clicked() {
						a.playMix()
					}
				})
			}
		})
	})
}

// playerNext is what plays after this: the queue, on a card of its own beside
// the player.
//
// It opens on the track that plays, not on the first of the queue: a playlist
// of a thousand tracks would otherwise need a scroll to find what is on. When
// the track changes it follows, and the reader can still scroll anywhere after
// that.
func (a *app) playerNext(c *ui.Context) {
	s := surface(c)
	t := c.Theme()
	ui.Column(c).Width(t.Space(88)).Shrink(0).FillHeight().MinHeight(0).Children(func() {
		card(c, func() {
			listHead(c, "Up next", fmt.Sprintf("%d", len(a.queue)))
			if len(a.queue) == 0 {
				ui.Box(c).Grow(1).MinHeight(t.Space(20)).Center().Children(func() {
					components.Text(c, "The queue is empty.").FontSize(t.Rem(remBody)).
						TextColor(s.muted())
				})
				return
			}
			a.followPlaying(c, &a.queueL)
			components.List(c, &a.queueL, len(a.queue), func(i int) {
				a.queueRow(c, i)
			}).Grow(1).MinHeight(t.Space(30))
		}).Grow(1).MinHeight(0)
	})
}

// followPlaying scrolls a list to the track that plays, once, when it changes.
//
// "Once" matters: scrolling on every frame would fight the reader, who may
// want to look further down the queue while the music plays.
func (a *app) followPlaying(c *ui.Context, list *components.ListState) {
	current := a.currentIndex()
	if current < 0 || current == a.followedTrack {
		return
	}
	a.followedTrack = current
	// The list lays out after this frame, so the scroll is asked for then.
	c.After(0)
	list.ScrollTo(current, ui.Start)
}

// queuePage draws the queue: the tracks that play, and the one that plays
// now.
func (a *app) queuePage(c *ui.Context) {
	t := c.Theme()
	if len(a.queue) == 0 {
		a.pageFrame(c, false, func() {
			empty(c, lucide.ListOrdered(), "The queue is empty",
				"Play a playlist, and its tracks line up here in the order they will play.")
		})
		return
	}
	a.pageFrame(c, false, func() {
		ui.Column(c).Fill().Gap(t.Space(5)).Children(func() {
			pageHead(c, "Queue", fmt.Sprintf("%d tracks · %s", len(a.queue), totalLength(a.queue)), nil)
			components.List(c, &a.queueL, len(a.queue), func(i int) {
				a.queueRow(c, i)
			}).Grow(1).MinHeight(t.Space(30))
		})
	})
}

// queueRow draws one track of the queue, the one that plays marked.
func (a *app) queueRow(c *ui.Context, i int) {
	s := surface(c)
	t := c.Theme()
	if i < 0 || i >= len(a.queue) {
		return
	}
	track := a.queue[i]
	playing := i == a.currentIndex()
	button := ui.ButtonBase(c).Key("queue-"+track.ID).Grow(1).FillWidth().
		Radius(radiusControl(t)).Padding(t.Space(1.5), t.Space(2)).Transition(motionPress)
	switch {
	case playing:
		button.Background(s.chosen())
	case button.Hovered():
		button.Background(s.card())
	}
	button.Children(func() {
		ui.Row(c).FillWidth().Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			a.picture(c, track.Thumbnail, t.Space(9), radiusControl(t))
			ui.Column(c).Grow(1).MinWidth(0).Gap(t.Space(0.25)).Children(func() {
				components.Text(c, track.Title).FontSize(t.Rem(remBody)).FontWeight(500).
					MaxLines(1).Ellipsis("…")
				line := trackLine(track.Title, track.Artist, track.Album)
				if line != track.Title {
					caption(c, line).MaxLines(1).Ellipsis("…")
				}
			})
			if playing {
				components.Icon(c, lucide.AudioLines()).Size(t.Space(4), t.Space(4)).
					TextColor(s.accent())
			}
			caption(c, clock(track.Duration)).Width(t.Space(11)).TextAlign(ui.End)
		})
	})
	if button.Clicked() {
		a.play(i)
	}
}
