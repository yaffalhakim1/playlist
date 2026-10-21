package src

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestDrawDeleteQuestion draws the confirmation, so the wording and the button
// order can be looked at: a destructive dialog is worth seeing before it ships.
func TestDrawDeleteQuestion(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "playlist-pages")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("make the folder: %v", err)
	}
	a := loaded(pageHome)
	a.store = openStore(t)
	a.askDelete("link-a", "Mix - Bruno Mars - It Will Rain")

	tester := ui.NewTester(a.frame, 1280, 840)
	tester.Frame()
	path := filepath.Join(dir, "delete-question.png")
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
