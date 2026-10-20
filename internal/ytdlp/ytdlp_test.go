package ytdlp

import "testing"

func TestSplitTitle(t *testing.T) {
	cases := []struct {
		in         string
		wantTitle  string
		wantArtist string
	}{
		{in: "Bruno Mars - It Will Rain (Lyrics)", wantTitle: "It Will Rain (Lyrics)", wantArtist: "Bruno Mars"},
		{in: "Katy Perry - Roar", wantTitle: "Roar", wantArtist: "Katy Perry"},
		{in: "A song with no artist", wantTitle: "A song with no artist", wantArtist: ""},
		{in: " - something", wantTitle: " - something", wantArtist: ""},
	}
	for _, tc := range cases {
		title, artist := splitTitle(tc.in)
		if title != tc.wantTitle || artist != tc.wantArtist {
			t.Errorf("splitTitle(%q) = %q, %q; want %q, %q",
				tc.in, title, artist, tc.wantTitle, tc.wantArtist)
		}
	}
}

func TestDurationFromURL(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "https://x/videoplayback?expire=1&dur=213.061&itag=251", want: "3m33.061s"},
		{in: "https://x/videoplayback?itag=251", want: "0s"},
		{in: "https://x/videoplayback", want: "0s"},
	}
	for _, tc := range cases {
		got := DurationFromURL(tc.in)
		if got.String() != tc.want {
			t.Errorf("DurationFromURL(%q) = %s, want %s", tc.in, got, tc.want)
		}
	}
}
