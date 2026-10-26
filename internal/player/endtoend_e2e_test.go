//go:build e2e

package player

import (
	"testing"
	"time"

	"youtube-playlist/internal/music"
	"youtube-playlist/internal/ytdlp"
)

// TestARealTrackPlaysToItsEnd is the check for the bug reported as "music
// stopped before the music end".
//
// It plays a real track with the engine's own rules and fails if the queue moves
// on before the audio is over. That is what a stream that broke looks like from
// the outside, and also what a clean end taken for a break looks like: three
// recoveries, then the next track, with the last seconds of the song heard three
// times over.
//
// It is slow on purpose: a real track, from the network, decoded by ffmpeg. The
// pure rules are checked by the ordinary suite; this is the one that would catch
// YouTube changing how it serves audio.
func TestARealTrackPlaysToItsEnd(t *testing.T) {
	// A short track, so the check is a minute or two and not a coffee break.
	const video = "kXYiU_JCYtU"

	e := New(ytdlp.NewResolver())
	defer e.Close()
	e.SetQueue([]music.Track{{ID: video, Title: "a real track"}})
	e.Play(0)

	deadline := time.After(5 * time.Minute)
	var lastPosition, lastDuration time.Duration
	for {
		select {
		case event := <-e.Events():
			switch event.Kind {
			case KindTrack:
				if event.Err != nil {
					t.Fatalf("the track did not start: %v", event.Err)
				}
			case KindPosition:
				lastPosition, lastDuration = event.Position, event.Duration
				if event.Duration > 0 && event.Position >= event.Duration-time.Second {
					t.Logf("the track played to %v of %v",
						event.Position.Round(time.Second), event.Duration.Round(time.Second))
					return
				}
			case KindState:
				if event.State == StateIdle {
					t.Fatalf("the queue went idle at %v of %v: the track stopped before its end",
						lastPosition.Round(time.Second), lastDuration.Round(time.Second))
				}
			}
		case <-deadline:
			t.Fatalf("the track was still at %v of %v after five minutes",
				lastPosition.Round(time.Second), lastDuration.Round(time.Second))
		}
	}
}
