package src

import (
	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
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
	return components.Text(c, name).FontSize(t.Rem(remDisplay)).FontWeight(600).
		LetterSpacing(trackDisplay(t)).LineHeight(leadDisplay).
		MaxLines(2).Ellipsis("…")
}

// leadName is the one name a page is about, as the track on the player: it is
// a shade under a page's name, because the artwork above it already says what
// the page is.
func leadName(c *ui.Context, name string) ui.Element {
	t := c.Theme()
	return components.Text(c, name).FontSize(t.Rem(remLead)).FontWeight(600).
		LetterSpacing(trackHeadline(t)).LineHeight(leadHeadline).
		MaxLines(2).Ellipsis("…")
}

// statValue is the number a page answers with, at the top of a card.
func statValue(c *ui.Context, value string) ui.Element {
	t := c.Theme()
	return components.Text(c, value).FontSize(t.Rem(remDisplay)).FontWeight(600).
		LetterSpacing(trackDisplay(t)).LineHeight(1.1).FontFeatures("tnum")
}

// sectionName is the name of a group of things.
func sectionName(c *ui.Context, name string) ui.Element {
	t := c.Theme()
	return components.Text(c, name).FontSize(t.Rem(remHeadline)).FontWeight(600).
		LetterSpacing(trackHeadline(t)).LineHeight(leadHeadline)
}

// cardName is the name at the top of a card.
func cardName(c *ui.Context, name string) ui.Element {
	t := c.Theme()
	return components.Text(c, name).FontSize(t.Rem(remTitle)).FontWeight(600).
		LineHeight(leadTitle)
}

// itemName is the name of a track, or of a setting.
func itemName(c *ui.Context, name string) ui.Element {
	t := c.Theme()
	return components.Text(c, name).FontSize(t.Rem(remTitle)).FontWeight(500).
		LineHeight(leadTitle)
}

// body is what a row says under its name, and the words of a page.
func body(c *ui.Context, words string) ui.Element {
	t := c.Theme()
	return components.Text(c, words).FontSize(t.Rem(remBody)).
		LineHeight(leadBody).TextColor(surface(c).muted())
}

// caption is the smallest text: a length, a count, a hint. It is the floor of
// the scale, and says so by being the only role that names a size under the
// body.
func caption(c *ui.Context, words string) ui.Element {
	t := c.Theme()
	return components.Text(c, words).FontSize(t.Rem(remCaption)).
		TextColor(surface(c).muted()).FontFeatures("tnum")
}

// card draws a group of things on the surface, rounded once, with the room it
// needs and no border: a shadow too faint to notice separates it from the page
// without drawing a line around it.
func card(c *ui.Context, build func()) ui.Element {
	s := surface(c)
	t := c.Theme()
	return ui.Column(c).FillWidth().Gap(t.Space(5)).Padding(t.Space(6)).
		Radius(radiusCard(t)).Background(s.card()).
		Shadow(0, 1, 2, 0, s.shadow()).Children(build)
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
func empty(c *ui.Context, icon *ui.SVG, title, hint string) ui.Element {
	s := surface(c)
	t := c.Theme()
	return ui.Column(c).Fill().FillWidth().Center().Gap(t.Space(3)).Children(func() {
		ui.Box(c).Size(t.Space(18), t.Space(18)).Shrink(0).Radius(radiusPill(t)).
			Background(s.chosen()).Center().Children(func() {
			components.Icon(c, icon).Size(t.Space(8), t.Space(8)).TextColor(s.accent())
		})
		components.Text(c, title).FontSize(t.Rem(remTitle)).FontWeight(600).
			TextAlign(ui.Center)
		components.Text(c, hint).FontSize(t.Rem(remBody)).TextColor(s.muted()).
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

// iconButton draws a button of an icon only, with its name for the pointer.
func iconButton(c *ui.Context, icon *ui.SVG, label string) ui.Element {
	return components.IconButton(c, icon, label, components.IconButtonPlain)
}

// primaryButton is a button that carries its icon and its name, at the size a
// main action wants.
func primaryButton(c *ui.Context, icon *ui.SVG, label string) ui.Element {
	t := c.Theme()
	b := components.Button(c, "", components.ButtonPrimary).Label(label).
		Padding(t.Space(2), t.Space(5))
	b.Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			if icon != nil {
				components.Icon(c, icon).Size(t.Space(4.5), t.Space(4.5)).TextColor(t.AccentText)
			}
			components.Text(c, label).FontSize(t.Rem(remControl)).FontWeight(500).
				TextColor(t.AccentText)
		})
	})
	return b
}

// remControl is the size of a control's own words: a shade under the body,
// because a button is read at a glance and not in lines.
const remControl = 0.93

// secondaryButton is the same button, quieter: a bordered face for the second
// thing a page offers.
func secondaryButton(c *ui.Context, icon *ui.SVG, label string) ui.Element {
	s := surface(c)
	t := c.Theme()
	b := components.Button(c, "", components.ButtonSecondary).Label(label).
		Padding(t.Space(2), t.Space(5))
	b.Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			if icon != nil {
				components.Icon(c, icon).Size(t.Space(4.5), t.Space(4.5)).TextColor(s.muted())
			}
			components.Text(c, label).FontSize(t.Rem(remControl)).FontWeight(500)
		})
	})
	return b
}
