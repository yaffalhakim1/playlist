package src

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/icons/lucide"
)

// settle is how long the seek slider rests before the player jumps there.
const settle = 200 * time.Millisecond

// playerbar draws the bar that floats at the bottom of the page: the track
// that plays, the controls, and the place in the track.
//
// It is the one surface that is always there, so it is drawn as a material
// over the page rather than a card in it: the content fades under it, and the
// bar itself is a capsule with the accent on the one control that matters.
// Its three parts sit on one line — what plays, the transport, and the sound —
// with the place in the track running under them.
func (a *app) playerbar(c *ui.Context) {
	s := surface(c)
	t := c.Theme()
	track := a.currentTrack()
	position, duration := a.times()
	height := playerBarOptions(c).Height

	bar := ui.Column(c).Absolute().Left(t.Space(4)).Right(t.Space(4)).
		Bottom(t.Space(4)).Height(height).Justify(ui.Center).
		Radius(radiusCard(t)).Background(s.raised()).
		Shadow(0, 8, 24, 0, s.shadow()).
		Padding(t.Space(2), t.Space(3)).Gap(t.Space(1.5))

	bar.Children(func() {
		ui.Row(c).FillWidth().Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
			// What plays, at the left, with its cover.
			ui.Row(c).Grow(1).MinWidth(0).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
				a.picture(c, track.Thumbnail, t.Space(11), radiusControl(t))
				ui.Column(c).Grow(1).MinWidth(0).Gap(t.Space(0.25)).Children(func() {
					if track.Title == "" {
						components.Text(c, "Nothing playing").FontSize(t.Rem(remTitle)).
							FontWeight(500).TextColor(s.muted())
						return
					}
					components.Text(c, track.Title).FontSize(t.Rem(remTitle)).
						FontWeight(500).LineHeight(leadTitle).MaxLines(1).Ellipsis("…")
					line := trackLine(track.Title, track.Artist, track.Album)
					if line != track.Title {
						caption(c, line).MaxLines(1).Ellipsis("…")
					}
				})
			})

			// The transport, in the middle, where a thumb expects it.
			a.transport(c, a.isPlaying(), false)

			// The sound, at the right.
			ui.Row(c).Grow(1).MinWidth(0).Justify(ui.End).Gap(t.Space(2)).
				AlignItems(ui.Center).Children(func() {
				a.volumeControl(c)
			})
		})

		// The place in the track, under the whole bar.
		ui.Row(c).FillWidth().Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			caption(c, clock(position)).Width(t.Space(10)).TextAlign(ui.End)
			a.seekSlider(c, duration)
			caption(c, clock(duration)).Width(t.Space(10))
		})
	})
}

// transport draws the buttons that play the music.
//
// The one control that changes the state of the app is the one in the accent;
// the rest are quiet, and say they are on by their colour rather than by a
// badge. With no queue, the buttons that move through it are unavailable
// rather than absent: the bar keeps its shape, and nothing does what it
// cannot.
func (a *app) transport(c *ui.Context, playing, big bool) {
	s := surface(c)
	t := c.Theme()
	size := t.Space(11)
	if big {
		size = t.Space(15)
	}
	empty := len(a.queue) == 0
	ui.Row(c).Gap(t.Space(1)).AlignItems(ui.Center).Shrink(0).Children(func() {
		shuffle := iconButton(c, lucide.Shuffle(), "Shuffle")
		if a.isShuffle() {
			shuffle.TextColor(s.accent())
		}
		if shuffle.Clicked() {
			a.toggleShuffle()
		}
		shuffle.Disabled(empty)

		previous := iconButton(c, lucide.SkipBack(), "Previous")
		if previous.Clicked() {
			a.engine.Previous()
		}
		previous.Disabled(empty)

		icon := lucide.Play()
		if playing {
			icon = lucide.Pause()
		}
		button := components.Button(c, "", components.ButtonPrimary).Size(size, size).
			Radius(size / 2).Transition(motionPress)
		button.Disabled(empty)
		button.Children(func() {
			// The glyph is nudged a hair right of center when it is a
			// triangle: a play mark looks left of center when it is centered.
			iconBox := ui.Box(c).Fill().Center()
			if !playing {
				iconBox.PaddingStart(size * 0.04)
			}
			iconBox.Children(func() {
				components.Icon(c, icon).Size(size*0.4, size*0.4).TextColor(t.AccentText)
			})
		})
		if button.Clicked() {
			a.engine.Toggle()
		}

		next := iconButton(c, lucide.SkipForward(), "Next")
		if next.Clicked() {
			a.engine.Next()
		}
		next.Disabled(empty)

		repeat := iconButton(c, lucide.Repeat(), "Repeat")
		if a.repeatMode() == 2 {
			repeat = iconButton(c, lucide.Repeat1(), "Repeat one")
		}
		if a.repeatMode() != 0 {
			repeat.TextColor(s.accent())
		}
		if repeat.Clicked() {
			a.cycleRepeat()
		}
		repeat.Disabled(empty)
	})
}

// seekSlider draws the bar that shows the place in the track, and lets the
// user jump.
func (a *app) seekSlider(c *ui.Context, duration time.Duration) {
	high := duration.Seconds()
	if high < 1 {
		high = 1
	}
	slider := components.Slider(c, &a.seekValue, 0, high)
	slider.Grow(1).Label("Position")
	if slider.Changed() {
		a.scrubbing = true
		a.scrubAt = time.Now()
		return
	}
	if a.scrubbing && time.Since(a.scrubAt) > settle {
		a.scrubbing = false
		a.engine.Seek(time.Duration(a.seekValue * float64(time.Second)))
	}
}

// volumeControl draws the speaker and the knob that goes with it.
//
// The speaker is the button: a click on it mutes the player, which is what the
// icon looks like it does.
func (a *app) volumeControl(c *ui.Context) {
	s := surface(c)
	t := c.Theme()
	volume := a.volumeLevel()
	icon := lucide.Volume2()
	switch {
	case volume <= 0:
		icon = lucide.VolumeX()
	case volume < 0.5:
		icon = lucide.Volume1()
	}
	button := ui.ButtonBase(c).Size(t.Space(8), t.Space(8)).Radius(radiusPill(t)).
		Center().Label("Mute").Transition(motionPress)
	if button.Hovered() {
		button.Background(s.card())
	}
	button.Children(func() {
		components.Icon(c, icon).Size(t.Space(4.5), t.Space(4.5)).TextColor(s.muted())
	})
	if button.Clicked() {
		a.volumeBy(-volume)
	}
	slider := components.Slider(c, &a.volume, 0, 1)
	slider.Width(t.Space(22)).Label("Volume")
	if slider.Changed() {
		if a.engine != nil {
			a.engine.SetVolume(a.volume)
		}
		a.rememberVolume(a.volume)
	}
}
