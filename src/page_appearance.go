package src

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// accents are the studio tones the accent may take: the charcoal of the paper's
// own ink, then four muted colours that read on it.
//
// They are muted on purpose. A saturated accent turns the transport and the
// scrubber into the loudest thing on a page that is meant to be a quiet desktop
// player.
var accents = []struct {
	name string
	hex  string
}{
	{"Charcoal", "#1C1C1A"},
	{"Terra Cotta", "#B86A51"},
	{"Sage", "#6B7A68"},
	{"Slate Blue", "#566E7C"},
	{"Desert Ochre", "#B08848"},
}

// materials are the window's backdrops, weakest first.
//
// They are Windows 11's own three: Tabbed tints the window faintly, Mica tints it
// with the wallpaper, and Acrylic blurs what is behind it. On a desktop without
// them the window simply draws its own face, which is what it does anyway.
var materials = []struct {
	label string
	value mygo.Vibrancy
}{
	{"Subtle", mygo.VibrancyTabbed},
	{"Standard", mygo.VibrancyMica},
	{"High", mygo.VibrancyAcrylic},
}

// appearancePage draws the settings of the look.
//
// It is four cards on the paper: the interface theme as three previews, the accent
// and what may tint it, the window's material, and the account the app shares
// what plays with. The page is the same grid as every other one — the same card,
// the same room, the same type roles — so settings sit where a track would.
func (a *app) appearancePage(c *ui.Context) {
	t := c.Theme()
	// The switches are bound to these fields for the frame, because a switch flips
	// the value it is given as it is built. They are seeded from the real state
	// here, before anything is drawn, and read back after.
	a.switchDynamic = a.appearance.dynamic
	a.switchDiscord = a.appearance.discord
	a.switchSolid = a.appearance.solid

	a.pageFrame(c, true, func() {
		ui.Column(c).FillWidth().Gap(t.Space(8)).Children(func() {
			a.appearanceHead(c)
			a.themeSection(c)
			a.accentSection(c)
			a.materialSection(c)
			a.discordSection(c)
		})
	})

	// Read the switches back: a switch flips its own field as it is built, so what
	// it holds now is what the user chose.
	if a.switchDynamic != a.appearance.dynamic {
		a.setDynamic(a.switchDynamic)
	}
	if a.switchDiscord != a.appearance.discord {
		a.setDiscord(a.switchDiscord)
	}
	if a.switchSolid != a.appearance.solid {
		a.setSolid(a.switchSolid)
	}
}

// appearanceHead is the page's name, what it does, and where it sits.
func (a *app) appearanceHead(c *ui.Context) {
	s := surface(c)
	t := c.Theme()
	ui.Column(c).FillWidth().Gap(t.Space(2)).Padding(0, 0, t.Space(2)).Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			eyebrow(c, "Preferences")
			ui.Text(c, "/").FontSize(t.Rem(remCaption)).TextColor(s.faint())
			ui.Text(c, "Appearance").FontSize(t.Rem(remCaption)).FontWeight(600).
				LetterSpacing(0.06 * t.Rem(remCaption)).TextColor(s.text())
		})
		pageName(c, "Appearance")
		body(c, "The visual tone of the window, the colour it grows from, and the material behind it.").
			MaxWidth(t.Space(120))
	})
}

// themeSection is the three previews: light, dark, and the desktop's own.
func (a *app) themeSection(c *ui.Context) {
	t := c.Theme()
	card(c, func() {
		ui.Column(c).FillWidth().Gap(t.Space(2)).Children(func() {
			sectionName(c, "Interface theme")
			body(c, "Match the Windows theme, or anchor the window to a fixed brightness.")
		})
		ui.Row(c).FillWidth().Gap(t.Space(5)).Padding(t.Space(4), 0, 0, 0).Children(func() {
			a.themeTile(c, "Light", "Optimized for daylight focus", themeLight)
			a.themeTile(c, "Dark", "Deep charcoal acoustics", themeDark)
			a.themeTile(c, "Auto", "Synchronized with Windows", themeAuto)
		})
	})
}

