package ffmpeg

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

// The decoder has to say when its stream has stopped, because the engine cannot
// tell a track that finished from one whose stream died early by the place in
// the track alone: a stream that stopped at 2:38 of 3:15 never reaches 3:15.

// TestExhaustedIsFalseWhileAudioIsComing checks the ordinary case: a stream
// still writing is not exhausted.
func TestExhaustedIsFalseWhileAudioIsComing(t *testing.T) {
	path := writeTone(t, 3*time.Second)
	p, err := NewPCM(context.Background(), path, 3*time.Second)
	if err != nil {
		t.Fatalf("start the decoder: %v", err)
	}
	defer p.Close()

	buf := make([]float32, 1024*2)
	if _, err := p.Read(buf); err != nil {
		t.Fatalf("read the first samples: %v", err)
	}
	if p.Exhausted() {
		t.Error("a stream that just gave samples reports exhausted")
	}
}

// TestExhaustedBecomesTrueAtTheEnd checks that the flag turns on when the
// stream ends, which is what tells the engine the track is over.
func TestExhaustedBecomesTrueAtTheEnd(t *testing.T) {
	path := writeTone(t, time.Second)
	p, err := NewPCM(context.Background(), path, time.Second)
	if err != nil {
		t.Fatalf("start the decoder: %v", err)
	}
	defer p.Close()

	buf := make([]float32, 4096*2)
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := p.Read(buf); err != nil {
			if !p.Exhausted() {
				t.Fatal("the stream ended but the decoder does not report it exhausted")
			}
			return
		}
	}
	t.Fatal("a one second stream did not end in twenty seconds")
}

// TestExhaustedIsFalseAgainAfterASeek checks the case the repeat control leans
// on: a seek starts ffmpeg again, so the stream is live once more.
func TestExhaustedIsFalseAgainAfterASeek(t *testing.T) {
	path := writeTone(t, time.Second)
	p, err := NewPCM(context.Background(), path, 3*time.Second)
	if err != nil {
		t.Fatalf("start the decoder: %v", err)
	}
	defer p.Close()

	// Drain it to the end.
	buf := make([]float32, 4096*2)
	for i := 0; i < 200; i++ {
		if _, err := p.Read(buf); err != nil {
			break
		}
	}
	if !p.Exhausted() {
		t.Fatal("the stream did not report exhausted after playing out")
	}

	if err := p.SetPosition(0); err != nil {
		t.Fatalf("seek back to the start: %v", err)
	}
	if p.Exhausted() {
		t.Error("after a seek the decoder still reports exhausted: " +
			"a repeated track would be skipped at once")
	}
	if _, err := p.Read(buf); err != nil {
		t.Errorf("no audio after the seek: %v", err)
	}
}

// TestExhaustedIsReportedWhenTheProcessDies checks the case a dead link leaves:
// the stream stops writing without a clean end, so the read fails rather than
// returning io.EOF. The decoder has to report that as exhausted too, or the
// engine waits forever for a stream that will never come back.
func TestExhaustedIsReportedWhenTheProcessDies(t *testing.T) {
	path := writeTone(t, 30*time.Second)
	p, err := NewPCM(context.Background(), path, 30*time.Second)
	if err != nil {
		t.Fatalf("start the decoder: %v", err)
	}
	defer p.Close()

	buf := make([]float32, 1024*2)
	if _, err := p.Read(buf); err != nil {
		t.Fatalf("read the first samples: %v", err)
	}

	// Kill it under the read, as a dropped connection does.
	p.mu.Lock()
	_ = p.cmd.Process.Kill()
	p.mu.Unlock()

	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		p.Read(buf)
		if p.Exhausted() {
			return
		}
	}
	t.Fatal("the stream died and the decoder never reported it exhausted")
}

// writeTone writes a file of silence of the given length, for ffmpeg to read,
// so the test needs no network.
func writeTone(t *testing.T, length time.Duration) string {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is not installed")
	}
	path := filepath.Join(t.TempDir(), "tone.wav")
	out, err := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "anullsrc=r=48000:cl=stereo",
		"-t", strconv.FormatFloat(length.Seconds(), 'f', 2, 64), "-y", path).CombinedOutput()
	if err != nil {
		t.Skipf("could not write the test file: %v\n%s", err, out)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("the test file was not written: %v", err)
	}
	return path
}
