package src

import (
	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/shells/sidebar"
)

// shellView draws the window as Onevra's sidebar shell draws it, without a
// window.
//
// The shell that opens the real window builds its own view in a method of its
// own package, which a test cannot reach. This is that view: the same theme
// installed first, the same sidebar beside the same content, the same
// floating header over both, and the same room kept for it. The tests draw
// the app through it, so what they check is what the window shows.
func (a *app) shellView(c *ui.Context) {
	c.SetTheme(a.theme(c))

	t := c.Theme()
	header := a.headerHeight(c)
	root := ui.Column(c).Fill().Clip().Background(t.Background.Alpha(1))

	root.Children(func() {
		ui.Row(c).Fill().Children(func() {
			// The sidebar is its own surface, a shade apart from the page, and
			// a hairline marks where it ends.
			ui.Column(c).Width(a.sidebarWidth()).Shrink(0).FillHeight().
				Background(t.Surface.Alpha(1)).BorderWidth(0, 1, 0, 0).
				BorderColor(t.Border.Alpha(0.7)).Children(func() {
				a.sidebar(c)
			})
			ui.Column(c).Grow(1).FillHeight().MinWidth(0).Background(t.Background.Alpha(1)).
				Children(func() {
					a.content(c)
				})
		})
		// The header floats over both, with nothing drawn behind it: what it
		// holds is the actions of the page, and the window controls the
		// platform draws beside them. It keeps the page's own inset and the
		// page's own column, so an action lines up with the page it acts on.
		ui.Row(c).Absolute().Top(0).Left(0).Right(0).Height(header).
			Padding(t.Space(1.5), pageInset(c), t.Space(1.5),
				a.sidebarWidth()+pageInset(c)).
			AlignItems(ui.Center).DragWindow().Children(func() {
			ui.Row(c).FillWidth().MaxWidth(pageWidth(c)).MarginX(ui.Auto).
				Children(func() {
					a.contentTitle(c)
				})
		})
	})

	a.toasts(c)
}

// headerHeight returns the room the floating header takes, as Onevra's shell
// computes it: it accounts for the native window controls and the theme's
// font size.
func (a *app) headerHeight(c *ui.Context) float32 {
	return sidebar.HeaderHeight(c)
}

// sidebarWidth returns the width the sidebar is drawn at.
func (a *app) sidebarWidth() float32 {
	if a.sidebarPixels > 0 {
		return a.sidebarPixels
	}
	return 236
}

// toasts draws the messages the app shows.
func (a *app) toasts(c *ui.Context) {
	a.mu.Lock()
	notice := a.notice
	a.notice = nil
	a.mu.Unlock()
	if notice != nil {
		c.AddToast(ui.Toast{
			ID:          "notice",
			Title:       notice.title,
			Description: notice.body,
			Type:        notice.kind,
		})
	}
	components.Toasts(c)
}
