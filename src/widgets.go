package src

import (
	"github.com/egoist/mygo/ui"
)

// The type of the app, in one place.
//
// Each helper is a role, not a size: a page name, a section name, the name of
// a track, the line under it, a caption, a small label. They carry the size,
// the weight, the tracking and the leading together, because those are one
// decision — change one alone and the page stops hanging together.

// pageName is the name of the page, at the top of it.
func pageName(c *ui.Context, name string) ui.Element {
	t := c.Theme()
	return ui.Text(c, name).FontSize(t.Rem(remDisplay)).FontWeight(700).
		LetterSpacing(trackDisplay(t)).LineHeight(leadDisplay).
		MaxLines(2).Ellipsis("…")
}

// leadName is the one name a page is about, as the track on the player.
func leadName(c *ui.Context, name string) ui.Element {
	t := c.Theme()
	return ui.Text(c, name).FontSize(t.Rem(remLead)).FontWeight(700).
		LetterSpacing(trackHeadline(t)).LineHeight(leadHeadline).
		MaxLines(2).Ellipsis("…")
}

// statValue is the number a page answers with, at the top of a card.
func statValue(c *ui.Context, value string) ui.Element {
	t := c.Theme()
	return ui.Text(c, value).FontSize(t.Rem(remDisplay)).FontWeight(700).
		LetterSpacing(trackDisplay(t)).LineHeight(1.1).FontFeatures("tnum")
}

// sectionName is the name of a group of things.
func sectionName(c *ui.Context, name string) ui.Element {
	t := c.Theme()
	return ui.Text(c, name).FontSize(t.Rem(remHeadline)).FontWeight(600).
		LetterSpacing(trackHeadline(t)).LineHeight(leadHeadline)
}

// cardName is the name at the top of a card.
func cardName(c *ui.Context, name string) ui.Element {
	t := c.Theme()
	return ui.Text(c, name).FontSize(t.Rem(remTitle)).FontWeight(600).
		LineHeight(leadTitle)
}

// itemName is the name of a track, or of a setting.
func itemName(c *ui.Context, name string) ui.Element {
	t := c.Theme()
	return ui.Text(c, name).FontSize(t.Rem(remTitle)).FontWeight(500).
		LineHeight(leadTitle)
}

// body is what a row says under its name, and the words of a page.
func body(c *ui.Context, words string) ui.Element {
	t := c.Theme()
	return ui.Text(c, words).FontSize(t.Rem(remBody)).
		LineHeight(leadBody).TextColor(surface(c).muted())
}

// caption is the smallest text: a length, a count, a hint. It is the floor of
// the scale, and says so by being the only role that names a size under the
// body.
func caption(c *ui.Context, words string) ui.Element {
	t := c.Theme()
	return ui.Text(c, words).FontSize(t.Rem(remCaption)).
		TextColor(surface(c).muted()).FontFeatures("tnum")
}

// eyebrow is the tracked, uppercased label over a name, as the design uses for
// "Live radio session" and the section headings in the rail.
func eyebrow(c *ui.Context, words string) ui.Element {
	t := c.Theme()
	return ui.Text(c, words).FontSize(t.Rem(remCaption)).FontWeight(600).
		LetterSpacing(0.06 * t.Rem(remCaption)).TextColor(surface(c).muted())
}

// mono is a number read as a number: a length, a count, an index. Tabular
// figures keep a column of them from shifting as they change.
func mono(c *ui.Context, words string) ui.Element {
	t := c.Theme()
	return ui.Text(c, words).FontSize(t.Rem(remCaption)).FontFeatures("tnum").
		TextColor(surface(c).muted())
}

// remControl is the size of a control's own words: a shade under the body,
// because a button is read at a glance and not in lines.
const remControl = 0.93

// card draws a group of things on a card face, with the design's one rounding,
// a hairline, and a shadow too faint to notice.
//
// A card does not move under the pointer. A card is a group of content, not a
// control: lifting it says "click me" about a thing that does nothing.
func card(c *ui.Context, build func()) ui.Element {
	s := surface(c)
	t := c.Theme()
	return ui.Column(c).FillWidth().Gap(t.Space(4)).Padding(t.Space(5)).
		Radius(radiusCard(t)).Background(s.card()).Border(1, s.hairline()).
		Shadow(0, 1, 4, 0, s.shadow()).Children(build)
}

// panel draws the larger card the design uses for a hero and a table: one
// rounding bigger, a little more room.
func panel(c *ui.Context, build func()) ui.Element {
	s := surface(c)
	t := c.Theme()
	return ui.Column(c).FillWidth().Gap(t.Space(5)).Padding(t.Space(6)).
		Radius(radiusCard(t)+t.Space(1)).Background(s.card()).
		Border(1, s.hairline()).Shadow(0, 1, 4, 0, s.shadow()).Children(build)
}

