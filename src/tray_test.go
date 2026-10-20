package src

import (
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
	icon, err := trayIcon()
	if err != nil {
		t.Fatalf("draw the icon: %v", err)
	}
	if len(icon) == 0 {
		t.Fatal("the icon is empty")
	}
	if len(icon) < 100 {
		t.Fatalf("the icon is too small to be a picture: %d bytes", len(icon))
	}
}
