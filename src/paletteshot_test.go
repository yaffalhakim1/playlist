package src

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestDrawPalettes draws the home page in every palette, in the light and the
// dark, and joins them into one sheet. A palette is a colour decision, so it has
// to be looked at.
func TestDrawPalettes(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "playlist-pages")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("make the folder: %v", err)
	}

	const width, height = 640, 420
	columns := 4
	rows := (len(palettes) + columns - 1) / columns
	sheet := image.NewRGBA(image.Rect(0, 0, width*columns, height*rows))

	for i, p := range palettes {
		for _, dark := range []bool{false, true} {
			if dark {
				continue
			}
			a := loaded(pageHome)
			a.appearance.palette = p.id
			tester := ui.NewTester(a.frame, width, height)
			tester.SetDark(dark)
			tester.Frame()
			shot := tester.Image()
			if shot == nil {
				t.Fatalf("%s drew nothing", p.name)
			}
			col, row := i%columns, i/columns
			offset := image.Pt(col*width, row*height)
			drawOver(sheet, shot, offset)
			t.Logf("drew %s", p.name)
		}
	}

	path := filepath.Join(dir, "palettes.png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("make the file: %v", err)
	}
	if err := png.Encode(file, sheet); err != nil {
		t.Fatalf("write the picture: %v", err)
	}
	file.Close()
	t.Logf("wrote %s", path)
}

// drawOver copies one picture onto another at an offset.
func drawOver(dst *image.RGBA, src *image.RGBA, at image.Point) {
	for y := 0; y < src.Bounds().Dy(); y++ {
		for x := 0; x < src.Bounds().Dx(); x++ {
			px := at.X + x
			py := at.Y + y
			if px >= dst.Bounds().Dx() || py >= dst.Bounds().Dy() {
				continue
			}
			dst.Set(px, py, src.At(x, y))
		}
	}
}

// TestDrawPaletteDark draws the palettes in the dark as well, as a second
// sheet, so both appearances can be checked.
func TestDrawPaletteDark(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "playlist-pages")
	const width, height = 640, 420
	columns := 4
	rows := (len(palettes) + columns - 1) / columns
	sheet := image.NewRGBA(image.Rect(0, 0, width*columns, height*rows))

	for i, p := range palettes {
		a := loaded(pageHome)
		a.appearance.palette = p.id
		tester := ui.NewTester(a.frame, width, height)
		tester.SetDark(true)
		tester.Frame()
		col, row := i%columns, i/columns
		drawOver(sheet, tester.Image(), image.Pt(col*width, row*height))
	}

	path := filepath.Join(dir, fmt.Sprintf("palettes-dark.png"))
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("make the file: %v", err)
	}
	if err := png.Encode(file, sheet); err != nil {
		t.Fatalf("write the picture: %v", err)
	}
	file.Close()
	t.Logf("wrote %s", path)
}