// group draws a card with a name at the top, one thing to do with it at the
// right, and its content under it.
func group(c *ui.Context, title string, trailing, build func()) ui.Element {
	t := c.Theme()
	return card(c, func() {
		if title != "" || trailing != nil {
			ui.Row(c).FillWidth().Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
				if title != "" {
					sectionName(c, title)
				}
				ui.Spacer(c)
				if trailing != nil {
					trailing()
				}
			})
		}
		if build != nil {
			build()
		}
	})
}

// sectionHead names a group of cards and says what it holds, with one action at
// the right: the shape of every section of the home page.
func sectionHead(c *ui.Context, title, note string, trailing func()) ui.Element {
	t := c.Theme()
	return ui.Row(c).FillWidth().Gap(t.Space(4)).AlignItems(ui.End).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(t.Space(0.5)).Children(func() {
			ui.Text(c, title).FontSize(t.Rem(remTitle)).FontWeight(600).
				LetterSpacing(trackHeadline(t)).TextColor(surface(c).text())
			if note != "" {
				ui.Text(c, note).FontSize(t.Rem(remCaption)).TextColor(surface(c).muted())
			}
		})
		if trailing != nil {
			trailing()
		}
	})
}

// pageHead draws the name of a page, what it holds, and what can be done
// there, on one line, with air under it.
func pageHead(c *ui.Context, title, subtitle string, trailing func()) ui.Element {
	t := c.Theme()
	return ui.Column(c).FillWidth().Gap(t.Space(1)).Padding(0, 0, t.Space(2)).Children(func() {
		ui.Row(c).FillWidth().Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Gap(t.Space(1)).Children(func() {
				pageName(c, title)
				if subtitle != "" {
					body(c, subtitle)
				}
			})
			if trailing != nil {
				trailing()
			}
		})
	})
}

// listHead names a list and says how much of it shows, in the small type that
// belongs beside a name.
func listHead(c *ui.Context, title, note string) ui.Element {
	t := c.Theme()
	return ui.Row(c).FillWidth().Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
		sectionName(c, title)
		ui.Spacer(c)
		if note != "" {
			caption(c, note)
		}
	})
}

// empty says that there is nothing here yet, and what would put something
// here, with a quiet mark of the thing that is missing.
func empty(c *ui.Context, mark *ui.SVG, title, hint string) ui.Element {
	s := surface(c)
	t := c.Theme()
	return ui.Column(c).Fill().FillWidth().Center().Gap(t.Space(3)).Children(func() {
		ui.Box(c).Size(t.Space(18), t.Space(18)).Shrink(0).Radius(radiusPill(t)).
			Background(s.sunken()).Center().Children(func() {
			ui.Icon(c, mark).Size(t.Space(8), t.Space(8)).TextColor(s.muted())
		})
		ui.Text(c, title).FontSize(t.Rem(remTitle)).FontWeight(600).
			TextColor(s.text()).TextAlign(ui.Center)
		ui.Text(c, hint).FontSize(t.Rem(remBody)).TextColor(s.muted()).
			LineHeight(leadBody).TextAlign(ui.Center).MaxWidth(t.Space(76))
	})
}

// setting draws one setting: what it is, what it does, and the control that
// changes it, in a row.
func setting(c *ui.Context, name, description string, control func()) ui.Element {
	t := c.Theme()
	return ui.Row(c).FillWidth().Gap(t.Space(6)).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(t.Space(0.5)).Children(func() {
			itemName(c, name)
			if description != "" {
				body(c, description)
			}
		})
		if control != nil {
			control()
		}
	})
}

// hairline draws the line between two settings inside a card.
func hairline(c *ui.Context) ui.Element {
	return ui.Box(c).FillWidth().Height(1).Background(surface(c).hairline())
}

// pressPose is how far a control settles when it is pressed.
//
// It moves by a DIP rather than shrinking: a Scale on a button resamples its
// label, and a label that blurs for 160 ms looks like a rendering fault rather
// than a press.
func pressPose(pressed bool) ui.Pose {
	if pressed {
		return ui.Pose{Y: 1}
	}
	return ui.Pose{}
}

// risePose is how far a tile that is itself a link rises under the pointer.
//
// Translation only: a Scale would resample the words on the tile.
func risePose(hovered bool) ui.Pose {
	if hovered {
		return ui.Pose{Y: -2}
	}
	return ui.Pose{}
}

// iconButton draws a button of an icon only, with its name for the pointer and
// for assistive technology.
func iconButton(c *ui.Context, mark *ui.SVG, label string) ui.Element {
	return iconButtonIn(c, mark, label, surface(c).muted())
}

// iconButtonIn is iconButton with the colour of its mark chosen by the caller.
//
// The colour has to come in here rather than through `Element.TextColor` after the
// fact: the mark is a child with a colour of its own, and a colour set on the
// button above it does not reach it. A toggle that says it is on by its colour
// therefore needs this form, or it never looks on.
func iconButtonIn(c *ui.Context, mark *ui.SVG, label string, ink ui.Color) ui.Element {
	s := surface(c)
	t := c.Theme()
	size := t.Space(8)
	b := ui.ButtonBase(c).Size(size, size).Radius(radiusControl(t)).Center().
		Label(label).Tooltip(label)
	face := ui.Transparent
	if b.Hovered() {
		face = s.sunken()
	}
	b.Background(b.AnimateColor("face", face, motionColor))
	b.AnimatePose(pressPose(b.Pressed()), motionPress)
	b.Children(func() {
		ui.Icon(c, mark).Size(t.Space(4.5), t.Space(4.5)).TextColor(ink)
	})
	return b
}

