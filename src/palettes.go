package src

import (
	"github.com/egoist/mygo/ui"
)

// A palette is a Material 3 colour scheme, reduced to what this app actually
// paints.
//
// The app derives almost everything from one accent, so a palette is mostly a
// choice of accent plus the three surface roles the design uses: the page, the
// card, and the line. That is why a whole Material 3 scheme fits in six
// values, and why adding one does not touch a single page.
//
// The values are Material 3's own tones: a primary at tone 40 in the light and
// 80 in the dark, surfaces at 98 and 6, and a container at 90 and 30. A palette
// that broke those would stop looking like Material 3 the moment it was used.
type palette struct {
	// id is what the settings store keeps.
	id string
	// name is what the Look page shows.
	name string
	// light and dark are the two schemes of the same colour.
	light scheme
	dark  scheme
}

// scheme is one appearance of a palette.
type scheme struct {
	// accent is Material 3's primary.
	accent string
	// surface is the page, and raised is the card.
	surface string
	raised  string
	// text and muted are the two inks.
	text  string
	muted string
}

// palettes are the Material 3 schemes the Look page offers.
//
// Onevra's own terracotta is first, because it is what the app looked like
// before this list existed, and a palette called "Default" that is not the
// default is a small lie.
var palettes = []palette{
	{
		id:   "",
		name: "Onevra",
		light: scheme{
			accent: "#bd481b", surface: "#fcfbf9", raised: "#fefefd",
			text: "#33332f", muted: "#72726a",
		},
		dark: scheme{
			accent: "#ed986e", surface: "#211f1c", raised: "#2b2925",
			text: "#eeeae2", muted: "#bdb6aa",
		},
	},
	{
		id:   "violet",
		name: "Violet",
		light: scheme{
			accent: "#6750a4", surface: "#fef7ff", raised: "#ffffff",
			text: "#1d1b20", muted: "#49454f",
		},
		dark: scheme{
			accent: "#d0bcff", surface: "#141218", raised: "#211f26",
			text: "#e6e0e9", muted: "#cac4d0",
		},
	},
	{
		id:   "indigo",
		name: "Indigo",
		light: scheme{
			accent: "#4a5c92", surface: "#fbf8ff", raised: "#ffffff",
			text: "#1a1b21", muted: "#45464f",
		},
		dark: scheme{
			accent: "#b3c5ff", surface: "#121318", raised: "#1e1f25",
			text: "#e3e1e9", muted: "#c5c5d0",
		},
	},
	{
		id:   "blue",
		name: "Blue",
		light: scheme{
			accent: "#00639b", surface: "#f8f9ff", raised: "#ffffff",
			text: "#191c20", muted: "#42474e",
		},
		dark: scheme{
			accent: "#96ccff", surface: "#101418", raised: "#1c2024",
			text: "#e1e2e8", muted: "#c2c7ce",
		},
	},
	{
		id:   "teal",
		name: "Teal",
		light: scheme{
			accent: "#006a6a", surface: "#f5fbfa", raised: "#ffffff",
			text: "#191c1c", muted: "#3f4948",
		},
		dark: scheme{
			accent: "#4cdada", surface: "#0e1514", raised: "#1a2120",
			text: "#dee4e3", muted: "#bec9c8",
		},
	},
	{
		id:   "green",
		name: "Green",
		light: scheme{
			accent: "#386a20", surface: "#f8fbf1", raised: "#ffffff",
			text: "#191d16", muted: "#43483e",
		},
		dark: scheme{
			accent: "#9cd67e", surface: "#11140f", raised: "#1d211a",
			text: "#e2e4dc", muted: "#c3c8bb",
		},
	},
	{
		id:   "amber",
		name: "Amber",
		light: scheme{
			accent: "#6b5e00", surface: "#fefae0", raised: "#fffdf3",
			text: "#1d1c16", muted: "#48473d",
		},
		dark: scheme{
			accent: "#dbc90a", surface: "#14130b", raised: "#201f17",
			text: "#e6e3d4", muted: "#cac6b4",
		},
	},
	{
		id:   "rose",
		name: "Rose",
		light: scheme{
			accent: "#984061", surface: "#fff8f8", raised: "#ffffff",
			text: "#201a1b", muted: "#514345",
		},
		dark: scheme{
			accent: "#ffb1c8", surface: "#181113", raised: "#251c1e",
			text: "#ece0e1", muted: "#d5c2c5",
		},
	},
}

// paletteByID returns the palette with an id, and whether it exists. An empty
// id is Onevra's own, which is the default.
func paletteByID(id string) (palette, bool) {
	for _, p := range palettes {
		if p.id == id {
			return p, true
		}
	}
	return palette{}, false
}

// tint lays a Material 3 scheme over Onevra's theme.
//
// Only the roles the app paints are changed, and the accent's own hover,
// pressed, text-on-accent, selection and focus are derived exactly as before,
// so a palette changes the colour of the app and nothing about its shape,
// spacing or type.
func (p palette) tint(t *ui.Theme) {
	s := p.light
	if t.Dark {
		s = p.dark
	}
	t.Accent = ui.Hex(s.accent)
	t.Background = ui.Hex(s.surface)
	t.Surface = ui.Hex(s.raised)
	// The card's hovered face is a step darker than the card, as Onevra's own
	// is, so a control on a card still reads as raised.
	t.SurfaceHover = ui.Hex(s.surface).Mix(ui.Hex(s.text), 0.06)
	t.Text = ui.Hex(s.text)
	t.TextMuted = ui.Hex(s.muted)
	// The border follows the ink, faintly, so a hairline sits in the palette
	// instead of being a fixed grey that suits only one of them.
	t.Border = ui.Hex(s.text).Alpha(0.14)
}