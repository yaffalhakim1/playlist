package src

import (
	"github.com/egoist/mygo/ui"
)

// frame draws the whole window: the rail at the left, the header over the page,
// the page itself, the bar at the bottom, and the messages over all of it.
//
// This is the only view: the window runs it, and the tests run it too, so what a
// test draws is what the window shows.
func (a *app) frame(c *ui.Context) {
	c.SetTheme(a.theme(c))
	a.shortcuts(c)
	a.shellView(c)
	a.confirmDelete(c)
}

// shellView draws the shell's layout.
//
// The rail is its own surface with a hairline where it ends. The header sits over
// the page only — not over the rail — so the rail's own window dots and the
// page's search sit on their own surfaces, as the design has them.
//
// A material behind the window only shows where the view paints nothing, so when
// one is on the page leaves its face transparent and the desktop's backdrop shows
// through. That is what the Appearance page's material setting is for; with it off,
// or on a desktop that has no materials, the page paints the paper.
func (a *app) shellView(c *ui.Context) {
	t := c.Theme()
	// `c.Vibrancy()` says whether a material is actually showing, not whether one
	// was asked for, so an unsupported desktop falls back to the paper.
	vibrant := c.Vibrancy()

	root := ui.Column(c).Fill().Clip()
	page, rail := t.Background.Alpha(1), railFace(t).Alpha(1)
	if !vibrant {
		root.Background(t.Background.Alpha(1))
	} else {
		// The material is the window's. The rail keeps a veil over it so its own
		// text reads whatever is behind the window.
		page = ui.Transparent
		rail = railFace(t).Alpha(0.72)
	}

	// The rail opens and closes by animating its own width, so the page grows
	// into the room rather than jumping. Reduce Motion stills it.
	target := float32(0)
	if !a.railClosed {
		target = railWidth
	}
	open := root.Animate("rail", target, railMotion)
	a.sidebarPixels = open

	root.Children(func() {
		ui.Row(c).Fill().Children(func() {
			if open > 0 {
				ui.Column(c).Key("rail").Width(open).Shrink(0).FillHeight().Clip().
					Background(rail).
					BorderWidth(0, 1, 0, 0).BorderColor(t.Border.Alpha(1)).
					Children(func() {
					ui.Column(c).Width(railWidth).Shrink(0).FillHeight().Children(func() {
						a.sidebar(c)
					})
				})
			}
			ui.Column(c).Grow(1).FillHeight().MinWidth(0).Background(page).
				Children(func() {
					a.content(c)
				})
		})
		// The header floats over the page, and over the page only: it keeps clear
		// of the rail so the rail's own top row is not covered. With the rail
		// closed it keeps clear of the toggle instead, which moves to the window's
		// edge.
		//
		// It starts at the rail and keeps the page's inset, which is what makes
		// its name line up with the column under it: the page's own frame does the
		// same sum.
		ui.Row(c).Absolute().Top(0).Left(open).Right(0).Height(a.headerHeight(c)).
			Padding(0, pageInset(c), 0, pageInset(c)).
			AlignItems(ui.Center).DragWindow().Children(func() {
			ui.Row(c).FillWidth().MaxWidth(pageWidth(c)).MarginX(ui.Auto).
				AlignItems(ui.Center).Children(func() {
					a.topBar(c)
				})
		})
		// The rail's own top row: its toggle, over the rail rather than over the
		// page, so the two never collide.
		//
		// The toggle is drawn at every width, closed rail included: it is the one
		// way back, so hiding it with the rail would strand the window with no
		// sidebar and no way to ask for one. While the rail is closed it sits at
		// the window's edge instead.
		ui.Row(c).Absolute().Top(0).Left(open).Height(a.headerHeight(c)).
			Padding(0, 0, 0, t.Space(2.5)+c.TitleBar().Left).
			AlignItems(ui.Center).Children(func() {
			a.railToggle(c)
		})
	})

	a.toasts(c)
}

// topBar is what the header holds: the app's name, and nothing else.
//
// No search and no account mark. The app has one list to read and the rail
// already names it; a search field over a page that is one playlist would be a
// control looking for a job, and a round initial is a decoration.
func (a *app) topBar(c *ui.Context) {
	s := surface(c)
	t := c.Theme()
	ui.Row(c).FillWidth().Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "Playlist").FontSize(t.Rem(remBody)).FontWeight(600).
			LetterSpacing(trackHeadline(t)).TextColor(s.text())
	})
}

// railToggle is the button that opens and closes the rail.
func (a *app) railToggle(c *ui.Context) {
	s := surface(c)
	t := c.Theme()
	b := ui.ButtonBase(c).Size(t.Space(7), t.Space(7)).Radius(radiusControl(t)).
		Center().Label("Show or hide the sidebar").Tooltip("Show or hide the sidebar")
	face := ui.Transparent
	if b.Hovered() {
		face = s.card()
	}
	b.Background(b.AnimateColor("face", face, motionColor))
	b.AnimatePose(pressPose(b.Pressed()), motionPress)
	b.Children(func() {
		mark := iconChevronLeft
		if a.railClosed {
			mark = iconChevronRight
		}
		ui.Icon(c, mark).Size(t.Space(4), t.Space(4)).TextColor(s.muted())
	})
	if b.Clicked() {
		a.railClosed = !a.railClosed
	}
}

// headerHeight returns the room the floating header takes.
func (a *app) headerHeight(c *ui.Context) float32 {
	t := c.Theme()
	return max(t.Space(14), c.TitleBar().Height)
}

// toasts draws the messages the app shows.
//
// MyGo draws them itself, in the theme's inverted face, so there is nothing to
// build here but the queue of them: the state holds one at a time, because two
// messages at once say neither.
func (a *app) toasts(c *ui.Context) {
	a.mu.Lock()
	notice := a.notice
	a.notice = nil
	a.mu.Unlock()
	if notice == nil {
		return
	}
	c.AddToast(ui.Toast{
		ID:          "notice",
		Title:       notice.title,
		Description: notice.body,
		Type:        notice.kind,
	})
}
