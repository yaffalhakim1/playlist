package src

import (
	"bytes"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestFontInUse draws the same words in the app's typeface and in a family
// the desktop does not have, and checks that the two pictures differ: if the
// app's font were not in use, both would fall back to the same system face.
func TestFontInUse(t *testing.T) {
	view := func(family string) func(c *ui.Context) {
		return func(c *ui.Context) {
			th := *c.Theme()
			th.Font = family
			c.SetTheme(&th)
			ui.Column(c).Fill().Padding(20).Children(func() {
				ui.Text(c, "Paste a link. Hear the music.").FontSize(40)
			})
		}
	}
	shot := func(family string) []byte {
		tester := ui.NewTester(view(family), 700, 120)
		tester.Frame()
		img := tester.Image()
		if img == nil {
			t.Fatal("nothing drew")
		}
		return img.Pix
	}
	inter, fallback := shot(familySans), shot("NoSuchFamilyZZZ")
	if bytes.Equal(inter, fallback) {
		t.Fatal("the app's font draws the same as a family that does not exist: it is not in use")
	}
	t.Log("the app draws in its own font")
}
