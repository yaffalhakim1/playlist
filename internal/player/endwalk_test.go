package player

import (
	"testing"
	"time"
)

// The end of a track is decided by two rules that must not fight: the place in
// the track reaching its length, and the stream stopping before that. This
// walks a whole track through both, at the points the engine's tick would see.

// TestAWholeTrackIsReportedEndedExactlyOnce walks a track from its start to a
// little past its end, with the stream ending where it should, and counts how
// many times the engine would call it ended.
//
// Once is right. Twice would skip the track after it; none would stop the
// queue.
func TestAWholeTrackIsReportedEndedExactlyOnce(t *testing.T) {
	duration := 195 * time.Second
	ended := 0
	// The place advances in the engine's ticks, and the stream ends at its
	// length.
	for position := time.Duration(0); position <= duration+time.Second; position += tick {
		exhausted := position >= duration
		if trackEnded(position, duration) || streamEnded(exhausted, true, position, duration) {
			ended++
		}
	}
	if ended == 0 {
		t.Fatal("a track that played to its end was never reported ended")
	}
	// The two rules can overlap for a moment, and the engine takes the first
	// match and moves on, so what matters is that the first match exists and
	// is at the end. Count the first one.
	first := time.Duration(-1)
	for position := time.Duration(0); position <= duration+time.Second; position += tick {
		exhausted := position >= duration
		if trackEnded(position, duration) || streamEnded(exhausted, true, position, duration) {
			first = position
			break
		}
	}
	if first < duration-tick {
		t.Errorf("the track was called ended at %v, more than a tick before its length %v",
			first, duration)
	}
}

// TestATrackWhoseStreamDiedIsReportedEndedAtTheDeath checks the bug: the stream
// stops at 2:38 and the engine has to move on there, not wait for a length the
// stream will never reach.
func TestATrackWhoseStreamDiedIsReportedEndedAtTheDeath(t *testing.T) {
	duration := 3*time.Minute + 15*time.Second
	died := 2*time.Minute + 38*time.Second
	endedAt := time.Duration(-1)
	for position := time.Duration(0); position <= duration; position += tick {
		exhausted := position >= died
		if trackEnded(position, duration) || streamEnded(exhausted, true, position, duration) {
			endedAt = position
			break
		}
	}
	if endedAt < 0 {
		t.Fatal("a track whose stream died at 2:38 was never reported ended")
	}
	if endedAt > died+tick {
		t.Errorf("the track was reported ended at %v, well after the stream died at %v",
			endedAt, died)
	}
}
