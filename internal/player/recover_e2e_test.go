//go:build e2e

package player

import (
	"os/exec"
	"testing"
	"time"

	"youtube-playlist/internal/music"
	"youtube-playlist/internal/ytdlp"
)

// TestTheStreamIsFoundAgain is the bug on a real stream: the music stopped at
// 4:21 of 5:37 and never came back.
//
// It plays a real track, kills the decoder partway through, and checks that the
// same track plays on from where it stopped rather than the queue moving on.
func TestTheStreamIsFoundAgain(t *testing.T) {
	engine := New(ytdlp.NewResolver())
	defer engine.Close()
	engine.SetQueue([]music.Track{
		{ID: testVideo, Title: "First", Duration: 213 * time.Second},
		{ID: testVideo, Title: "Second", Duration: 213 * time.Second},
	})
	engine.Play(0)

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

	// Let it play, then cut the stream under it.
	time.Sleep(6 * time.Second)
	before := engine.Position()
	if before < 3*time.Second {
		t.Fatalf("the track barely played: %v", before)
	}
	t.Logf("killing the stream at %v", before.Round(time.Second))
	exec.Command("taskkill", "/F", "/IM", "ffmpeg.exe").Run()

	// The engine notices the stall, finds the stream again, and plays on from
	// where it stopped. The queue must not move.
	deadline = time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		position := engine.Position()
		if engine.Current() != 0 {
			t.Fatalf("the queue moved on to track %d instead of recovering", engine.Current())
		}
		if position > before+2*time.Second && engine.State() == StatePlaying {
			t.Logf("the track played on from %v to %v", before.Round(time.Second), position.Round(time.Second))
			return
		}
		time.Sleep(300 * time.Millisecond)
	}
	t.Fatalf("the stream was never found again: stuck at %v of %v, state %v",
		engine.Position().Round(time.Second), engine.Duration().Round(time.Second), engine.State())
}
