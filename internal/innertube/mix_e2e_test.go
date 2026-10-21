//go:build e2e

package innertube_test

import (
	"context"
	"testing"
	"time"

	"youtube-playlist/internal/innertube"
	"youtube-playlist/internal/playlist"
)

// A mix is not a playlist that exists: YouTube builds it from the track the
// listener is on. These checks read one for real, which is the only way to know
// that the link the app builds is the one YouTube answers.

// testVideo is a public video with music, used to build a mix from.
const testVideo = "Il-an3K9pjg"

// mixLinkForTest is the link the app builds for a mix from a track.
func mixLinkForTest(videoID string) string { return innertube.MixLink(videoID) }

// TestAMixFromATrackLoads reads the mix the app builds from a track.
func TestAMixFromATrackLoads(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	link := mixLinkForTest(testVideo)
	t.Logf("link: %s", link)

	list, err := playlist.Load(ctx, link)
	if err != nil {
		t.Fatalf("read the mix: %v", err)
	}
	if len(list.Tracks) < 2 {
		t.Fatalf("the mix holds %d tracks, want several", len(list.Tracks))
	}
	t.Logf("%q: %d tracks, first %q", list.Title, len(list.Tracks), list.Tracks[0].Title)

	// The mix is built around the track it was started from, so that track is
	// in it.
	found := false
	for _, track := range list.Tracks {
		if track.ID == testVideo {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("the mix does not hold the track it was started from")
	}
	// Every track has to be playable, which means an id.
	for i, track := range list.Tracks {
		if track.ID == "" {
			t.Errorf("track %d has no id: %q", i, track.Title)
		}
	}
}
