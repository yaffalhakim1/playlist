package src

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"youtube-playlist/internal/library"
)

// homePage draws the page one lands on: the link to play, the playlist that is
// open, and what was played before.
//
// The page is one column of cards on the paper: an import row, the hero of the
// playlist that is open, its tracks, the playlists played before, and a shelf of
// them to pick up again. Every section is real: the hero is the playlist the app
// has loaded, the tracklist is its queue, and the shelf is what the app
// remembers.
func (a *app) homePage(c *ui.Context) {
	t := c.Theme()
	a.pageFrame(c, true, func() {
		ui.Column(c).FillWidth().Gap(t.Space(8)).Children(func() {
			a.importRow(c)
			if len(a.queue) > 0 {
				a.homeHero(c)
				a.homeTracks(c)
			}
			if len(a.recents) > 0 {
				a.homeRecents(c)
				a.homeShelf(c)
			}
		})
	})
}

// importRow is the design's link bar: the field, the button that reads it, and a
// clear button while it holds anything.
//
// It has three states and shows all of them: empty, it invites a paste; holding a
// link, it offers Play; and while a playlist is read, the button says so and
// cannot be pressed again.
func (a *app) importRow(c *ui.Context) {
	s := surface(c)
	t := c.Theme()
	ui.Column(c).FillWidth().Gap(t.Space(2)).Children(func() {
		ui.Row(c).FillWidth().Gap(t.Space(2)).Padding(t.Space(2.5)).
			Radius(radiusCard(t)).Background(s.card()).Border(1, s.hairline()).
			Shadow(0, 1, 4, 0, s.shadow()).AlignItems(ui.Center).Children(func() {
			ui.Row(c).Grow(1).MinWidth(0).Gap(t.Space(2.5)).AlignItems(ui.Center).
				Padding(0, 0, 0, t.Space(3)).Children(func() {
				ui.Icon(c, iconLink).Size(t.Space(4), t.Space(4)).TextColor(s.faint())
				field := ui.TextInput(c, &a.link).Grow(1).Border(0, ui.Transparent).
					Background(ui.Transparent).Padding(0).FontSize(t.Rem(remCaption)).
					Label("Playlist link").
					Placeholder("Paste a YouTube playlist or radio link…")
				field.Disabled(a.loading)
				if field.Submitted() {
					a.load()
				}
				if !a.linkEmpty() && !a.loading {
					if b := iconButton(c, iconClose, "Clear"); b.Clicked() {
						a.link = ""
					}
				}
			})
			a.importAction(c)
		})
		a.loadLine(c)
	})
}

// loadLine is the one line under the field: what is being read, or what went
// wrong. The two never speak together, because a load in flight clears the status
// and a failure stops the spinner.
func (a *app) loadLine(c *ui.Context) {
	s := surface(c)
	t := c.Theme()
	switch {
	case a.loading:
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Spinner(c).Size(t.Space(3.5), t.Space(3.5)).TextColor(s.muted())
			ui.Text(c, "Reading the playlist. A long one takes a few seconds.").
				FontSize(t.Rem(remCaption)).TextColor(s.muted())
		})
	case a.status != "":
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconWarning).Size(t.Space(4), t.Space(4)).TextColor(s.danger())
			ui.Text(c, a.status).FontSize(t.Rem(remCaption)).
				TextColor(s.danger()).MaxLines(2).Ellipsis("…")
		})
	}
}

// importAction is the button beside the field, in one of three states.
//
// Building the first and then replacing the variable would leave both in the row:
// the element is already a child by the time it is reassigned.
func (a *app) importAction(c *ui.Context) {
	switch {
	case a.loading:
		busyButton(c, "Reading the playlist…")
	case a.linkEmpty():
		if b := primaryButton(c, iconSparkle, "Extract Playlist"); b.Clicked() {
			a.pasteLink()
		}
	default:
		if b := primaryButton(c, iconSparkle, "Extract Playlist"); b.Clicked() {
			a.load()
		}
	}
}

// homeHero is the playlist that is open: its cover, what it is, and what can be
// done with it.
func (a *app) homeHero(c *ui.Context) {
	s := surface(c)
	t := c.Theme()
	title := "Playlist"
	var thumbnail string
	if a.playlist != nil {
		title = a.playlist.Title
		thumbnail = a.playlist.Thumbnail
	}
	panel(c, func() {
		ui.Row(c).FillWidth().Gap(t.Space(6)).AlignItems(ui.End).Children(func() {
			a.picture(c, thumbnail, t.Space(48), radiusCard(t))
			ui.Column(c).Grow(1).MinWidth(0).Gap(t.Space(2)).Children(func() {
				ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
					eyebrow(c, "Now playing from")
					ui.Text(c, "•").FontSize(t.Rem(remCaption)).TextColor(s.faint())
					if b := iconButton(c, iconMore, "More"); b.Clicked() {
						a.page = pageQueue
					}
				})
				ui.Text(c, title).FontSize(t.Rem(remDisplay)).FontWeight(700).
					LetterSpacing(trackDisplay(t)).LineHeight(leadDisplay).
					MaxLines(2).Ellipsis("…").TextColor(s.text())
				caption(c, fmt.Sprintf("%d tracks · %s · %s",
					len(a.queue), totalLength(a.queue), a.nowPlayingLine()))

				ui.Row(c).Gap(t.Space(2.5)).AlignItems(ui.Center).
					Padding(t.Space(2), 0, 0, 0).Children(func() {
					a.playToggle(c, t.Space(9))
					if b := secondaryButton(c, nil, "Open queue"); b.Clicked() {
						a.page = pageQueue
					}
					if b := secondaryButton(c, iconExport, "Start a mix"); b.Clicked() {
						a.playMix()
					}
				})
			})
		})
	})
}