// themeKind is which of the three previews a tile shows.
type themeKind int

const (
	themeLight themeKind = iota
	themeDark
	themeAuto
)

// themeTile is one preview: a miniature of the window in that appearance, its
// name, and a mark when it is the one in use.
//
// The miniature is drawn rather than pictured: three dots for the window's
// controls, a rail, and a page with a cover and a scrubber. It is small enough
// that a real screenshot would be mud, and drawing it keeps it in the palette.
func (a *app) themeTile(c *ui.Context, name, note string, kind themeKind) {
	t := c.Theme()
	chosen := a.modeKind() == kind
	b := ui.ButtonBase(c).Grow(1).MinWidth(0).AlignItems(ui.Start)
	face := ui.Transparent
	if b.Hovered() {
		face = surface(c).sunken().Alpha(0.5)
	}
	b.Background(b.AnimateColor("face", face, motionColor))
	b.AnimatePose(pressPose(b.Pressed()), motionPress)
	b.Children(func() {
		ui.Column(c).FillWidth().Gap(t.Space(2)).Children(func() {
			a.themePreview(c, kind, chosen)
			ui.Row(c).FillWidth().Gap(t.Space(2)).AlignItems(ui.Center).
				Padding(0, t.Space(1), 0, t.Space(1)).Children(func() {
				ui.Column(c).Grow(1).MinWidth(0).Gap(t.Space(0.25)).Children(func() {
					ink := surface(c).text()
					weight := 500
					if chosen {
						weight = 600
					}
					ui.Text(c, name).FontSize(t.Rem(remCaption)).FontWeight(weight).
						TextColor(ink)
					ui.Text(c, note).FontSize(t.Rem(remCaption)).
						TextColor(surface(c).muted()).MaxLines(1).Ellipsis("…")
				})
				a.checkMark(c, chosen)
			})
		})
	})
	if b.Clicked() {
		switch kind {
		case themeLight:
			a.setMode("Light")
		case themeDark:
			a.setMode("Dark")
		default:
			a.setMode("System")
		}
	}
}

