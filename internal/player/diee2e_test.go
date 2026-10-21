//go:build e2e

// Package player holds the end-to-end check of the audio path: it asks
// yt-dlp for a real stream, decodes it with ffmpeg and reads the samples, to
// show that music really comes out.
//
// It reaches the network, so it does not run with the ordinary tests:
//
//	go test -tags e2e ./internal/player/
package player

import (
	"testing"
	"time"

	"youtube-playlist/internal/music"
	"youtube-playlist/internal/ytdlp"
)

// TestEngineAdvancesWhenTheStreamDies is the bug, on a real stream: a track
// stops at 2:38 of 3:15 and the queue never moves.
//
// It plays a real track, kills the decoder partway through, and checks that the
// engine moves on by itself.
func TestEngineAdvancesWhenTheStreamDies(t *testing.T) {
	engine := New(ytdlp.NewResolver())
	defer engine.Close()
	engine.SetQueue([]music.Track{
		{ID: testVideo, Title: "First", Duration: 213 * time.Second},
		{ID: testVideo, Title: "Second", Duration: 213 * time.Second},
	})
	engine.Play(0)

	// Wait for it to play.
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		if engine.State() == StatePlaying {
			break
		}
		if engine.State() == StateFailed {
			t.Fatal("the track failed to play")
		}
		time.Sleep(200 * time.Millisecond)
	}
	if engine.State() != StatePlaying {
		t.Fatalf("the track did not start: %v", engine.State())
	}
	t.Logf("playing, at %v of %v", engine.Position(), engine.Duration())

	// Let a little audio play, then kill the stream under it.
	time.Sleep(3 * time.Second)
	before := engine.Position()
	engine.mu.Lock()
	if engine.decoder != nil {
		if closer, ok := engine.decoder.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
	}
	engine.mu.Unlock()
	t.Logf("killed the stream at %v", before)

	// The engine has to move to the second track without being asked.
	moved := time.Now().Add(30 * time.Second)
	for time.Now().Before(moved) {
		if engine.Current() == 1 {
			t.Logf("the queue moved on after %v", time.Since(moved.Add(-30*time.Second)))
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("the queue stopped where the stream died: still on track %d at %v of %v",
		engine.Current(), engine.Position(), engine.Duration())
}