// nowPlayingLine names what plays, for the hero's caption.
func (a *app) nowPlayingLine() string {
	track := a.currentTrack()
	if track.Title == "" {
		return "nothing playing"
	}
	if track.Artist != "" {
		return track.Artist
	}
	return "playing"
}

// playToggle is the round filled button that plays or holds the queue.
func (a *app) playToggle(c *ui.Context, size float32) {
	s := surface(c)
	t := c.Theme()
	playing := a.isPlaying()
	b := ui.ButtonBase(c).Size(size, size).Radius(size / 2).Center().
		Label("Play or hold")
	face := s.ink()
	if b.Hovered() {
		face = t.AccentHover
	}
	if b.Pressed() {
		face = t.AccentPressed
	}
	b.Background(b.AnimateColor("face", face, motionColor))
	b.AnimatePose(pressPose(b.Pressed()), motionPress)
	b.Disabled(len(a.queue) == 0)
	b.Children(func() {
		mark := iconPlay
		if playing {
			mark = iconPause
		}
		ui.Icon(c, mark).Size(size*0.42, size*0.42).TextColor(s.onInk())
	})
	if b.Clicked() {
		a.engine.Toggle()
	}
}

// homeTracks is the playlist's tracks as a table: a title column, the artist, the
// album, and the length at the right.
//
// It is the design's table, and it is the queue: the same rows the Queue page
// shows, cut to what fits on the page so the home page stays one screen of
// scanning.
func (a *app) homeTracks(c *ui.Context) {
	s := surface(c)
	t := c.Theme()
	shown := min(len(a.visible), homeTrackLimit)
	panel(c, func() {
		ui.Column(c).FillWidth().Gap(0).Children(func() {
			// The header row.
			ui.Row(c).FillWidth().Gap(t.Space(4)).Padding(t.Space(3), t.Space(4)).
				BorderWidth(0, 0, 1, 0).BorderColor(s.hairline()).Children(func() {
				trackHead(c, "Title", true)
				trackHead(c, "Artist", false)
				trackHead(c, "Album / Tape", false)
				// The heading sits in the same box as the row's clock, so the
				// two line up. 16 units is 64 DIPs: wide enough for the word
				// at caption size, where 12 broke it into "Duratio n".
				ui.Box(c).Width(t.Space(16)).Shrink(0).Children(func() {
					ui.Text(c, "Duration").FontSize(t.Rem(remCaption)).FontWeight(600).
						LetterSpacing(0.06 * t.Rem(remCaption)).TextColor(s.muted()).
						TextAlign(ui.End).Grow(1)
				})
			})
			for i := range shown {
				a.trackTableRow(c, i)
			}
			if len(a.visible) > shown {
				ui.Row(c).FillWidth().Padding(t.Space(2.5), t.Space(4)).Children(func() {
					ui.Spacer(c)
					if b := quietButton(c, fmt.Sprintf("All %d tracks", len(a.visible))); b.Clicked() {
						a.page = pageQueue
					}
				})
			}
		})
	})
}

// homeTrackLimit is how many rows the home page's table shows before it points at
// the Queue page for the rest.
//
// Four, not six: the home page is a glance at the playlist, and the rows here are
// the same height as the Queue page's — six of them pushed the link to the rest off
// the bottom of the window.
const homeTrackLimit = 4

// trackHead is one column heading of the table.
func trackHead(c *ui.Context, label string, grow bool) ui.Element {
	t := c.Theme()
	head := ui.Text(c, label).FontSize(t.Rem(remCaption)).FontWeight(600).
		LetterSpacing(0.06 * t.Rem(remCaption)).TextColor(surface(c).muted())
	if grow {
		return head.Grow(1)
	}
	return head.Grow(1)
}

