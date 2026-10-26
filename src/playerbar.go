package src

import (
	"time"

	"github.com/egoist/mygo/ui"

	"youtube-playlist/internal/music"
)

// settle is how long the seek slider rests before the player jumps there.
const settle = 200 * time.Millisecond

// playerbar draws the footer: the place in the track across the top, and one row
// under it with what plays at the left, the transport in the middle, and the
// output at the right.
//
// It is the one surface that is always there, so it is drawn as the window's own
// chrome rather than as a card in the page: a hairline along the top, the paper's
// face behind it, and the page scrolling under neither — `pageFrame` keeps the
// page clear of it.
//
// It is also the only place the app says what the playback is doing: a track
// being found, a track that failed, and a track that plays all look different
// here, so the state is never guessed from whether the window seems frozen.
func (a *app) playerbar(c *ui.Context) {
	s := surface(c)
	t := c.Theme()
	track := a.currentTrack()
	position, duration := a.times()
	options := playerBarOptions(c)

	bar := ui.Column(c).Absolute().Left(0).Right(0).Bottom(0).
		Height(options.Height).Justify(ui.Center).
		Background(s.card()).BorderWidth(1, 0, 0, 0).BorderColor(s.hairline()).
		Padding(0, pageInset(c)).Gap(t.Space(0.5))

	bar.Children(func() {
		// The place in the track, across the whole footer.
		ui.Row(c).FillWidth().Gap(t.Space(2.5)).AlignItems(ui.Center).Children(func() {
			mono(c, clock(position))
			a.seekSlider(c, duration)
			mono(c, clock(duration))
		})

		ui.Row(c).FillWidth().Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
			// What plays, at the left, with its cover.
			ui.Row(c).Grow(1).MinWidth(0).Gap(t.Space(2.5)).AlignItems(ui.Center).Children(func() {
				a.picture(c, track.Thumbnail, t.Space(10), radiusControl(t))
				ui.Column(c).Grow(1).MinWidth(0).Gap(t.Space(0.25)).Children(func() {
					a.barTitle(c, track)
				})
			})

			// The transport, in the middle, where a thumb expects it.
			a.transport(c, a.isPlaying(), false)

			// The sound, at the right.
			ui.Row(c).Grow(1).MinWidth(0).Justify(ui.End).Gap(t.Space(1)).
				AlignItems(ui.Center).Children(func() {
				a.volumeControl(c)
			})
		})
	})
}

// barTitle says what plays, and what is happening to it: finding the stream,
// playing, held, or failed with a way to try again.
func (a *app) barTitle(c *ui.Context, track music.Track) {
	s := surface(c)
	t := c.Theme()
	if track.Title == "" {
		ui.Text(c, "Nothing playing").FontSize(t.Rem(remCaption)).
			TextColor(s.muted())
		return
	}
	ui.Text(c, track.Title).FontSize(t.Rem(remCaption)).FontWeight(600).
		TextColor(s.text()).MaxLines(1).Ellipsis("…")
	switch {
	case a.isBuffering():
		ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
			ui.Spinner(c).Size(t.Space(3), t.Space(3)).TextColor(s.muted())
			mono(c, "Finding the track…")
		})
	case a.status != "" && !a.isPlaying():
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			mono(c, "That track will not play")
			if retry := quietButton(c, "Try again"); retry.Clicked() {
				a.retryTrack()
			}
		})
	default:
		line := trackLine(track.Title, track.Artist, track.Album)
		if line != track.Title {
			ui.Text(c, line).FontSize(t.Rem(remCaption)).TextColor(s.muted()).
				MaxLines(1).Ellipsis("…")
		}
	}
}

