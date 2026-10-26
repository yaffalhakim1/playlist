package src

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// The look of the app: warm paper, near-black ink, and one hairline.
//
// The scheme is monochrome on purpose. There is no accent colour and no palette
// picker: the window is paper and ink, and the only colour on it is the artwork
// of the music. That is what makes a list of covers read as the subject rather
// than as decoration beside a coloured chrome.
//
// Every face below is a role of MyGo's theme, so the framework's own widgets —
// sliders, switches, text inputs, scroll bars — are in the app's colours without
// being restyled one by one.

// The shapes. A card is the one rounded surface; a control is the smaller
// rounding inside it; a pill is round whatever its height.
func radiusCard(t *ui.Theme) float32    { return t.Spacing * 4 }
func radiusControl(t *ui.Theme) float32 { return t.Spacing * 2 }
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

// The motion.
//
// The rule that makes this app feel fast on a fast display: **move what only
// needs paint.** An element moved with AnimatePose keeps going in frames that
// repaint without rebuilding the view, so the cost of a frame is a transform
// and not a layout pass. A pose that only translates is folded by MyGo into
// untransformed content, so the words inside it stay sharp; a pose with a Scale
// is a real transform and resamples them.
var (
	// motionReveal brings a page in from a little below, as it appears.
	motionReveal = ui.ElementTransition{
		Animation: &ui.Smooth,
		Enter:     &ui.Motion{Y: 6},
		Exit:      &ui.Motion{Y: -4},
	}
	// motionPress is a control settling under the pointer, and back out again.
	motionPress = ui.Animation{Spring: ui.SpringOptions{Duration: 160 * time.Millisecond, Bounce: 0.2}}
	// motionRise is a tile that is itself a link, lifting a little.
	motionRise = ui.Animation{Spring: ui.SpringOptions{Duration: 220 * time.Millisecond, Bounce: 0.28}}
	// motionColor recolours a face: 120 ms is long enough not to flicker and
	// short enough that a fast pointer never sees the end of it.
	motionColor = ui.Animation{Duration: 120 * time.Millisecond, Ease: ui.EaseOut}
	// motionFade crossfades one picture into another.
	motionFade = ui.Animation{Duration: 240 * time.Millisecond, Ease: ui.EaseInOut}
)

// The colours of the paper.
//
// Taken from the design: a warm off-white page, white cards, a hairline that is
// barely there, near-black ink and a grey for what is read second. The dark
// appearance is the same scheme turned over, so nothing changes but the light.
const (
	paperPage   = "#FAF9F5"
	paperCard   = "#FFFFFF"
	paperRail   = "#F5F4F0"
	paperSunken = "#EFEEEA"
	paperLine   = "#EAE8E3"
	paperInk    = "#1B1C1A"
	paperMuted  = "#787774"
	paperFaint  = "#B8B7B2"
	paperAction = "#1C1C1A"

	inkPage   = "#131312"
	inkCard   = "#1C1C1B"
	inkRail   = "#171716"
	inkSunken = "#242322"
	inkLine   = "#2E2D2B"
	inkText   = "#EDEBE6"
	inkMuted  = "#9A9791"
	inkFaint  = "#6E6C68"
	inkAction = "#EDEBE6"
)

// theme returns the theme of the window: the app's paper, in the appearance the
// user chose, in the app's typeface, with the accent the user picked.
//
// The order matters. MyGo's light or dark theme sets every role, including the
// ones the desktop's preferences adjust; the paper then replaces the faces and the
// ink; the accent then replaces the one colour the app has. So the two compose:
// pick Sage, then let the artwork cast its hue, and the window keeps the paper
// with the cover's colour on top.
func (a *app) theme(c *ui.Context) *ui.Theme {
	dark := c.Theme() != nil && c.Theme().Dark
	t := ui.LightTheme()
	if dark {
		t = ui.DarkTheme()
	}
	// MyGo's own theme follows the desktop's text scale; a copy keeps that,
	// because the paper below only touches colour.
	if p := c.Preferences(); p.TextScale > 0 && p.TextScale != 1 {
		t.FontSize *= p.TextScale
	}
	if jakartaReady {
		t.Font = familySans
	}
	paper(t, dark)
	accent := a.accentColor(dark)
	if accent != "" {
		applyAccent(t, ui.Hex(accent), dark)
	}
	if p := c.Preferences(); p.HighContrast {
		t.Border = t.Border.Mix(t.Text, 0.45)
		t.TextMuted = t.TextMuted.Mix(t.Text, 0.4)
		t.Focus = t.Focus.Alpha(1)
	}
	return t
}

// accentColor returns the colour of the accent: the one the user chose, or the one
// the artwork of the track that plays casts, or empty for the paper's own ink.
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

// applyAccent lays an accent over the paper, and derives what a filled control
// needs from it: the hover and pressed faces, the text on it, what is chosen, and
// the focus ring. Nothing here is written by hand, so any accent composes.
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
		t.AccentText = white
	}
	t.Selection = accent.Alpha(0.10)
	t.Focus = accent.Alpha(0.5)
}

// paper lays the scheme over a theme.
func paper(t *ui.Theme, dark bool) {
	if dark {
		t.Background = ui.Hex(inkPage)
		t.Surface = ui.Hex(inkCard)
		t.SurfaceHover = ui.Hex(inkRail)
		t.SurfacePressed = ui.Hex(inkSunken)
		t.Border = ui.Hex(inkLine)
		t.Text = ui.Hex(inkText)
		t.TextMuted = ui.Hex(inkMuted)
		t.Accent = ui.Hex(inkAction)
		t.AccentHover = ui.RGB(255, 255, 255)
		t.AccentPressed = ui.RGB(214, 211, 205)
		t.AccentText = ui.Hex(inkPage)
		t.Selection = ui.Hex(inkText).Alpha(0.09)
		t.Focus = ui.Hex(inkText).Alpha(0.55)
		t.Danger = ui.Hex("#FFB4AB")
		t.Scrollbar = ui.Hex(inkText).Alpha(0.18)
		return
	}
	t.Background = ui.Hex(paperPage)
	t.Surface = ui.Hex(paperCard)
	t.SurfaceHover = ui.Hex(paperRail)
	t.SurfacePressed = ui.Hex(paperSunken)
	t.Border = ui.Hex(paperLine)
	t.Text = ui.Hex(paperInk)
	t.TextMuted = ui.Hex(paperMuted)
	// The accent is the near-black of a filled button, not a hue: the only
	// colour in the window comes from the artwork.
	t.Accent = ui.Hex(paperAction)
	t.AccentHover = ui.Hex("#30312E")
	t.AccentPressed = ui.Hex("#000000")
	t.AccentText = ui.Hex("#FFFFFF")
	t.Selection = ui.Hex(paperInk).Alpha(0.07)
	t.Focus = ui.Hex(paperInk).Alpha(0.45)
	t.Danger = ui.Hex("#BA1A1A")
	t.Scrollbar = ui.Hex(paperInk).Alpha(0.16)
}

// railFace is the face of the sidebar, a step off the page so the rail reads as
// chrome rather than as part of the page.
func railFace(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.Hex(inkRail)
	}
	return ui.Hex(paperRail)
}

// faint is the colour of what is barely there: a scrollbar, a disabled mark, the
// hairline inside a card.
func faint(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.Hex(inkFaint)
	}
	return ui.Hex(paperFaint)
}
