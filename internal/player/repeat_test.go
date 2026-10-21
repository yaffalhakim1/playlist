package player

import (
	"testing"
	"time"
)

// What happens at the end of a track: the next one plays, the same one plays
// again, or the queue stops. These are the three settings the repeat control
// offers, and the engine has to honour each without the interface's help.

// TestRepeatOneStartsTheTrackAgain checks that RepeatOne replays the track
// rather than moving on.
func TestRepeatOneStartsTheTrackAgain(t *testing.T) {
	e := New(nil)
	defer e.Close()
	e.SetQueue(tracks(3))
	e.engineStubResolver()
	e.SetRepeat(RepeatOne)
	e.cursor = 1

	e.ended()

	e.mu.Lock()
	cursor, state := e.cursor, e.state
	e.mu.Unlock()
	if cursor != 1 {
		t.Fatalf("RepeatOne moved the queue to %d, want the same track (1)", cursor)
	}
	// The track is loaded again rather than seeked, so the state is the wait
	// for its stream.
	if state != StateBuffering {
		t.Errorf("the state is %v, want the wait for the track to load", state)
	}
}

// TestRepeatOffStopsAtTheEndOfTheQueue checks that the last track of a queue
// with repeat off stops, rather than wrapping round to the first.
func TestRepeatOffStopsAtTheEndOfTheQueue(t *testing.T) {
	e := New(nil)
	defer e.Close()
	e.SetQueue(tracks(3))
	e.engineStubResolver()
	e.SetRepeat(RepeatOff)
	e.cursor = len(e.order) - 1

	e.ended()

	e.mu.Lock()
	cursor, state := e.cursor, e.state
	e.mu.Unlock()
	if cursor != len(e.order)-1 {
		t.Errorf("the queue moved to %d, want to stay on the last track", cursor)
	}
	if state != StateIdle {
		t.Errorf("the state is %v, want idle at the end of the queue", state)
	}
}

// TestRepeatAllWrapsToTheStart checks that a queue with repeat all goes round.
func TestRepeatAllWrapsToTheStart(t *testing.T) {
	e := New(nil)
	defer e.Close()
	e.SetQueue(tracks(3))
	e.engineStubResolver()
	e.SetRepeat(RepeatAll)
	e.cursor = len(e.order) - 1

	e.ended()

	e.mu.Lock()
	cursor := e.cursor
	e.mu.Unlock()
	if cursor != 0 {
		t.Errorf("the queue sits at %d, want the first track (0)", cursor)
	}
}

// TestRepeatOneOnAnEmptyQueueDoesNothing checks the guard: the repeat setting
// is reachable from the interface at any time, including before a queue is
// loaded.
func TestRepeatOneOnAnEmptyQueueDoesNothing(t *testing.T) {
	e := New(nil)
	defer e.Close()
	e.SetRepeat(RepeatOne)
	// No queue, no order: ended must not panic or pick a track.
	e.ended()
	if got := e.Current(); got != -1 {
		t.Errorf("a queue that never had a track reports %d", got)
	}
}

// TestATrackThatStopsEarlyIsNotTheSameAsOneThatPlaysOut checks the pair of
// rules against a real pair of numbers: the screenshot's 2:38 of 3:15.
func TestATrackThatStopsEarlyIsNotTheSameAsOneThatPlaysOut(t *testing.T) {
	duration := 3*time.Minute + 15*time.Second
	stopped := 2*time.Minute + 38*time.Second

	if trackEnded(stopped, duration) {
		t.Fatal("the length comparison claims a track that stopped at 2:38 of 3:15")
	}
	if !streamEnded(true, true, stopped, duration) {
		t.Error("a stream that stopped at 2:38 of 3:15 was not reported ended")
	}
	// The stream that did play out is the length comparison's business.
	if streamEnded(true, true, duration, duration) {
		t.Error("a stream that reached its length was claimed twice")
	}
}
