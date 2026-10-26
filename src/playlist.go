package src

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"

	"youtube-playlist/internal/music"
)

// The parts a playlist is drawn with: its head, its table, and the shape of it
// while it is read.
//
// A playlist is the Queue page. The tracks are the queue, in the order they will
// play, with the one that plays marked — so a page called Playlist and a page
// called Queue would be the same page twice.

// playlistHead draws the playlist as a poster: the cover, what it is, and the one
// action that matters.
//
// The cover is the largest thing on the page and the name is the second, because a
// playlist is recognised before it is read. The badge over the name says what kind
// of thing it is; the line under it says how much of it there is.
func (a *app) playlistHead(c *ui.Context) {
	s := surface(c)
	t := c.Theme()
	art := t.Space(48)
	title := "Playlist"
	var thumbnail string
	if a.playlist != nil {
		title = a.playlist.Title
		thumbnail = a.playlist.Thumbnail
	}

	ui.Row(c).FillWidth().Gap(t.Space(8)).AlignItems(ui.End).Children(func() {
		// The cover, with the design's soft shadow: it is a photograph, so a
		// hairline around it would cut it out of the page.
		ui.Box(c).Size(art, art).Shrink(0).Radius(radiusCard(t)).Clip().
			Shadow(0, 6, 18, 0, s.shadow()).Children(func() {
			a.picture(c, thumbnail, art, radiusCard(t))
		})

		ui.Column(c).Grow(1).MinWidth(0).Gap(t.Space(2)).Children(func() {
			// What kind of thing this is.
			ui.Row(c).Gap(t.Space(2.5)).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Padding(t.Space(0.5), t.Space(2)).Radius(radiusPill(t)).
					Background(s.sunken()).Children(func() {
					eyebrow(c, "Playlist")
				})
				ui.Row(c).Gap(t.Space(1)).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(t.Space(1.5), t.Space(1.5)).Radius(radiusPill(t)).
						Background(s.faint())
					ui.Text(c, "Synced from YouTube").FontSize(t.Rem(remCaption)).
						TextColor(s.muted())
				})
			})
			ui.Text(c, title).FontSize(t.Rem(remDisplay)).FontWeight(700).
				LetterSpacing(trackDisplay(t)).LineHeight(leadDisplay).
				MaxLines(2).Ellipsis("…").TextColor(s.text())
			caption(c, fmt.Sprintf("%d tracks · %s", len(a.queue), totalLength(a.queue)))

			ui.Row(c).Gap(t.Space(2.5)).AlignItems(ui.Center).
				Padding(t.Space(3), 0, 0, 0).Children(func() {
				if b := primaryButton(c, iconPlay, "Play all"); b.Clicked() {
					a.playAll()
				}
				if b := secondaryButton(c, iconSync, "Re-sync"); b.Clicked() {
					a.resync()
				}
				if b := roundButton(c, iconMore, "More"); b.Clicked() {
					a.page = pagePlayer
				}
			})
		})
	})
}

// playlistFilter is the strip over the table: a field to narrow the rows.
//
// There is one chip, not two. The design has a "Downloaded" filter beside "All
// tracks", and the app streams: nothing is ever downloaded, so a chip for it would
// be a control that can only ever be empty.
func (a *app) playlistFilter(c *ui.Context) {
	s := surface(c)
	t := c.Theme()
	ui.Row(c).FillWidth().Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
		chip(c, "All tracks", true)
		ui.Spacer(c)

		// The field, in a pill of its own, as the design draws it. The contents are
		// centered on both axes: a Box lays its children out from the top left, so
		// a Row that only grows leaves the text sitting above the middle.
		ui.Box(c).Width(t.Space(48)).Height(t.Space(8)).Radius(radiusPill(t)).
			Background(s.sunken()).Center().Padding(0, t.Space(2.5)).Children(func() {
			ui.Row(c).FillWidth().Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, iconSearch).Size(t.Space(4), t.Space(4)).TextColor(s.faint())
				field := ui.TextInput(c, &a.filter).Grow(1).Border(0, ui.Transparent).
					Background(ui.Transparent).Padding(0).FontSize(t.Rem(remCaption)).
					Label("Filter playlist").Placeholder("Filter playlist…")
				if field.Changed() {
					a.refilter()
				}
			})
		})
	})
}

