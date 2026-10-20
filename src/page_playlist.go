package src

import (
	"fmt"
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/icons/lucide"

	"youtube-playlist/internal/music"
)

// playlistPage draws the playlist that is loaded: its head, and its tracks.
//
// The head is the cover beside what the playlist is, and the songs are one
// list that takes the height that is left, so the page scrolls only when the
// list is short and the list scrolls itself when it is long.
func (a *app) playlistPage(c *ui.Context) {
	t := c.Theme()
	if a.playlist == nil {
		a.pageFrame(c, false, func() {
			empty(c, lucide.ListMusic(), "No playlist yet",
				"Paste a link on the home page, and its songs line up here.")
		})
		return
	}

	a.pageFrame(c, false, func() {
		ui.Column(c).Fill().Gap(t.Space(7)).Children(func() {
			a.playlistHead(c)
			ui.Column(c).FillWidth().Grow(1).MinHeight(0).Gap(t.Space(3)).Children(func() {
				note := ""
				if len(a.visible) != len(a.queue) {
					note = fmt.Sprintf("%d of %d", len(a.visible), len(a.queue))
				}
				listHead(c, "Songs", note)
				components.List(c, &a.list, len(a.visible), func(i int) {
					a.trackRow(c, i)
				}).Grow(1).MinHeight(t.Space(40))
			})
		})
	})
}

// playlistHead draws the cover of the playlist, its name, and what can be
// done with it.
func (a *app) playlistHead(c *ui.Context) {
	t := c.Theme()
	art := t.Space(42)
	ui.Row(c).FillWidth().Gap(t.Space(6)).AlignItems(ui.Center).Children(func() {
		a.picture(c, a.playlist.Thumbnail, art, radiusCard(t))

		ui.Column(c).Grow(1).MinWidth(0).Gap(t.Space(2)).Children(func() {
			components.Text(c, a.playlist.Title).FontSize(t.Rem(remDisplay)).
				FontWeight(600).LetterSpacing(trackDisplay(t)).LineHeight(leadDisplay).
				MaxLines(2).Ellipsis("…")
			caption(c, fmt.Sprintf("%d tracks · %s", len(a.queue), totalLength(a.queue)))

			ui.Row(c).Gap(t.Space(2)).Padding(0, t.Space(1), 0, 0).Children(func() {
				play := primaryButton(c, lucide.Play(), "Play")
				if play.Clicked() {
					a.playAll()
				}
				shuffle := secondaryButton(c, lucide.Shuffle(), "Shuffle")
				if shuffle.Clicked() {
					a.shuffleAndPlay()
				}
			})
		})
	})
}

// totalLength returns how long the whole playlist plays.
func totalLength(tracks []music.Track) string {
	var total time.Duration
	for _, track := range tracks {
		total += track.Duration
	}
	if total == 0 {
		return "unknown length"
	}
	return clock(total)
}

// trackRow draws one track of the list.
//
// The row is quiet until the pointer is on it or it is the track that plays:
// the number, the cover, the name and the length, and nothing else, because a
// list read in a glance wants no boxes.
func (a *app) trackRow(c *ui.Context, i int) {
	s := surface(c)
	t := c.Theme()
	if i < 0 || i >= len(a.visible) {
		return
	}
	index := a.visible[i]
	if index < 0 || index >= len(a.queue) {
		return
	}
	track := a.queue[index]
	playing := index == a.currentIndex()

	button := ui.ButtonBase(c).Key("track-"+track.ID).Grow(1).FillWidth().
		Radius(radiusControl(t)).Padding(t.Space(1.5), t.Space(2)).Transition(motionPress)
	switch {
	case playing:
		button.Background(s.chosen())
	case button.Hovered():
		button.Background(s.card())
	}
	button.Children(func() {
		ui.Row(c).FillWidth().Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Width(t.Space(6)).Shrink(0).Center().Children(func() {
				if playing {
					components.Icon(c, lucide.AudioLines()).Size(t.Space(4), t.Space(4)).
						TextColor(s.accent())
					return
				}
				components.Text(c, strconv.Itoa(i+1)).FontSize(t.Rem(remCaption)).
					TextColor(s.muted()).FontFeatures("tnum")
			})
			a.picture(c, track.Thumbnail, t.Space(10), radiusControl(t))
			ui.Column(c).Grow(1).MinWidth(0).Gap(t.Space(0.25)).Children(func() {
				components.Text(c, track.Title).FontSize(t.Rem(remTitle)).
					FontWeight(500).LineHeight(leadTitle).MaxLines(1).Ellipsis("…")
				line := trackLine(track.Title, track.Artist, track.Album)
				if line != track.Title {
					caption(c, line).MaxLines(1).Ellipsis("…")
				}
			})
			caption(c, clock(track.Duration)).Width(t.Space(12))
		})
	})
	if button.Clicked() {
		a.play(index)
	}
}
