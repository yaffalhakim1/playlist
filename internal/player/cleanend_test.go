package player

import (
	"testing"
	"time"
)

// A track whose audio is shorter than the length the metadata promised used to
// stop early, and not in a way the listener could see the cause of.
//
// YouTube rounds `lengthSeconds`, so a track advertised as 3:42 can hold 3:40 of
// audio. The decoder writes its last sample and exits cleanly, which leaves the
// place in the track standing still 2 seconds short of the length. The engine
// took that for a broken stream, dropped the cached URL, asked yt-dlp for a new
// one, and started again from where it stopped — three times, with a wait each
// time. What the listener heard was the last seconds of every song played over
// and over, then silence.
//
// The fix is to tell a stream that finished from a stream that broke, and to
// never recover the first.

// TestAFinishedStreamEndsTheTrack checks the case: the decoder reached its own
// end, well before the length the metadata promised.
func TestAFinishedStreamEndsTheTrack(t *testing.T) {
	e := New(nil)
	defer e.Close()
	e.SetQueue(tracks(3))
	e.engineStubResolver()

	// 3:40 of audio in a track advertised as 3:42, stopped for longer than the
	// drain wait.
	duration := 3*time.Minute + 42*time.Second
	position := 3*time.Minute + 40*time.Second
	handled := e.stopped(position, duration, drainTime+time.Second, true, true)
	if !handled {
		t.Fatal("a finished stream was left to the stall rules")
	}
}

// TestAFinishedStreamIsNeverRecovered checks the outcome that matters: the
// stream is not found again, so the track does not replay its last seconds.
func TestAFinishedStreamIsNeverRecovered(t *testing.T) {
	e := New(nil)
	defer e.Close()
	e.SetQueue(tracks(3))
	e.engineStubResolver()

	// Pretend the stream broke and the tries were spent: if a clean finish went
	// down the recovery path, this count would be what the track leaves behind.
	e.mu.Lock()
	e.recoveryFor, e.recoveries = 0, maxRecoveries
	e.mu.Unlock()

	duration := 3 * time.Minute
	e.stopped(duration-20*time.Second, duration, drainTime+time.Second, true, true)

	e.mu.Lock()
	recoveries := e.recoveries
	e.mu.Unlock()
	if recoveries != maxRecoveries {
		t.Errorf("a finished stream went down the recovery path: %d tries left of %d",
			recoveries, maxRecoveries)
	}
}

// TestABrokenStreamIsStillRecovered checks that the fix did not throw away the
// case it was built for: a stream that died with the decoder still running is
// found again.
func TestABrokenStreamIsStillRecovered(t *testing.T) {
	e := New(nil)
	defer e.Close()
	e.SetQueue(tracks(3))
	e.engineStubResolver()

	duration := 5*time.Minute + 37*time.Second
	position := 4*time.Minute + 21*time.Second
	e.mu.Lock()
	e.recoveries = 0
	e.mu.Unlock()

	if !e.stopped(position, duration, stallTime+time.Second, false, false) {
		t.Fatal("a broken stream was not handled at all")
	}
	e.mu.Lock()
	recoveries := e.recoveries
	e.mu.Unlock()
	if recoveries != 1 {
		t.Errorf("a broken stream was not found again: %d recoveries", recoveries)
	}
}

// TestFinishedIsFalseForADecoderThatCannotSay checks the interface guard: a
// decoder that does not report a clean end is never taken for one.
func TestFinishedIsFalseForADecoderThatCannotSay(t *testing.T) {
	if finished(plainDecoder{}) {
		t.Error("a decoder that cannot say it finished was taken for one")
	}
	if finished(finishedDecoder{}) {
		t.Error("a decoder that broke was taken for one that finished")
	}
	if !finished(finishedDecoder{clean: true}) {
		t.Error("a decoder that finished was not taken for one")
	}
}

// plainDecoder is a decoder of the old kind, which only says it is exhausted.
type plainDecoder struct{}

func (plainDecoder) Exhausted() bool { return true }

// finishedDecoder says both, as ffmpeg's does.
type finishedDecoder struct{ clean bool }

func (finishedDecoder) Exhausted() bool      { return true }
func (d finishedDecoder) Finished() bool     { return d.clean }