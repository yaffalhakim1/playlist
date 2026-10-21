package src

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestDrawSharing draws the Look page tall enough to show the Sharing group,
// so the Discord switch can be looked at.
func TestDrawSharing(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "playlist-pages")
	a := loaded(pageAppearance)
	tester := ui.NewTester(a.frame, 1280, 1200)
	tester.Frame()
	path := filepath.Join(dir, "look-sharing.png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("make the file: %v", err)
	}
	if err := png.Encode(file, tester.Image()); err != nil {
		t.Fatalf("write the picture: %v", err)
	}
	file.Close()
	if !tester.HasText("Discord") {
		t.Errorf("the Look page has no Discord switch: %v", tester.Texts())
	}
	t.Logf("wrote %s", path)
}
