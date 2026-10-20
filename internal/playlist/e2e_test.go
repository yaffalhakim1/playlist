//go:build e2e

package playlist

import (
	"context"
	"testing"
	"time"
)

// The links the check reads. The first is a playlist of YouTube Music, which
// InnerTube knows; the second is a mix YouTube generates, which only yt-dlp
// can read.
const (
	musicPlaylist = "https://music.youtube.com/playlist?list=PL30kw0f7AkXpwhQ9kwk9K0w4EAqBgXk9-"
	mixPlaylist   = "https://youtube.com/playlist?list=RDhpX6oorlyow&playnext=1&si=UhyUlaXB55ojuNEK"
)

// TestLoadMusicPlaylist reads a playlist of YouTube Music.
func TestLoadMusicPlaylist(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	playlist, err := Load(ctx, musicPlaylist)
	if err != nil {
		t.Fatalf("read the playlist: %v", err)
	}
	t.Logf("%q: %d tracks, artist of the first: %q", playlist.Title, len(playlist.Tracks), playlist.Tracks[0].Artist)
	if len(playlist.Tracks) == 0 {
		t.Fatal("the playlist has no tracks")
	}
	if playlist.Tracks[0].Duration <= 0 {
		t.Error("the first track has no length")
	}
}

// TestLoadMix reads a mix YouTube generated, which InnerTube cannot show.
func TestLoadMix(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	playlist, err := Load(ctx, mixPlaylist)
	if err != nil {
		t.Fatalf("read the mix: %v", err)
	}
	t.Logf("%q: %d tracks, first: %q", playlist.Title, len(playlist.Tracks), playlist.Tracks[0].Title)
	if len(playlist.Tracks) < 10 {
		t.Fatalf("the mix holds %d tracks, want more", len(playlist.Tracks))
	}
}