// themePreview draws the miniature window.
func (a *app) themePreview(c *ui.Context, kind themeKind, chosen bool) {
	t := c.Theme()
	// The two faces the miniature is made of, which are the tile's own and not the
	// window's: a preview of the dark appearance has to be dark while the window
	// is light.
	var page, card, line, ink ui.Color
	switch kind {
	case themeLight:
		page, card = ui.Hex(paperRail), ui.Hex(paperCard)
		line, ink = ui.Hex(paperLine), ui.Hex(paperInk)
	case themeDark:
		page, card = ui.Hex("#161614"), ui.Hex("#282824")
		line, ink = ui.Hex("#2E2E2A"), ui.Hex("#FAF9F5")
	default:
		page, card = ui.Hex(paperRail), ui.Hex(paperCard)
		line, ink = ui.Hex(paperLine), ui.Hex(paperInk)
	}

	frame := ui.Box(c).FillWidth().AspectRatio(1.6).Radius(radiusCard(t)).
		Background(page).Clip().Padding(t.Space(2))
	if chosen {
		frame.Border(2, t.Accent)
	}
	frame.Children(func() {
		ui.Column(c).FillWidth().FillHeight().Gap(t.Space(1.5)).Children(func() {
			// The window's own dots, and the bar beside them.
			ui.Row(c).FillWidth().Gap(t.Space(1)).AlignItems(ui.Center).Children(func() {
				if kind == themeAuto {
					// Half and half: the light face on the left, the dark on the
					// right, which is what "follow the desktop" means.
					ui.Row(c).FillWidth().Children(func() {
						ui.Box(c).Grow(1).Height(t.Space(1)).Radius(t.Space(0.5)).
							Background(ui.Hex(paperFaint))
						ui.Box(c).Grow(1).Height(t.Space(1)).Radius(t.Space(0.5)).
							Background(ui.Hex("#6A6963"))
					})
					return
				}
				for range 3 {
					ui.Box(c).Size(t.Space(1), t.Space(1)).Radius(radiusPill(t)).
						Background(ink.Alpha(0.25))
				}
				ui.Spacer(c)
				ui.Box(c).Width(t.Space(6)).Height(t.Space(1)).Radius(t.Space(0.5)).
					Background(line)
			})
			// The rail beside the page.
			ui.Row(c).FillWidth().Grow(1).Gap(t.Space(1.5)).Children(func() {
				ui.Column(c).WidthPercent(34).FillHeight().Gap(t.Space(1)).
					Padding(t.Space(1)).Radius(t.Space(1)).Background(page).Children(func() {
					for i := range 3 {
						width := ui.Auto
						if i == 1 {
							width = 0.7
						}
						_ = width
						ui.Box(c).FillWidth().Height(t.Space(1)).Radius(t.Space(0.5)).
							Background(line)
					}
				})
				ui.Column(c).Grow(1).FillHeight().Gap(t.Space(1)).
					Padding(t.Space(1.5)).Radius(t.Space(1)).Background(card).Children(func() {
					ui.Row(c).Gap(t.Space(1)).AlignItems(ui.Center).Children(func() {
						ui.Box(c).Size(t.Space(3), t.Space(3)).Radius(t.Space(0.75)).
							Background(line)
						ui.Column(c).Grow(1).Gap(t.Space(0.5)).Children(func() {
							ui.Box(c).FillWidth().Height(t.Space(1)).Radius(t.Space(0.5)).
								Background(ink)
							ui.Box(c).WidthPercent(60).Height(t.Space(0.75)).
								Radius(t.Space(0.4)).Background(line)
						})
					})
					ui.Spacer(c)
					// The scrubber, half played.
					ui.Box(c).FillWidth().Height(t.Space(1)).Radius(t.Space(0.5)).
						Background(line).Children(func() {
						ui.Box(c).WidthPercent(50).FillHeight().Radius(t.Space(0.5)).
							Background(t.Accent)
					})
				})
			})
		})
	})
}

// checkMark is the round mark a chosen preview carries.
func (a *app) checkMark(c *ui.Context, chosen bool) {
	t := c.Theme()
	box := ui.Box(c).Size(t.Space(4), t.Space(4)).Shrink(0).Radius(radiusPill(t)).
		Center()
	if chosen {
		box.Background(t.Accent)
	} else {
		box.Border(1, surface(c).hairline())
	}
	box.Children(func() {
		if chosen {
			ui.Icon(c, iconCheck).Size(t.Space(2.5), t.Space(2.5)).
				TextColor(t.AccentText)
		}
	})
}

// accentSection is the colour row and the switch that lets the artwork cast it.
func (a *app) accentSection(c *ui.Context) {
	t := c.Theme()
	card(c, func() {
		ui.Column(c).FillWidth().Gap(t.Space(2)).Children(func() {
			sectionName(c, "Accent colour")
			body(c, "A studio tone for the scrubber, the play button and what is selected.")
		})
		ui.Row(c).Gap(t.Space(3)).Padding(t.Space(3), 0, 0, 0).Children(func() {
			for _, accent := range accents {
				a.accentSwatch(c, accent.name, accent.hex)
			}
		})
		ui.Box(c).FillWidth().Radius(radiusControl(t)).Padding(t.Space(3.5)).
			Background(surface(c).sunken().Alpha(0.6)).Children(func() {
			ui.Row(c).FillWidth().Gap(t.Space(3.5)).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(t.Space(8), t.Space(8)).Shrink(0).
					Radius(radiusControl(t)).Background(surface(c).card()).Center().
					Children(func() {
					ui.Icon(c, iconPalette).Size(t.Space(4), t.Space(4)).
						TextColor(surface(c).text())
				})
				ui.Column(c).Grow(1).MinWidth(0).Gap(t.Space(0.5)).Children(func() {
					ui.Text(c, "Dynamic artwork radiance").FontSize(t.Rem(remCaption)).
						FontWeight(600).TextColor(surface(c).text())
					ui.Text(c, "Let the hue of the cover cast a faint tint over the accent.").
						FontSize(t.Rem(remCaption)).TextColor(surface(c).muted()).
						MaxLines(2)
				})
				ui.Switch(c, &a.switchDynamic).Label("Dynamic artwork radiance")
			})
		})
	})
}

