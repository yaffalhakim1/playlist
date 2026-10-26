package src

import (
	"github.com/egoist/mygo/ui"
)

// queuePage draws the playlist that is loaded: its head, the strip that narrows
// it, and its tracks.
//
// This is the page the Playlist destination used to be. A playlist is a queue — the
// tracks, in order, with the one that plays marked — so a page of its own was a
// second view of the same list.
func (a *app) queuePage(c *ui.Context) {
	t := c.Theme()
	if len(a.queue) == 0 {
		a.pageFrame(c, false, func() {
			if a.loading {
				a.readingSkeleton(c)
				return
			}
			empty(c, iconQueue, "The queue is empty",
				"Paste a link on the home page, and its tracks line up here in the order they will play.")
		})
		return
	}
	a.pageFrame(c, false, func() {
		ui.Column(c).Fill().Gap(t.Space(6)).Children(func() {
			a.playlistHead(c)
			a.playlistFilter(c)
			a.playlistTable(c)
		})
	})
}
