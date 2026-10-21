package src

import (
	"strings"
	"testing"

	"youtube-playlist/internal/music"
)

// The panel Discord draws has two text lines and a progress bar. What the app
// puts in them decides whether it reads like the music players people know.

// TestTheStateLineCarriesTheAlbum checks the second line: the artist, then the
// album under it.
//
// Discord has no third text field, but it draws a newline inside `state` as a
// line of its own, which is how a track shows artist and album as the reference
// screenshots do.
func TestTheStateLineCarriesTheAlbum(t *testing.T) {
	cases := []struct {
		name  string
		track music.Track
		want  string
	}{
		{
			name:  "artist and album",
			track: music.Track{Artist: "Justin Bieber", Album: "My World"},
			want:  "Justin Bieber\nMy World",
		},
		{
			name:  "no album",
			track: music.Track{Artist: "Justin Bieber"},
			want:  "Justin Bieber",
		},
		{
			name:  "the album repeats the artist",
			track: music.Track{Artist: "Bruno Mars", Album: "Bruno Mars"},
			want:  "Bruno Mars",
		},
		{
			name:  "no artist",
			track: music.Track{Album: "My World"},
			want:  "My World",
		},
		{
			name:  "neither",
			track: music.Track{},
			want:  "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := stateLine(tc.track); got != tc.want {
				t.Errorf("stateLine = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestTheStateLineHasAtMostTwoLines checks that a title with newlines in it
// cannot push the panel into three lines: the protocol draws one per newline,
// and the app should send the two it means.
func TestTheStateLineHasAtMostTwoLines(t *testing.T) {
	track := music.Track{Artist: "A\nB", Album: "C\nD"}
	got := stateLine(track)
	if lines := strings.Count(got, "\n") + 1; lines > 2 {
		t.Errorf("the state line is %d lines, not at most 2: %q", lines, got)
	}
}
