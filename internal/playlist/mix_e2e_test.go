//go:build e2e

package playlist

import (
	"context"
	"testing"
	"time"

	"youtube-playlist/internal/innertube"
)

// TestAMixFromATrackLoads checks the mix the app offers under the player: the
// one YouTube builds around a track that is playing, which is the only mix that
// needs no account.
func TestAMixFromATrackLoads(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	link := innertube.MixLink("Il-an3K9pjg")
	list, err := Load(ctx, link)
	if err != nil {
		t.Fatalf("read the mix: %v", err)
	}
	if len(list.Tracks) < 5 {
		t.Errorf("the mix holds %d tracks, which is not a mix", len(list.Tracks))
	}
	found := false
	for _, track := range list.Tracks {
		if track.ID == "" {
			t.Error("a track of the mix has no id, so it cannot play")
		}
		if track.ID == "Il-an3K9pjg" {
			found = true
		}
	}
	if !found {
		t.Error("the mix does not hold the track it was built from")
	}
	// The artist comes from the channel now, so a mix reads like a playlist.
	withArtist := 0
	for _, track := range list.Tracks {
		if track.Artist != "" {
			withArtist++
		}
	}
	t.Logf("%d of %d tracks name an artist", withArtist, len(list.Tracks))
	if withArtist == 0 {
		t.Error("no track of the mix names an artist")
	}
}