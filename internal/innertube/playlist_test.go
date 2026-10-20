package innertube

import (
	"os"
	"testing"
	"time"
)

func TestParsePlaylistID(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{name: "playlist page", in: "https://www.youtube.com/playlist?list=PLabcdefghijklmnop", want: "PLabcdefghijklmnop"},
		{name: "music playlist page", in: "https://music.youtube.com/playlist?list=OLAK5uy_abcdefghijklmnop", want: "OLAK5uy_abcdefghijklmnop"},
		{name: "watch link with a list", in: "https://www.youtube.com/watch?v=abcdefghijk&list=PLabcdefghijklmnop", want: "PLabcdefghijklmnop"},
		{name: "short link with a list", in: "https://youtu.be/abcdefghijk?list=PLabcdefghijklmnop", want: "PLabcdefghijklmnop"},
		{name: "bare id", in: "PLabcdefghijklmnop", want: "PLabcdefghijklmnop"},
		{name: "browse id", in: "VLPLabcdefghijklmnop", want: "PLabcdefghijklmnop"},
		{name: "spaces around a link", in: "  https://www.youtube.com/playlist?list=PLabcdefghijklmnop  ", want: "PLabcdefghijklmnop"},
		{name: "watch link without a list", in: "https://www.youtube.com/watch?v=abcdefghijk", wantErr: true},
		{name: "a video id is not a playlist", in: "dQw4w9WgXcQ", wantErr: true},
		{name: "nothing", in: "   ", wantErr: true},
		{name: "nonsense", in: "hello there", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParsePlaylistID(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParsePlaylistID(%q) = %q, want an error", tc.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParsePlaylistID(%q): %v", tc.in, err)
			}
			if got != tc.want {
				t.Fatalf("ParsePlaylistID(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestParseResponse reads a real InnerTube answer, saved in testdata.
func TestParseResponse(t *testing.T) {
	data, err := os.ReadFile("testdata/playlist.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	root, err := decodeOrdered(data)
	if err != nil {
		t.Fatalf("decode fixture: %v", err)
	}

	title, artwork := parseHeader(root)
	if title != "#FIFTYDEEP | Music Class of 2024" {
		t.Errorf("title = %q, want %q", title, "#FIFTYDEEP | Music Class of 2024")
	}
	if artwork == "" {
		t.Error("the playlist has no artwork")
	}

	tracks := parseTracks(root)
	if len(tracks) != 3 {
		t.Fatalf("parsed %d tracks, want 3", len(tracks))
	}
	first := tracks[0]
	if first.ID != "8i5Cs66-2vk" {
		t.Errorf("track id = %q, want %q", first.ID, "8i5Cs66-2vk")
	}
	if first.Title != "#FIFTYDEEP Music Class of 2024" {
		t.Errorf("track title = %q, want %q", first.Title, "#FIFTYDEEP Music Class of 2024")
	}
	if first.Artist != "YouTube Music" {
		t.Errorf("track artist = %q, want %q", first.Artist, "YouTube Music")
	}
	if first.Duration != 89*time.Second {
		t.Errorf("track duration = %s, want 1m29s", first.Duration)
	}
	if first.Thumbnail == "" {
		t.Error("the track has no artwork")
	}
	for i, track := range tracks {
		if track.ID == "" {
			t.Errorf("track %d has no id", i)
		}
		if track.Title == "" {
			t.Errorf("track %d has no title", i)
		}
	}
}

func TestSplitByline(t *testing.T) {
	cases := []struct {
		byline     string
		wantArtist string
		wantAlbum  string
	}{
		{byline: "Rick Astley • Whenever You Need Somebody • 1987", wantArtist: "Rick Astley", wantAlbum: "Whenever You Need Somebody"},
		{byline: "Some Artist", wantArtist: "Some Artist"},
		{byline: "", wantArtist: ""},
	}
	for _, tc := range cases {
		artist, album := splitByline(tc.byline)
		if artist != tc.wantArtist || album != tc.wantAlbum {
			t.Errorf("splitByline(%q) = %q, %q; want %q, %q",
				tc.byline, artist, album, tc.wantArtist, tc.wantAlbum)
		}
	}
}

func TestParseDuration(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
	}{
		{in: "3:42", want: 3*time.Minute + 42*time.Second},
		{in: "1:02:03", want: time.Hour + 2*time.Minute + 3*time.Second},
		{in: "45", want: 45 * time.Second},
		{in: "", want: 0},
		{in: "not a length", want: 0},
	}
	for _, tc := range cases {
		if got := parseDuration(tc.in); got != tc.want {
			t.Errorf("parseDuration(%q) = %s, want %s", tc.in, got, tc.want)
		}
	}
}