// accentSwatch is one of the tones: a disc in the colour, with a mark when it is
// the one in use.
func (a *app) accentSwatch(c *ui.Context, name, hex string) {
	t := c.Theme()
	chosen := a.appearance.accent == hex
	b := ui.ButtonBase(c).Size(t.Space(8), t.Space(8)).Center().
		Radius(radiusPill(t)).Label("Accent " + name).Tooltip(name)
	b.AnimatePose(pressPose(b.Pressed()), motionPress)
	b.Children(func() {
		disc := ui.Box(c).Size(t.Space(7), t.Space(7)).Radius(radiusPill(t)).
			Background(ui.Hex(hex)).Center()
		if chosen {
			disc.Children(func() {
				ui.Icon(c, iconCheck).Size(t.Space(3.5), t.Space(3.5)).
					TextColor(ui.Hex(markOn(hex)))
			})
		}
	})
	// A chosen tone is ringed, which reads on any of them.
	if chosen {
		b.Border(2, t.Accent)
	}
	if b.Clicked() {
		a.setAccent(hex)
	}
}

// materialSection is the window's backdrop: how strong it is, and whether the
// window draws its own face instead.
func (a *app) materialSection(c *ui.Context) {
	t := c.Theme()
	card(c, func() {
		ui.Column(c).FillWidth().Gap(t.Space(2)).Children(func() {
			sectionName(c, "Window & materials")
			body(c, "How much of the desktop shows through the window. Windows 11 draws these backdrops; another desktop draws the window's own face.")
		})
		setting(c, "Sidebar acrylic & Mica", "The material behind the library rail.", func() {
			a.materialSegments(c)
		})
		hairline(c)
		setting(c, "Reduce transparency", "Draw an opaque face instead of the material behind it.", func() {
			ui.Switch(c, &a.switchSolid).Label("Reduce transparency")
		})
	})
}

// materialSegments is the three-step control, as the design draws it.
func (a *app) materialSegments(c *ui.Context) {
	s := surface(c)
	t := c.Theme()
	ui.Row(c).Padding(t.Space(1)).Radius(radiusControl(t)).Gap(t.Space(1)).
		Background(s.sunken()).Children(func() {
		for _, material := range materials {
			chosen := a.appearance.material == string(material.value)
			b := ui.ButtonBase(c).Padding(t.Space(1.5), t.Space(3)).
				Radius(radiusControl(t)).Label(material.label)
			face := ui.Transparent
			switch {
			case chosen:
				face = s.card()
			case b.Hovered():
				face = s.card().Alpha(0.5)
			}
			b.Background(b.AnimateColor("face", face, motionColor))
			if chosen {
				b.Shadow(0, 1, 2, 0, s.shadow())
			}
			ink := s.muted()
			if chosen {
				ink = s.text()
			}
			b.Children(func() {
				ui.Text(c, material.label).FontSize(t.Rem(remCaption)).
					FontWeight(600).TextColor(ink)
			})
			if b.Clicked() {
				a.setMaterial(string(material.value))
			}
		}
	})
}

