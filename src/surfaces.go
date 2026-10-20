package src

import "github.com/egoist/mygo/ui"

// surfaces names the few faces the app paints, in the terms of Onevra's
// theme. Nothing here invents a colour: every field is a role of the theme, so
// the window follows Onevra's palette, its appearance and the accent over it.
//
// The app uses three faces and one line. The page is the background, a group
// of things is a card on the surface, and what floats over the page is the
// hovered surface, a shade apart so it reads as nearer without a border.
type surfaces struct {
	theme *ui.Theme
}

// surface returns the faces of the frame.
func surface(c *ui.Context) surfaces { return surfaces{theme: c.Theme()} }

// card is the face of a group that holds settings, a chart or a row of
// pictures.
func (s surfaces) card() ui.Color { return s.theme.Surface }

// raised is the face of what floats over the page, as the player bar does.
func (s surfaces) raised() ui.Color { return s.theme.SurfaceHover }

// sunken is the face of what sits inside a card: a well for a chart, a mark
// that a number sits in.
func (s surfaces) sunken() ui.Color { return s.theme.Background }

// muted is the colour of what a row says under its name.
func (s surfaces) muted() ui.Color { return s.theme.TextMuted }

// chosen is the face of what is chosen, and of the track that plays.
func (s surfaces) chosen() ui.Color { return s.theme.Selection }

// hairline is a line light enough to sit inside a card without cutting it.
func (s surfaces) hairline() ui.Color { return s.theme.Border.Alpha(0.6) }

// accent is the colour of the accent.
func (s surfaces) accent() ui.Color { return s.theme.Accent }

// shadow is the colour a surface casts: the text's own, very faint, so a card
// lifts off the page without drawing a grey halo.
func (s surfaces) shadow() ui.Color { return s.theme.Text.Alpha(0.06) }