// homeRecents is the shelf of playlists that were played before, as covers.
func (a *app) homeRecents(c *ui.Context) {
	t := c.Theme()
	cover := t.Space(36)
	ui.Column(c).FillWidth().Gap(t.Space(3)).Children(func() {
		sectionHead(c, "Recently played", "Pick up where you left off.", func() {
			if b := quietButton(c, "View all"); b.Clicked() {
				a.page = pageQueue
			}
		})
		ui.ScrollHorizontal(c).FillWidth().Height(cover + t.Space(14)).
			ScrollbarInsets(0, 0, t.Space(2), 0).Children(func() {
			ui.Row(c).Gap(t.Space(3)).Children(func() {
				for _, recent := range a.recents {
					a.recentCard(c, recent, cover)
				}
			})
		})
	})
}

// homeShelf is the design's "recently extracted" row: three wide cards with a
// mark, the name and what it holds, and a round button that plays it.
func (a *app) homeShelf(c *ui.Context) {
	t := c.Theme()
	shown := min(len(a.recents), 3)
	ui.Column(c).FillWidth().Gap(t.Space(3)).Children(func() {
		sectionHead(c, "Your playlists", "Opened from the cache, without touching the network.", nil)
		ui.Row(c).FillWidth().Gap(t.Space(3)).Children(func() {
			for i := range shown {
				a.shelfCard(c, a.recents[i])
			}
		})
	})
}

// shelfCard is one wide card of the shelf.
func (a *app) shelfCard(c *ui.Context, recent library.Recent) {
	s := surface(c)
	t := c.Theme()
	title := recent.Title
	if title == "" {
		title = recent.Link
	}
	ui.Box(c).Grow(1).MinWidth(0).Padding(t.Space(3)).Radius(radiusCard(t)).
		Background(s.card()).Border(1, s.hairline()).AlignItems(ui.Center).
		Children(func() {
		ui.Row(c).FillWidth().Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Size(t.Space(14), t.Space(14)).Shrink(0).Radius(radiusControl(t)).
				Background(s.sunken()).Border(1, s.hairline()).Center().Children(func() {
				ui.Icon(c, iconGraphicEq).Size(t.Space(6), t.Space(6)).TextColor(s.muted())
			})
			ui.Column(c).Grow(1).MinWidth(0).Gap(t.Space(0.5)).Children(func() {
				mono(c, "Playlist")
				ui.Text(c, title).FontSize(t.Rem(remCaption)).FontWeight(600).
					TextColor(s.text()).MaxLines(1).Ellipsis("…")
			})
			if b := roundButton(c, iconPlay, "Open "+title); b.Clicked() {
				a.openRecent(recent.Link)
			}
		})
	})
}

// recentCard draws one playlist of the shelf, as a cover with its name under it.
func (a *app) recentCard(c *ui.Context, recent library.Recent, width float32) {
	s := surface(c)
	t := c.Theme()
	title := recent.Title
	if title == "" {
		title = recent.Link
	}
	button := ui.ButtonBase(c).Key("recent-card-"+recent.Link).Width(width).Shrink(0).
		Padding(t.Space(2)).Radius(radiusCard(t)).Gap(t.Space(2))
	face := ui.Transparent
	if button.Hovered() {
		face = s.card()
	}
	button.Background(button.AnimateColor("face", face, motionColor))
	button.Children(func() {
		ui.Column(c).FillWidth().Gap(t.Space(2)).Children(func() {
			a.picture(c, recent.Thumbnail, width-t.Space(4), radiusControl(t))
			ui.Column(c).FillWidth().Gap(t.Space(0.25)).Children(func() {
				ui.Text(c, title).FontSize(t.Rem(remCaption)).FontWeight(600).
					TextColor(s.text()).MaxLines(1).Ellipsis("…")
				mono(c, "Playlist")
			})
		})
	})
	// Translation only, and only because the whole tile is a button: a Scale
	// would resample the words under the cover.
	button.AnimatePose(risePose(button.Hovered()), motionRise)
	if button.Clicked() {
		a.openRecent(recent.Link)
	}
}

// roundButton is the design's small circular action: a hairline ring with one
// mark in it.
func roundButton(c *ui.Context, mark *ui.SVG, label string) ui.Element {
	s := surface(c)
	t := c.Theme()
	size := t.Space(8)
	b := ui.ButtonBase(c).Size(size, size).Radius(size / 2).Center().
		Label(label).Tooltip(label).Border(1, s.hairline())
	face := ui.Transparent
	if b.Hovered() {
		face = s.sunken()
	}
	b.Background(b.AnimateColor("face", face, motionColor))
	b.AnimatePose(pressPose(b.Pressed()), motionPress)
	b.Children(func() {
		ui.Icon(c, mark).Size(t.Space(4), t.Space(4)).TextColor(s.text())
	})
	return b
}

// playlistTitle returns the name of the playlist that plays.
func (a *app) playlistTitle() string {
	if a.playlist == nil {
		return ""
	}
	return a.playlist.Title
}

// linkEmpty tells whether the field holds nothing to play.
func (a *app) linkEmpty() bool { return strings.TrimSpace(a.link) == "" }
