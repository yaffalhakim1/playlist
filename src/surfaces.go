package src

import "github.com/egoist/mygo/ui"

// surfaces names the few faces the app paints, in the terms of the theme.
// Nothing here invents a colour: every field is a role of the theme, so the
// window follows the paper, its appearance, and nothing else.
//
// The app uses three faces and two lines. The page is the paper, a group of
// things is a card on it, and what sits inside a card — a well, a chip, a
// placeholder — is a step darker. There is no accent face, because there is no
// accent: the only colour in the window is the artwork.
type surfaces struct {
	theme *ui.Theme
}

// surface returns the faces of the frame.
func surface(c *ui.Context) surfaces { return surfaces{theme: c.Theme()} }

// card is the face of a card: a group of things, a table, a hero.
func (s surfaces) card() ui.Color { return s.theme.Surface }

// raised is a step off the card, for a chip that is chosen or a control under
// the pointer.
func (s surfaces) raised() ui.Color { return s.theme.SurfaceHover }

// sunken is what sits inside a card: a well, a placeholder, a hairline mark.
func (s surfaces) sunken() ui.Color { return s.theme.SurfacePressed }

// rail is the sidebar's own face, a step off the page so the rail reads as
// chrome rather than as part of the page.
func (s surfaces) rail() ui.Color { return railFace(s.theme) }

// text is the ink of what is read first.
func (s surfaces) text() ui.Color { return s.theme.Text }

// muted is the colour of what a row says under its name.
func (s surfaces) muted() ui.Color { return s.theme.TextMuted }

// faint is the colour of what is barely there: a scrollbar, a disabled mark.
func (s surfaces) faint() ui.Color { return faint(s.theme) }

// chosen is the face of the track that plays, and of a row under the pointer.
func (s surfaces) chosen() ui.Color { return s.theme.Selection }

// hairline is the line the design draws around every card and between rows.
func (s surfaces) hairline() ui.Color { return s.theme.Border }

// ink is the face of a filled control: the near-black of the primary button.
func (s surfaces) ink() ui.Color { return s.theme.Accent }

// onInk is what is written on it.
func (s surfaces) onInk() ui.Color { return s.theme.AccentText }

// danger is the colour of what went wrong.
func (s surfaces) danger() ui.Color { return s.theme.Danger }

// shadow is the colour a surface casts: the text's own, very faint, so a card
// lifts off the page without drawing a grey halo.
func (s surfaces) shadow() ui.Color { return s.theme.Text.Alpha(0.04) }
