package src

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/icons/lucide"
)

// accents are the colours the accent may take.
var accents = []string{
	"#6750a4", // purple
	"#4a5c92", // indigo
	"#00639b", // blue
	"#006a6a", // teal
	"#386a20", // green
	"#6b5e00", // yellow
	"#8f4c38", // brown
	"#984061", // pink
}

// appearancePage draws the settings of the look and of the queue.
//
// The page is built from the same cards and rows as every other page, with
// Onevra's own controls in them, so a setting sits where a track would: the
// app has one grid, and settings are not a screen of their own.
func (a *app) appearancePage(c *ui.Context) {
	t := c.Theme()
	// The switches are bound to these fields for the frame, because a switch
	// flips the value it is given as it is built. They are seeded from the
	// real state here, before anything is drawn, and read back after.
	a.switchShuffle = a.isShuffle()
	a.switchDynamic = a.appearance.dynamic

	a.pageFrame(c, true, func() {
		ui.Column(c).FillWidth().Gap(t.Space(6)).Children(func() {
			pageHead(c, "Look", "The theme, the colour it grows from, and how the queue plays.", nil)

			group(c, "Theme", nil, func() {
				setting(c, "Appearance", "Follow the desktop, or stay light or dark.", func() {
					modes := []string{"System", "Light", "Dark"}
					selected := 0
					switch a.appearance.mode {
					case "light":
						selected = 1
					case "dark":
						selected = 2
					}
					segmented := ui.Segmented(c, &selected, modes...)
					segmented.Width(t.Space(56))
					if segmented.Changed() {
						a.setMode(modes[selected])
					}
				})
				hairline(c)
				setting(c, "Palette", "A Material 3 colour scheme. Every surface, line and accent follows it.", nil)
				ui.Row(c).Gap(t.Space(1)).Wrap().Children(func() {
					for _, p := range palettes {
						a.paletteSwatch(c, p)
					}
				})
				hairline(c)
				setting(c, "Accent", "A colour laid over the palette's own.", nil)
				ui.Row(c).Gap(t.Space(1)).Wrap().Children(func() {
					// The first swatch is the palette's own accent, not one laid
					// over it, so a choice once made can be unmade.
					a.accentSwatch(c, "", "Palette", a.paletteAccent(t), t.AccentText)
					for _, accent := range accents {
						a.accentSwatch(c, accent, "", ui.Hex(accent), ui.Hex(markOn(accent)))
					}
				})
				hairline(c)
				setting(c, "Colour from artwork",
					"Take the accent from the cover of the track that plays.", func() {
						a.switchSetting(c, &a.switchDynamic, "Colour from artwork")
					})
			})

			group(c, "Queue", nil, func() {
				setting(c, "Repeat", "What happens at the end of the queue.", func() {
					labels := []string{"Off", "Queue", "Track"}
					selected := a.repeatMode()
					segmented := ui.Segmented(c, &selected, labels...)
					segmented.Width(t.Space(52))
					if segmented.Changed() {
						a.setRepeat(selected)
					}
				})
				hairline(c)
				setting(c, "Shuffle", "Play the queue in a random order.", func() {
					a.switchSetting(c, &a.switchShuffle, "Shuffle")
				})
			})
		})
	})

	// Read the switches back: a switch flips its own field as it is built, so
	// what it holds now is what the user chose.
	if a.switchDynamic != a.appearance.dynamic {
		a.setDynamic(a.switchDynamic)
	}
	if a.switchShuffle != a.isShuffle() {
		a.setShuffle(a.switchShuffle)
	}
}