// discordSection is the one sharing setting the app has.
//
// It keeps the card it had before this page was redesigned: the app shows what
// plays in Discord, and the switch is the only control it needs.
func (a *app) discordSection(c *ui.Context) {
	t := c.Theme()
	card(c, func() {
		ui.Column(c).FillWidth().Gap(t.Space(2)).Children(func() {
			sectionName(c, "Sharing")
			body(c, "What the app tells other applications about what plays.")
		})
		setting(c, "Discord", a.discordHint(), func() {
			ui.Switch(c, &a.switchDiscord).Label("Discord")
		})
	})
}

// discordHint says what the Discord switch does, and why it cannot be turned on
// when the build has no application id to register the presence under.
func (a *app) discordHint() string {
	if discordClientID == "" {
		return "Show what plays in your Discord profile. This build has no Discord application id, so it cannot."
	}
	return "Show what plays in your Discord profile."
}

// modeKind is which of the three previews the current appearance is.
func (a *app) modeKind() themeKind {
	switch a.appearance.mode {
	case "light":
		return themeLight
	case "dark":
		return themeDark
	default:
		return themeAuto
	}
}

// setDiscord turns the Discord presence on or off, and keeps the choice.
func (a *app) setDiscord(on bool) {
	a.appearance.discord = on
	if a.store != nil {
		a.store.SetDiscord(on)
	}
	if a.presence == nil {
		return
	}
	a.presence.enable(on && discordClientID != "")
	if on && discordClientID != "" {
		// Show what is playing now, rather than waiting for the next track.
		a.mu.Lock()
		track := a.trackLocked()
		playing := a.playing
		a.mu.Unlock()
		a.presence.show(track, playing)
	}
}

// setMode keeps the appearance the user chose, and tells the desktop.
//
// The two are separate on purpose. Recording the choice is the app's own state and
// costs nothing; `mygo.Theme.SetSource` reaches the desktop and waits for it, so it
// is only asked when there is a window to ask for. Without that guard a test that
// sets the appearance would hang on the main thread forever, which is what
// happened the first time this was written.
func (a *app) setMode(mode string) {
	switch mode {
	case "Light":
		a.appearance.mode = "light"
		a.applyThemeSource(mygo.ThemeLight)
	case "Dark":
		a.appearance.mode = "dark"
		a.applyThemeSource(mygo.ThemeDark)
	default:
		a.appearance.mode = "system"
		a.applyThemeSource(mygo.ThemeSystem)
	}
	a.rememberAppearance()
}

// applyThemeSource tells the desktop which appearance to draw its own chrome in,
// when there is a desktop to tell.
func (a *app) applyThemeSource(source mygo.ThemeSource) {
	if a.win == nil {
		return
	}
	mygo.Theme.SetSource(source)
}

// setAccent keeps the colour the user chose, and stops the artwork casting one:
// two things deciding the same colour is how a picker starts fighting itself.
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

// setMaterial keeps the strength of the window's backdrop, and tells the window.
func (a *app) setMaterial(material string) {
	a.appearance.material = material
	a.rememberAppearance()
	a.applyMaterial()
}

// setSolid keeps whether the window draws its own face.
func (a *app) setSolid(solid bool) {
	a.appearance.solid = solid
	a.rememberAppearance()
	a.applyMaterial()
}

// applyMaterial tells the window which backdrop to draw behind it.
//
// A window that draws its own opaque face asks for none: the material is behind
// the view, so a view that paints the paper would cover it anyway, and asking for
// it costs a compositor pass for nothing.
func (a *app) applyMaterial() {
	if a.win == nil {
		return
	}
	material := mygo.Vibrancy(a.appearance.material)
	if a.appearance.solid {
		material = mygo.VibrancyNone
	}
	a.win.SetVibrancy(material)
}

// rememberAppearance keeps what the user chose about the look.
func (a *app) rememberAppearance() {
	if a.store == nil {
		return
	}
	mode := a.appearance.mode
	if mode == "" {
		mode = "system"
	}
	a.store.SetAppearance(mode, a.appearance.accent, a.appearance.dynamic, "")
	a.store.SetMaterial(a.appearance.material, a.appearance.solid)
}
