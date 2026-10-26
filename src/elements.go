package src

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// picture draws the artwork of a track, or a placeholder while it comes.
//
// The picture is drawn to fill the box whatever its shape: a video thumbnail is
// wide and a cover is square, and both have to fill the same slot without
// stretching. A picture that arrives fades in over the placeholder rather than
// appearing, so a list of covers does not flash as it fills.
func (a *app) picture(c *ui.Context, url string, size, radius float32) ui.Element {
	return a.pictureBox(c, url, size, size, radius)
}

// pictureWide draws a picture in a frame that is wider than it is tall, as the
// thumbnail of a video is.
func (a *app) pictureWide(c *ui.Context, url string, width, height, radius float32) ui.Element {
	return a.pictureBox(c, url, width, height, radius)
}

// pictureBox draws a picture in a box of any shape.
func (a *app) pictureBox(c *ui.Context, url string, width, height, radius float32) ui.Element {
	s := surface(c)
	bitmap := a.art.get(url)
	if bitmap == nil {
		a.art.want(url)
		box := ui.Box(c).Size(width, height).Radius(radius).Background(s.sunken()).Center()
		box.Children(func() {
			ui.Icon(c, iconMusic).Size(minOf2(width, height)*0.28, minOf2(width, height)*0.28).
				TextColor(s.muted())
		})
		return box
	}
	box := ui.Box(c).Key("art-"+url).Size(width, height).Radius(radius).Clip()
	box.Transition(ui.ElementTransition{Animation: &motionFade, Enter: &ui.Motion{Opacity: 0}})
	box.Children(func() {
		ui.Image(c, bitmap).Size(width, height).Fit(ui.Cover)
	})
	return box
}

// clock writes a length as the time it takes.
func clock(duration time.Duration) string {
	if duration <= 0 {
		return "0:00"
	}
	total := int(duration.Seconds())
	hours := total / 3600
	minutes := (total % 3600) / 60
	seconds := total % 60
	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

// trackLine writes what is under the title of a track: its artist and album.
func trackLine(title, artist, album string) string {
	parts := make([]string, 0, 2)
	if artist != "" {
		parts = append(parts, artist)
	}
	if album != "" && album != artist {
		parts = append(parts, album)
	}
	if len(parts) == 0 {
		return title
	}
	return strings.Join(parts, " • ")
}

// minOf2 returns the smaller of two numbers.
func minOf2(a, b float32) float32 {
	if b < a {
		return b
	}
	return a
}