// switchSetting draws a switch bound to a setting, and reports whether the user
// turned it in this frame.
//
// This is subtler than it looks, and the obvious version does not work.
// components.Switch binds its own bool and flips it when clicked; it reports
// nothing back, and the row it returns never reports a change, because the
// switch inside it does. Worse, MyGo applies a toggle's input as the control
// is built, so the bool has already been flipped by the time Switch returns:
// comparing it with what was passed in is always equal, and the setting never
// changes.
//
// So the value lives on the app, the switch is handed a pointer to it, and the
// caller reads the field back once the frame has drawn. A click moves the
// field; nothing else does.
func (a *app) switchSetting(c *ui.Context, field *bool, label string) {
	components.Switch(c, field, label, components.SwitchOptions{HideLabel: true})
}
//
// The chosen one carries a check rather than a ring: a ring would fight the
// swatch's own edge, and the mark has to read on any colour. An empty accent
// is the theme's own, which is what the app shows before the user picks one.
// paletteSwatch draws one of the Material 3 schemes: a tile in the palette's
// own surface with a disc of its accent on it, so the swatch shows the scheme
// rather than only its colour.
func (a *app) paletteSwatch(c *ui.Context, p palette) {
	t := c.Theme()
	chosen := a.appearance.palette == p.id
	s := p.light
	if t.Dark {
		s = p.dark
	}
	key := "palette-" + p.id
	if p.id == "" {
		key = "palette-default"
	}
	button := ui.ButtonBase(c).Key(key).Size(t.Space(13), t.Space(9)).
		Center().Radius(radiusControl(t)).Label("Palette " + p.name).
		Transition(motionPress)
	if button.Hovered() {
		button.Background(surface(c).card())
	}
	button.Children(func() {
		tile := ui.Box(c).Size(t.Space(9), t.Space(7)).Radius(radiusControl(t)).
			Background(ui.Hex(s.surface)).Center()
		tile.Border(1, ui.Hex(s.text).Alpha(0.14))
		tile.Children(func() {
			ui.Box(c).Size(t.Space(4), t.Space(4)).Radius(radiusPill(t)).
				Background(ui.Hex(s.accent))
		})
	})
	if button.Clicked() {
		a.setPalette(p.id)
	}
	// A chosen tile is marked by a ring in the palette's own accent, which
	// reads on the tile's surface whatever colour the scheme is.
	if chosen {
		button.Border(2, ui.Hex(s.accent))
	}
}

// paletteAccent returns the accent of the palette in use, which is what the
// first accent swatch offers as "the palette's own".
func (a *app) paletteAccent(t *ui.Theme) ui.Color {
	if p, ok := paletteByID(a.appearance.palette); ok {
		if t.Dark {
			return ui.Hex(p.dark.accent)
		}
		return ui.Hex(p.light.accent)
	}
	return t.Accent
}

// setPalette keeps the Material 3 scheme the user chose.
//
// A palette brings its own accent, so the colour laid over it is cleared: two
// settings deciding the same thing is how a colour picker starts fighting
// itself.
func (a *app) setPalette(id string) {
	a.appearance.palette = id
	a.appearance.accent = ""
	a.appearance.dynamic = false
	a.rememberAppearance()
}

func (a *app) accentSwatch(c *ui.Context, accent, name string, colour ui.Color, mark ui.Color) {
	t := c.Theme()
	chosen := a.appearance.accent == accent
	key := "accent-" + accent
	if accent == "" {
		key = "accent-default"
	}
	if name == "" {
		name = accent
	}
	button := ui.ButtonBase(c).Key(key).Size(t.Space(11), t.Space(11)).
		Center().Radius(radiusPill(t)).Label("Accent "+name).Transition(motionPress)
	if button.Hovered() {
		button.Background(surface(c).card())
	}
	button.Children(func() {
		swatch := ui.Box(c).Size(t.Space(8), t.Space(8)).Radius(radiusPill(t)).
			Background(colour).Center()
		swatch.Children(func() {
			if chosen {
				components.Icon(c, lucide.Check()).Size(t.Space(4), t.Space(4)).
					TextColor(mark)
			}
		})
	})
	if button.Clicked() {
		a.setAccent(accent)
	}
}

// setMode keeps the appearance the user chose.
//
// The names are the ones the segmented control shows, not the ones the
// settings keep: "Dark" is what the user picked, "dark" is what is stored.
func (a *app) setMode(mode string) {
	switch mode {
	case "Light":
		a.appearance.mode = "light"
		mygo.Theme.SetSource(mygo.ThemeLight)
	case "Dark":
		a.appearance.mode = "dark"
		mygo.Theme.SetSource(mygo.ThemeDark)
	default:
		a.appearance.mode = "system"
		mygo.Theme.SetSource(mygo.ThemeSystem)
	}
	a.rememberAppearance()
}

// setAccent keeps the colour the user chose.
func (a *app) setAccent(accent string) {
	a.appearance.accent = accent
	a.appearance.dynamic = false
	a.rememberAppearance()
}

// setDynamic keeps whether the colour follows the artwork.
func (a *app) setDynamic(dynamic bool) {
	a.appearance.dynamic = dynamic
	if dynamic {
		a.appearance.accent = ""
	}
	a.rememberAppearance()
}

// rememberAppearance keeps what the user chose about the look.
//
// The mode is written back as the app holds it, so picking a colour does not
// quietly reset the appearance to whatever the desktop is doing: a setting the
// user did not touch has to survive a setting they did.
func (a *app) rememberAppearance() {
	if a.store == nil {
		return
	}
	mode := a.appearance.mode
	if mode == "" {
		mode = "system"
	}
	a.store.SetAppearance(mode, a.appearance.accent, a.appearance.dynamic, a.appearance.palette)
}