// playlistTable draws the tracks as the design's table: a number, the cover and
// name, the artist, the album, and the length at the right.
//
// It is one list that scrolls itself rather than a table of every row: a playlist
// of a thousand tracks draws the rows in view and no others.
func (a *app) playlistTable(c *ui.Context) {
	s := surface(c)
	t := c.Theme()
	ui.Column(c).FillWidth().FillHeight().MinHeight(0).Gap(0).Padding(t.Space(2)).
		Radius(radiusCard(t)).Background(s.card()).Border(1, s.hairline()).
		Shadow(0, 1, 4, 0, s.shadow()).Children(func() {
		// The column headings. They are not a row of the list: they stay put while
		// the tracks scroll under them.
		ui.Row(c).FillWidth().Gap(t.Space(4)).Padding(t.Space(3), t.Space(4)).
			Children(func() {
			ui.Box(c).Width(t.Space(6)).Shrink(0).Children(func() {
				ui.Text(c, "#").FontSize(t.Rem(remCaption)).FontWeight(600).
					LetterSpacing(0.06 * t.Rem(remCaption)).TextColor(s.muted()).
					TextAlign(ui.Center).Grow(1)
			})
			columnHead(c, "Title")
			columnHead(c, "Artist")
			columnHead(c, "Album")
			ui.Box(c).Width(t.Space(10)).Shrink(0).Children(func() {
				ui.Icon(c, iconClock).Size(t.Space(4), t.Space(4)).TextColor(s.muted()).
					AlignSelf(ui.End)
			})
		})
		ui.List(c, &a.list, len(a.visible), func(i int) {
			a.trackTableRow(c, i)
		}).Grow(1).MinHeight(t.Space(30)).Label("Songs")
	})
}

// columnHead is one heading of the table.
func columnHead(c *ui.Context, label string) ui.Element {
	t := c.Theme()
	return ui.Text(c, label).FontSize(t.Rem(remCaption)).FontWeight(600).
		LetterSpacing(0.06 * t.Rem(remCaption)).TextColor(surface(c).muted()).
		Grow(1).MaxLines(1)
}

// trackTableRow draws one row of the table.
//
// The row is quiet until the pointer is on it or it is the track that plays. The
// row that plays is marked by the bars in place of its number, and by a face a step
// off the card, so it is found without reading.
func (a *app) trackTableRow(c *ui.Context, i int) {
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

	row := ui.ButtonBase(c).Key("track-"+track.ID).FillWidth().
		Padding(t.Space(2.5), t.Space(4)).Radius(radiusControl(t)).
		AlignItems(ui.Center)
	face := ui.Transparent
	switch {
	case playing:
		face = s.sunken()
	case row.Hovered():
		face = s.sunken().Alpha(0.5)
	}
	row.Background(row.AnimateColor("face", face, motionColor))
	row.Children(func() {
		ui.Row(c).FillWidth().Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
			// The number, or the bars when it plays.
			ui.Box(c).Width(t.Space(6)).Shrink(0).Center().Children(func() {
				if playing {
					a.nowPlayingBars(c, t.Space(4), s.text())
					return
				}
				mono(c, fmt.Sprintf("%d", i+1)).TextAlign(ui.Center)
			})
			// The cover and the name.
			ui.Row(c).Grow(1).MinWidth(0).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
				a.picture(c, track.Thumbnail, t.Space(9), radiusControl(t))
				ui.Text(c, track.Title).FontSize(t.Rem(remCaption)).
					FontWeight(func() int {
						if playing {
							return 600
						}
						return 400
					}()).TextColor(s.text()).MaxLines(1).Ellipsis("…").Grow(1)
			})
			cell(c, track.Artist, s.muted(), false)
			cell(c, track.Album, s.muted(), false)
			// The same width as the heading above, so the clock sits under it.
			ui.Box(c).Width(t.Space(16)).Shrink(0).Children(func() {
				ink := s.muted()
				if playing {
					ink = s.text()
				}
				ui.Text(c, clock(track.Duration)).FontSize(t.Rem(remCaption)).
					FontFeatures("tnum").TextColor(ink).TextAlign(ui.End).Grow(1)
			})
		})
	})
	if row.Clicked() {
		a.play(index)
	}
}

