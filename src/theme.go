package src

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/theme"
)

// The look of the app is Onevra's, laid out by Apple's rules.
//
// Every colour is a role of Onevra's palette: the app adds no colour of its
// own, only the accent the user picks, laid over the palette. What the app
// owns is the space, the type and the motion, and those follow one scale:
// distances are multiples of the theme's spacing, sizes multiples of its font
// size, and corners multiples of its radius. Change the desktop's text size
// and the whole window follows.
//
// The type has four sizes and three weights, and nothing else. Names are
// semibold, labels medium, text regular; large text is tracked tight and small
// text open, because letters read further apart as they grow.

// The shapes. A card is the one rounded surface; a control is the smaller
// rounding inside it; a pill is round whatever its height.
func radiusCard(t *ui.Theme) float32    { return t.Radius * 2 }
func radiusControl(t *ui.Theme) float32 { return t.Radius }
func radiusPill(t *ui.Theme) float32    { return t.Radius * 100 }

// The sizes of the type, in multiples of the theme's font size: a display for
// the name of a page, a lead for the one name a page is about, a headline for
// the name of a group, a title for a track, a body for what a row says under
// its name, and a caption for what is beside it.
//
// Nothing is smaller than a caption: 0.86 of a 14 DIP base is 12 DIPs, and text
// under that is hard to read on a desktop at arm's length. The roles that carry
// the hierarchy are spaced widely enough to read as steps; the caption sits
// just under the body, as metadata does in any system.
const (
	remDisplay  = 2.05
	remLead     = 1.6
	remHeadline = 1.3
	remTitle    = 1.08
	remBody     = 0.93
	remCaption  = 0.86
)

// The tracking, in DIPs: negative as type grows, positive as it shrinks.
// A single value for every size is wrong somewhere, so there are two.
func trackDisplay(t *ui.Theme) float32  { return -0.022 * t.Rem(remDisplay) }
func trackHeadline(t *ui.Theme) float32 { return -0.012 * t.Rem(remHeadline) }

// The leading, which follows size the other way: tight on large text, open on
// what is read in lines.
const (
	leadDisplay  = 1.14
	leadHeadline = 1.24
	leadTitle    = 1.35
	leadBody     = 1.5
)

// The motion. The app moves what the user changes, and nothing else: a page
// arriving, a row under the pointer, a control pressed. All of it is a spring
// that can be interrupted, and all of it goes still when the desktop asks for
// less motion, which MyGo handles for us.
var (
	// motionReveal brings a page in from a little below, as it appears.
	motionReveal = ui.ElementTransition{
		Animation: &ui.Smooth,
		Enter:     &ui.Motion{Y: 6},
		Exit:      &ui.Motion{Y: -4},
	}
	// motionPress is the shortest change there is: a control settling into
	// the face it takes under the pointer.
	motionPress = ui.ElementTransition{Colors: true, Duration: 120 * time.Millisecond}
)

// theme returns the theme of the window: a palette, with the accent the user
// chose laid over it, in the app's typeface.
//
// The order matters. A palette sets the surfaces, the ink and its own accent;
// a colour the user picked, or one taken from the artwork, then replaces the
// accent alone. So the two settings compose: choose Violet, then take the
// accent from the cover, and the app keeps Violet's surfaces with the cover's
// colour on top.
func (a *app) theme(c *ui.Context) *ui.Theme {
	t := theme.Onevra(c)
	if interReady {
		t.Font = familySans
	}
	if p, ok := paletteByID(a.appearance.palette); ok {
		p.tint(t)
	}
	accent := a.accentColor(t.Dark)
	if accent == "" {
		return t
	}
	applyAccent(t, ui.Hex(accent), t.Dark)
	return t
}

// accentColor returns the colour of the accent: the one the user chose, or
// the one of the artwork of the track that plays.
func (a *app) accentColor(dark bool) string {
	if a.appearance.accent != "" {
		return a.appearance.accent
	}
	if !a.appearance.dynamic {
		return ""
	}
	track := a.currentTrack()
	if track.Thumbnail == "" {
		return ""
	}
	colour := a.art.color(track.Thumbnail)
	if colour == "" {
		return ""
	}
	red, green, blue := hexToRGB(colour)
	return fitAccent(red, green, blue, dark)
}

// applyAccent lays an accent over Onevra's palette, and derives what the
// palette derives from its own: the hover and pressed faces, the text on it,
// what is chosen, and the focus ring.
func applyAccent(t *ui.Theme, accent ui.Color, dark bool) {
	black := ui.RGB(0, 0, 0)
	white := ui.RGB(255, 255, 255)
	t.Accent = accent
	if dark {
		t.AccentHover = accent.Mix(white, 0.12)
		t.AccentPressed = accent.Mix(black, 0.25)
		t.AccentText = accent.Mix(black, 0.78)
	} else {
		t.AccentHover = accent.Mix(black, 0.12)
		t.AccentPressed = accent.Mix(black, 0.25)
		t.AccentText = ui.Hex("#ffffff")
	}
	t.Selection = accent.Alpha(0.22)
	t.Focus = accent.Alpha(0.8)
}

// hexToRGB reads the red, the green and the blue of a colour.
func hexToRGB(colour string) (red, green, blue int) {
	if len(colour) != 7 || colour[0] != '#' {
		return 0, 0, 0
	}
	value := 0
	for _, digit := range colour[1:] {
		value <<= 4
		switch {
		case digit >= '0' && digit <= '9':
			value |= int(digit - '0')
		case digit >= 'a' && digit <= 'f':
			value |= int(digit-'a') + 10
		case digit >= 'A' && digit <= 'F':
			value |= int(digit-'A') + 10
		default:
			return 0, 0, 0
		}
	}
	return value >> 16 & 0xff, value >> 8 & 0xff, value & 0xff
}
