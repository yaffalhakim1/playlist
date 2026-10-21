package src

import (
	"bytes"
	"image/png"
	"testing"

	"youtube-playlist/internal/music"
)

func TestTooltipFor(t *testing.T) {
	cases := []struct {
		track   music.Track
		playing bool
		want    string
	}{
		{track: music.Track{}, playing: false, want: "Playlist"},
		{track: music.Track{Title: "Roar"}, playing: true, want: "Playlist — Roar"},
		{track: music.Track{Title: "Roar"}, playing: false, want: "Playlist — paused"},
	}
	for _, tc := range cases {
		if got := tooltipFor(tc.track, tc.playing); got != tc.want {
			t.Errorf("tooltipFor(%q, %v) = %q, want %q", tc.track.Title, tc.playing, got, tc.want)
		}
	}
}

// TestTrayIcon draws the icon of the notification area, which has to be a
// picture the desktop can show.
func TestTrayIcon(t *testing.T) {
	for _, playing := range []bool{true, false} {
		icon, err := trayIcon(playing)
		if err != nil {
			t.Fatalf("draw the icon (playing %v): %v", playing, err)
		}
		if len(icon) == 0 {
			t.Fatalf("the icon is empty (playing %v)", playing)
		}
		if len(icon) < 100 {
			t.Fatalf("the icon is too small to be a picture: %d bytes", len(icon))
		}
	}
}

// TestTrayIconSaysWhetherItPlays checks that the two states are different
// pictures: the icon is how the tray shows that the music runs or holds.
func TestTrayIconSaysWhetherItPlays(t *testing.T) {
	playing, err := trayIcon(true)
	if err != nil {
		t.Fatalf("playing icon: %v", err)
	}
	paused, err := trayIcon(false)
	if err != nil {
		t.Fatalf("paused icon: %v", err)
	}
	if string(playing) == string(paused) {
		t.Error("the playing and paused icons are the same picture")
	}
}

// TestTrayIconFillsItsCanvas checks that the mark reaches near the edge of the
// bitmap: an icon whose drawing sits in a corner reads as a speck in the tray.
func TestTrayIconFillsItsCanvas(t *testing.T) {
	icon, err := trayIcon(true)
	if err != nil {
		t.Fatalf("icon: %v", err)
	}
	img, err := png.Decode(bytes.NewReader(icon))
	if err != nil {
		t.Fatalf("decode the icon: %v", err)
	}
	bounds := img.Bounds()
	ink := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a > 0x2000 {
				ink++
			}
		}
	}
	total := bounds.Dx() * bounds.Dy()
	share := float64(ink) / float64(total)
	// The mark covers roughly a third of its square. Anything under a tenth
	// means it is drawn too small or off to one side.
	if share < 0.10 {
		t.Errorf("the mark covers %.1f%% of the icon: it is too small to read in a tray", share*100)
	}
}