// cell is one text column of the table, with a dash when there is nothing.
func cell(c *ui.Context, words string, colour ui.Color, bold bool) ui.Element {
	t := c.Theme()
	if words == "" {
		words = "—"
	}
	el := ui.Text(c, words).FontSize(t.Rem(remCaption)).TextColor(colour).
		Grow(1).MaxLines(1).Ellipsis("…")
	if bold {
		el.FontWeight(500)
	}
	return el
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

// trackList draws the songs, or the shape of them while they are read.
func (a *app) trackList(c *ui.Context) {
	t := c.Theme()
	if a.loading {
		ui.Column(c).FillWidth().Grow(1).Gap(t.Space(1)).Children(func() {
			for i := range 8 {
				a.skeletonRow(c, i)
			}
		})
		return
	}
	ui.List(c, &a.list, len(a.visible), func(i int) {
		a.trackTableRow(c, i)
	}).Grow(1).MinHeight(t.Space(40)).Label("Songs")
}

// readingSkeleton is the page before its tracks arrive: the head's shape and a few
// rows of the list, so the window says what is coming instead of showing an empty
// page with a spinner in it.
func (a *app) readingSkeleton(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).FillWidth().Grow(1).Gap(t.Space(7)).Children(func() {
		ui.Row(c).FillWidth().Gap(t.Space(6)).AlignItems(ui.End).Children(func() {
			a.skeletonBox(c, t.Space(48), t.Space(48), radiusCard(t))
			ui.Column(c).Grow(1).MinWidth(0).Gap(t.Space(2)).Children(func() {
				a.skeletonBox(c, t.Space(90), t.Rem(remDisplay)*leadDisplay, radiusControl(t))
				a.skeletonBox(c, t.Space(40), t.Rem(remCaption)*1.6, radiusControl(t))
			})
		})
		ui.Column(c).FillWidth().Grow(1).Gap(t.Space(1)).Children(func() {
			for i := range 8 {
				a.skeletonRow(c, i)
			}
		})
	})
}

// skeletonRow is the shape of a track row, pulsing so it reads as a wait rather
// than as an empty row.
func (a *app) skeletonRow(c *ui.Context, i int) {
	t := c.Theme()
	ui.Row(c).FillWidth().Gap(t.Space(3)).AlignItems(ui.Center).
		Padding(t.Space(1.5), t.Space(2)).Children(func() {
		a.skeletonBox(c, t.Space(9), t.Space(9), radiusControl(t))
		ui.Column(c).Grow(1).MinWidth(0).Gap(t.Space(1)).Children(func() {
			width := t.Space(60)
			if i%3 == 1 {
				width = t.Space(44)
			}
			a.skeletonBox(c, width, t.Space(3.5), radiusControl(t))
			a.skeletonBox(c, t.Space(28), t.Space(3), radiusControl(t))
		})
	})
}

// skeletonBox is one bar of a skeleton: a face the theme supplies, pulsing.
//
// The pulse is a Loop, which keeps frames coming while the wait lasts and keeps
// going when the desktop asks for less motion, because it shows that something is
// happening rather than moving anything.
func (a *app) skeletonBox(c *ui.Context, width, height, radius float32) ui.Element {
	s := surface(c)
	box := ui.Box(c).Width(width).Height(height).Shrink(0).Radius(radius).
		Background(s.sunken())
	pulse := box.Loop("pulse", 1400*time.Millisecond, ui.EaseInOut)
	box.Opacity(0.55 + 0.45*pulse)
	return box
}

// nowPlayingBars draws the mark of the track that plays: four bars that move while
// the music does, and stand still when it is held.
//
// The bars are four elements with a Loop each, which is the cheapest moving thing
// MyGo draws: no layout changes, no rebuild of anything else on the page.
func (a *app) nowPlayingBars(c *ui.Context, size float32, colour ui.Color) ui.Element {
	playing := a.isPlaying()
	return ui.Row(c).Width(size).Height(size).AlignItems(ui.Center).Gap(size * 0.12).
		Children(func() {
			for i := range 4 {
				bar := ui.Box(c).Width(size * 0.16).Radius(size * 0.08).
					Background(colour)
				height := size * 0.35
				if playing {
					// Each bar is on its own beat, so the group reads as sound
					// rather than as a pulse.
					phase := float32(i) * 0.17
					wave := bar.Loop("bars", time.Duration(760+int(phase*600))*time.Millisecond, ui.EaseInOut)
					height = size * (0.28 + 0.62*wave)
				}
				bar.Height(height)
			}
		})
}
