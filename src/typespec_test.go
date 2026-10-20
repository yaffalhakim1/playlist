package src

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestDrawTypeSpecimen draws every role of the type in one place, so the face,
// the weights, the tracking and the leading can be looked at together.
func TestDrawTypeSpecimen(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "playlist-pages")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("make the folder: %v", err)
	}
	a := &app{art: newArtwork()}
	view := func(c *ui.Context) {
		c.SetTheme(a.theme(c))
		t := c.Theme()
		ui.Column(c).Fill().Padding(t.Space(8)).Gap(t.Space(3)).
			Background(t.Background).Children(func() {
			pageName(c, "Paste a link. Hear the music.")
			sectionName(c, "Listen again")
			itemName(c, "It Will Rain (Lyrics) | Wiz Khalifa, Ed Sheeran")
			body(c, "A playlist of YouTube Music, or any playlist of YouTube. The music plays; the video never shows.")
			caption(c, "3 tracks · 13:54 · Bruno Mars • Ballads")
		})
	}
	tester := ui.NewTester(view, 820, 340)
	tester.Frame()
	path := filepath.Join(dir, "type.png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("make the file: %v", err)
	}
	if err := png.Encode(file, tester.Image()); err != nil {
		t.Fatalf("write the picture: %v", err)
	}
	file.Close()
	t.Logf("wrote %s", path)
}
