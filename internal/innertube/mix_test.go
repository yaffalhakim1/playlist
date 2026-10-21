package innertube

import (
	"net/url"
	"strings"
	"testing"
)

// A mix is the one playlist YouTube builds rather than stores, and the id it is
// asked for by is not the id of anything: it carries the track the mix is built
// around. Getting that id wrong is a request YouTube answers with "This
// playlist type is unviewable", so it is worth pinning.

// TestMixIDCarriesTheVideo checks the shape of the id YouTube Music names a mix
// with: "RDAMVM" and the eleven characters of a video.
func TestMixIDCarriesTheVideo(t *testing.T) {
	const video = "Il-an3K9pjg"
	got := MixID(video)
	if got != "RDAMVM"+video {
		t.Errorf("MixID(%q) = %q, want %q", video, got, "RDAMVM"+video)
	}
	// A mix id is longer than a playlist id, which is what tells them apart.
	if !looksLikeID(got) {
		t.Errorf("%q is not shaped like a playlist id", got)
	}
}

// TestMixIDOfNothingIsEmpty checks the guard: no track, no mix.
func TestMixIDOfNothingIsEmpty(t *testing.T) {
	for _, video := range []string{"", "   "} {
		if got := MixID(video); got != "" {
			t.Errorf("MixID(%q) = %q, want empty", video, got)
		}
	}
}

// TestMixLinkCarriesTheWatchID checks the link.
//
// The watch id is the part that matters: YouTube will not serve a mix from the
// list id alone, because the list is built around the track being watched. A
// link with only list=RDAMVM... is refused as unviewable.
func TestMixLinkCarriesTheWatchID(t *testing.T) {
	const video = "Il-an3K9pjg"
	link := MixLink(video)

	parsed, err := url.Parse(link)
	if err != nil {
		t.Fatalf("the link is not a url: %v", err)
	}
	if got := parsed.Query().Get("v"); got != video {
		t.Errorf("v = %q, want %q: the mix is built around the track", got, video)
	}
	if got, want := parsed.Query().Get("list"), MixID(video); got != want {
		t.Errorf("list = %q, want %q", got, want)
	}
}

// TestMixLinkOfNothingIsEmpty checks the guard.
func TestMixLinkOfNothingIsEmpty(t *testing.T) {
	if got := MixLink(""); got != "" {
		t.Errorf("MixLink(\"\") = %q, want empty", got)
	}
}

// TestAMixLinkIsReadAsItsPlaylist checks that the link the app builds is one the
// loader can take apart again, which is how opening a mix goes through the same
// path as opening any other playlist.
func TestAMixLinkIsReadAsItsPlaylist(t *testing.T) {
	const video = "Il-an3K9pjg"
	id, err := ParsePlaylistID(MixLink(video))
	if err != nil {
		t.Fatalf("the link is not readable: %v", err)
	}
	if id != MixID(video) {
		t.Errorf("the link names %q, want %q", id, MixID(video))
	}
}

// TestMixTitleNamesTheTrack checks what the mix is called: the app shows it in
// the sidebar and on the page, so it has to say what it was built from.
func TestMixTitleNamesTheTrack(t *testing.T) {
	if got := MixTitle("2002 [Official Video]"); got != "Mix - 2002 [Official Video]" {
		t.Errorf("MixTitle = %q", got)
	}
	// A track with no title still gets a name.
	if got := MixTitle(""); got != "Mix" {
		t.Errorf("MixTitle(\"\") = %q, want Mix", got)
	}
}

// TestAWatchLinkWithAMixIsNotTakenForAVideo checks the other direction: the id
// of a mix is read out of a link that carries a video too, which is the shape of
// every mix link.
func TestAWatchLinkWithAMixIsNotTakenForAVideo(t *testing.T) {
	link := "https://www.youtube.com/watch?v=Il-an3K9pjg&list=RDAMVMIl-an3K9pjg&start_radio=1"
	id, err := ParsePlaylistID(link)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !strings.HasPrefix(id, "RDAMVM") {
		t.Errorf("the id came back as %q, want the mix", id)
	}
}