// primaryButton is the filled button the design leads with: near-black on paper,
// with its icon and its name.
func primaryButton(c *ui.Context, mark *ui.SVG, label string) ui.Element {
	t := c.Theme()
	b := buttonBase(c, true, mark, label)
	b.Padding(t.Space(2), t.Space(4)).Radius(radiusControl(t))
	return b
}

// secondaryButton is the same button, quieter: a hairline and the ink, for the
// second thing a page offers.
func secondaryButton(c *ui.Context, mark *ui.SVG, label string) ui.Element {
	t := c.Theme()
	b := buttonBase(c, false, mark, label)
	b.Padding(t.Space(2), t.Space(3.5)).Radius(radiusControl(t))
	return b
}

// quietButton is a button with no face until the pointer rests on it.
func quietButton(c *ui.Context, label string) ui.Element {
	s := surface(c)
	t := c.Theme()
	b := ui.ButtonBase(c).Radius(radiusControl(t)).Padding(t.Space(1.5), t.Space(3)).
		Label(label)
	face := ui.Transparent
	if b.Hovered() {
		face = s.sunken()
	}
	b.Background(b.AnimateColor("face", face, motionColor)).TextColor(s.muted())
	b.AnimatePose(pressPose(b.Pressed()), motionPress)
	b.Children(func() {
		ui.Text(c, label).FontSize(t.Rem(remControl)).FontWeight(500).TextColor(s.muted())
	})
	return b
}

// plainButton is a button of the ordinary kind at the size a settings row wants.
func plainButton(c *ui.Context, label string) ui.Element {
	t := c.Theme()
	b := buttonBase(c, false, nil, label)
	b.Padding(t.Space(1.5), t.Space(3.5)).Radius(radiusControl(t))
	return b
}

// buttonBase is the shape every button of the app shares.
//
// MyGo's own Button is used nowhere: its padding and its radius come from the
// theme's own numbers rather than the app's scale, and the design's filled button
// has to match the cards it sits on.
func buttonBase(c *ui.Context, primary bool, mark *ui.SVG, label string) ui.Element {
	s := surface(c)
	t := c.Theme()
	b := ui.ButtonBase(c).Gap(t.Space(2)).Label(label)
	ink := s.text()
	face := s.card()
	switch {
	case primary:
		face, ink = t.Accent, t.AccentText
	case !primary:
		b.Border(1, s.hairline())
	}
	if b.Hovered() {
		if primary {
			face = t.AccentHover
		} else {
			face = s.sunken()
		}
	}
	if b.Pressed() {
		face = face.Mix(ink, 0.08)
	}
	b.Background(b.AnimateColor("face", face, motionColor)).TextColor(ink)
	b.AnimatePose(pressPose(b.Pressed()), motionPress)
	b.Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			if mark != nil {
				ui.Icon(c, mark).Size(t.Space(4), t.Space(4)).TextColor(ink)
			}
			ui.Text(c, label).FontSize(t.Rem(remControl)).FontWeight(600).TextColor(ink)
		})
	})
	return b
}

// busyButton is the face a button takes while what it starts is running.
//
// It is not a disabled button, it is a status: a disabled one is drawn at half
// opacity, which makes the words that say what is happening the faintest thing
// on the page.
func busyButton(c *ui.Context, label string) ui.Element {
	s := surface(c)
	t := c.Theme()
	b := ui.Box(c).Radius(radiusControl(t)).Gap(t.Space(2)).
		Padding(t.Space(2), t.Space(4)).AlignItems(ui.Center).
		Background(s.sunken())
	b.Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Spinner(c).Size(t.Space(4), t.Space(4)).TextColor(s.text())
			ui.Text(c, label).FontSize(t.Rem(remControl)).FontWeight(600).
				TextColor(s.text())
		})
	})
	return b
}

// chip is a small selectable pill, as the design's "All / Electronic / Ambient"
// row.
func chip(c *ui.Context, label string, chosen bool) ui.Element {
	s := surface(c)
	t := c.Theme()
	b := ui.ButtonBase(c).Padding(t.Space(1), t.Space(2.5)).Radius(radiusControl(t)).
		Label(label)
	face := ui.Transparent
	switch {
	case chosen:
		face = s.card()
	case b.Hovered():
		face = s.sunken()
	}
	b.Background(b.AnimateColor("face", face, motionColor))
	if chosen {
		b.Border(1, s.hairline())
		b.Shadow(0, 1, 2, 0, s.shadow())
	}
	ink := s.muted()
	if chosen {
		ink = s.text()
	}
	b.Children(func() {
		ui.Text(c, label).FontSize(t.Rem(remCaption)).FontWeight(500).TextColor(ink)
	})
	return b
}
