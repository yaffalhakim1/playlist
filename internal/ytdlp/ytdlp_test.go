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

// TestAMixTrackGetsTheArtistOfItsChannel checks the one field a mix used to
// lose.
//
// A mix is read by yt-dlp, which knows no artist but does know the channel that
// published each track. Without it a mix's rows were a title and nothing else,
// which reads as a broken list and cannot be scrobbled.
func TestAMixTrackGetsTheArtistOfItsChannel(t *testing.T) {
	cases := []struct {
		name             string
		channel, uploader string
		title, want       string
	}{
		{name: "a performer's channel", channel: "Anne-Marie", title: "2002", want: "Anne-Marie"},
		{name: "a topic page", channel: "Anne-Marie - Topic", title: "2002", want: "Anne-Marie"},
		{name: "the uploader when there is no channel", uploader: "Katy Perry", title: "Roar", want: "Katy Perry"},
		{name: "a label", channel: "Various Artists", title: "Roar", want: ""},
		{name: "nothing at all", title: "Roar", want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := artistFromChannel(tc.channel, tc.uploader, tc.title); got != tc.want {
				t.Errorf("artistFromChannel(%q, %q, %q) = %q, want %q",
					tc.channel, tc.uploader, tc.title, got, tc.want)
			}
		})
	}
}
