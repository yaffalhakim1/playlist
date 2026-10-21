package player

import (
	"testing"
	"time"

	"github.com/egoist/mygo/audio"
)

// The stall that ends a track whose stream died must not be confused with the
// stall a person causes: holding the track, or seeking within it. Both stop the
// place in the track moving for as long as they last, and both are not the end.

// silentDecoder gives the audio player something to hold, so the engine's
// controls have a player to act on.
type silentDecoder struct{}

func (silentDecoder) SampleRate() int        { return 48000 }
func (silentDecoder) Channels() int          { return 2 }
func (silentDecoder) Len() int64             { return -1 }
func (silentDecoder) SetPosition(int64) error { return nil }

func (silentDecoder) Read(buf []float32) (int, error) {
	for i := range buf {
		buf[i] = 0
	}
	return len(buf), nil
}

// playingEngine returns an engine with a track loaded and playing, as if it had
// been running for a while.
func playingEngine(t *testing.T) *Engine {
	t.Helper()
	e := New(nil)
	t.Cleanup(e.Close)
	e.SetQueue(tracks(3))
	p := audio.NewPlayer(silentDecoder{})
	e.mu.Lock()
	e.current = p
	e.state = StatePlaying
	e.startedAt = time.Now().Add(-30 * time.Second)
	e.lastPlace = 60 * time.Second
	e.lastMoved = time.Now().Add(-10 * time.Second) // longer than stallTime
	e.mu.Unlock()
	return e
}

// TestHoldingATrackDoesNotStallItOut checks that pausing for longer than the
// stall window does not end the track when it resumes.
//
// The place in a held track stands still by definition. Without a reset on
// resume, the engine would read that as a dead stream and skip the track the
// moment it played again.
func TestHoldingATrackDoesNotStallItOut(t *testing.T) {
	e := playingEngine(t)

	e.Pause()
	e.Resume()

	e.mu.Lock()
	lastMoved, state := e.lastMoved, e.state
	e.mu.Unlock()
	if state != StatePlaying {
		t.Fatalf("the state after resuming is %v", state)
	}
	if time.Since(lastMoved) > stallTime {
		t.Errorf("resuming left the stall clock %v old: the track would end at once",
			time.Since(lastMoved))
	}
}

// TestSeekingClearsTheStall checks that a seek resets the clock: the place
// jumps, so it is not a stall, and the decoder starts again.
func TestSeekingClearsTheStall(t *testing.T) {
	e := playingEngine(t)

	e.Seek(30 * time.Second)

	e.mu.Lock()
	lastMoved := e.lastMoved
	e.mu.Unlock()
	if time.Since(lastMoved) > stallTime {
		t.Errorf("the seek left the stall clock %v old", time.Since(lastMoved))
	}
}

// TestTheStallWindowIsPastTheReadAhead checks the window against the player's
// buffer.
//
// MyGo's player reads about a second ahead. The window has to be comfortably
// longer than that, or the ordinary drain of that buffer would look like a
// stall and end every track early.
func TestTheStallWindowIsPastTheReadAhead(t *testing.T) {
	const readAhead = time.Second
	if stallTime <= 2*readAhead {
		t.Errorf("stallTime is %v, not comfortably past the %v read ahead",
			stallTime, readAhead)
	}
}