// transport draws the buttons that play the music.
//
// The one control that changes the state of the app is the round button in the
// middle; the rest are quiet, and say they are on by their colour rather than by a
// badge. With no queue, the buttons that move through it are unavailable rather
// than absent: the bar keeps its shape, and nothing does what it cannot.
//
// Two things here are not obvious and were both wrong once:
//
//   - **The icon is chosen before the button is built, never by rebuilding it.**
//     An element is already a child of its row by the time the constructor returns,
//     so `repeat = iconButton(...)` inside a second branch draws a *second* button
//     beside the first: clicking Repeat used to add a Repeat-one button rather than
//     changing the one there.
//   - **A toggle's colour goes into the button's icon, not onto the button.** A
//     colour set on the element above a child does not reach the child, so
//     `shuffle.TextColor(...)` left the mark grey and the toggle never looked on.
func (a *app) transport(c *ui.Context, playing, big bool) {
	s := surface(c)
	t := c.Theme()
	size := t.Space(8)
	if big {
		size = t.Space(11)
	}
	empty := len(a.queue) == 0
	ui.Row(c).Gap(t.Space(1)).AlignItems(ui.Center).Shrink(0).Children(func() {
		// Shuffle: one glyph, lit when the queue plays in a random order.
		shuffle := iconButtonIn(c, iconShuffle, "Shuffle", toggleInk(s, a.isShuffle()))
		if shuffle.Clicked() {
			a.toggleShuffle()
		}
		shuffle.Disabled(empty)

		previous := iconButton(c, iconPrev, "Previous")
		if previous.Clicked() {
			a.engine.Previous()
		}
		previous.Disabled(empty)

		a.playButtonLarge(c, playing, size, empty)

		next := iconButton(c, iconNext, "Next")
		if next.Clicked() {
			a.engine.Next()
		}
		next.Disabled(empty)

		// Repeat: off, the queue again, or the one track again. The three are one
		// button with three faces, as a player's are — the glyph and its name are
		// chosen before the button exists.
		mark, label := iconRepeat, "Repeat"
		switch a.repeatMode() {
		case 1:
			label = "Repeat the queue"
		case 2:
			mark, label = iconRepeat1, "Repeat this track"
		}
		repeat := iconButtonIn(c, mark, label, toggleInk(s, a.repeatMode() != 0))
		if repeat.Clicked() {
			a.cycleRepeat()
		}
		repeat.Disabled(empty)
	})
}

// toggleInk is the colour a toggle's mark takes: the ink when it is on, the muted
// grey when it is off.
func toggleInk(s surfaces, on bool) ui.Color {
	if on {
		return s.text()
	}
	return s.muted()
}

// playButtonLarge is the one control of the app that is filled: play, or hold.
//
// The glyph swaps with a crossfade, which only paints, so the button answers a
// press at the rate the display can show it.
func (a *app) playButtonLarge(c *ui.Context, playing bool, size float32, empty bool) {
	s := surface(c)
	button := ui.ButtonBase(c).Size(size, size).Radius(size / 2).Center().
		Label("Play or hold").Border(1, s.hairline())
	face := s.card()
	if button.Hovered() {
		face = s.sunken()
	}
	if button.Pressed() {
		face = s.sunken().Mix(s.text(), 0.06)
	}
	button.Background(button.AnimateColor("face", face, motionColor))
	button.AnimatePose(pressPose(button.Pressed()), motionPress)
	button.Disabled(empty)
	button.Children(func() {
		// The glyph is nudged a hair right of center when it is a triangle: a
		// play mark looks left of center when it is centered.
		iconBox := ui.Box(c).Fill().Center()
		if !playing {
			iconBox.PaddingStart(size * 0.04)
		}
		iconBox.Children(func() {
			mark := iconPlay
			if playing {
				mark = iconPause
			}
			glyph := ui.Icon(c, mark).Size(size*0.42, size*0.42).TextColor(s.text())
			// The glyph fades rather than scales: a scaled icon is a resampled
			// icon, and on a 24-unit outline that reads as mush.
			glyph.Transition(ui.ElementTransition{Animation: &motionFade, Enter: &ui.Motion{Opacity: 0}})
		})
	})
	if button.Clicked() {
		a.engine.Toggle()
	}
}

// seekSlider draws the bar that shows the place in the track, and lets the user
// jump.
func (a *app) seekSlider(c *ui.Context, duration time.Duration) {
	high := duration.Seconds()
	if high < 1 {
		high = 1
	}
	slider := ui.Slider(c, &a.seekValue, 0, high)
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
	mark := iconVolume
	switch {
	case volume <= 0:
		mark = iconVolumeX
	case volume < 0.5:
		mark = iconVolume1
	}
	button := ui.ButtonBase(c).Size(t.Space(8), t.Space(8)).Radius(radiusControl(t)).
		Center().Label("Mute").Tooltip("Mute")
	face := ui.Transparent
	if button.Hovered() {
		face = s.sunken()
	}
	button.Background(button.AnimateColor("face", face, motionColor))
	button.AnimatePose(pressPose(button.Pressed()), motionPress)
	button.Children(func() {
		ui.Icon(c, mark).Size(t.Space(4), t.Space(4)).TextColor(s.muted())
	})
	if button.Clicked() {
		a.volumeBy(-volume)
	}
	slider := ui.Slider(c, &a.volume, 0, 1)
	slider.Width(t.Space(22)).Label("Volume")
	if slider.Changed() {
		if a.engine != nil {
			a.engine.SetVolume(a.volume)
		}
		a.rememberVolume(a.volume)
	}
}
