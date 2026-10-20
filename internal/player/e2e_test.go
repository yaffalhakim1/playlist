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
	"context"
	"math"
	"testing"
	"time"

	"youtube-playlist/internal/ffmpeg"
	"youtube-playlist/internal/music"
	"youtube-playlist/internal/ytdlp"
)

// testVideo is a public video with music, used to check the audio path.
const testVideo = "dQw4w9WgXcQ"

// TestAudioPath decodes a real stream and reads it, which shows that the
// whole path works: yt-dlp finds the stream, ffmpeg decodes it, and the
// samples are sound and not silence.
func TestAudioPath(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	resolver := ytdlp.NewResolver()
	url, err := resolver.Resolve(ctx, testVideo)
	if err != nil {
		t.Fatalf("resolve the stream: %v", err)
	}
	t.Logf("stream: %.120s...", url)

	decoder, err := ffmpeg.NewPCM(ctx, url, 0)
	if err != nil {
		t.Fatalf("start the decoder: %v", err)
	}
	defer decoder.Close()

	buf := make([]float32, 4096*2)
	var peak, sum float64
	frames := 0
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		n, err := decoder.Read(buf)
		for i := 0; i < n; i++ {
			v := math.Abs(float64(buf[i]))
			if v > peak {
				peak = v
			}
			sum += v * v
		}
		frames += n / 2
		if err != nil {
			break
		}
		if frames > 48000 {
			break
		}
	}
	if frames == 0 {
		t.Fatal("no samples came out of the stream")
	}
	rms := math.Sqrt(sum / float64(frames))
	t.Logf("read %d frames in %v, peak %.3f, rms %.4f", frames, time.Since(deadline.Add(-10*time.Second)), peak, rms)
	if peak < 0.01 {
		t.Fatalf("the stream is silent (peak %.4f)", peak)
	}
}

// TestEnginePlays plays a track through the whole chain: the engine resolves
// the audio, ffmpeg decodes it, and MyGo's audio package sends it to the
// speakers. The place in the track has to move.
func TestEnginePlays(t *testing.T) {
	engine := New(ytdlp.NewResolver())
	defer engine.Close()
	engine.SetQueue([]music.Track{{
		ID:       testVideo,
		Title:    "Test",
		Duration: 213 * time.Second,
	}})
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
		t.Fatalf("the track did not start, the state is %v", engine.State())
	}

	start := engine.Position()
	time.Sleep(3 * time.Second)
	if moved := engine.Position(); moved <= start {
		t.Fatalf("the track stands still: %s after %s", moved, start)
	}
	t.Logf("the track plays: %s of %s", engine.Position(), engine.Duration())
}
