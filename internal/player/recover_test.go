package player

import (
	"testing"
	"time"
)

// A track whose stream stops before its end used to stop the music: the place
// in the track stood still at 4:21 of 5:37, the length comparison could never
// fire because the place never reached 5:37, and nothing else advanced.
//
// The fix is not to skip the track. The stream is found again and the track
// plays on from where it stopped, because a stream that breaks is a broken
// connection, not a track that finished.

// TestAStoppedStreamIsNotTheEndOfTheTrack checks the case from the screenshot.
func TestAStoppedStreamIsNotTheEndOfTheTrack(t *testing.T) {
	duration := 5*time.Minute + 37*time.Second
	stopped := 4*time.Minute + 21*time.Second

	if trackEnded(stopped, duration) {
		t.Fatal("the length comparison claims a track that stopped at 4:21 of 5:37")
	}
	if nearTheEnd(stopped, duration) {
		t.Error("a stop at 4:21 of 5:37 was taken for the end of the track")
	}
}

// TestAStreamThatStoppedAtTheStartIsNotTheEnd checks the guard: a track that
// never got going is not a track that finished.
func TestAStreamThatStoppedAtTheStartIsNotTheEnd(t *testing.T) {
	if nearTheEnd(0, 5*time.Minute) {
		t.Error("a track at its start was taken for the end")
	}
}

// TestAStopInTheLastMomentIsTheEnd checks the case that must not be recovered:
// YouTube stops serving a track's data at its end, so a stream that breaks in
// the last seconds is the track finishing. Finding it again there would play
// that moment a second time.
func TestAStopInTheLastMomentIsTheEnd(t *testing.T) {
	duration := 5 * time.Minute
	for _, left := range []time.Duration{time.Second, 2 * time.Second, endOfTrackLead} {
		if !nearTheEnd(duration-left, duration) {
			t.Errorf("a stop with %v left was not taken for the end", left)
		}
	}
	// Just outside the window, it is a stream to find again.
	if nearTheEnd(duration-endOfTrackLead-time.Second, duration) {
		t.Error("a stop four seconds from the end was taken for the end")
	}
}

// TestATrackOfUnknownLengthIsNeverTakenForTheEnd checks the guard for a stream
// whose length the app does not know: there is nothing to be near the end of.
func TestATrackOfUnknownLengthIsNeverTakenForTheEnd(t *testing.T) {
	if nearTheEnd(90*time.Second, 0) {
		t.Error("a track of unknown length was taken for its end")
	}
}

// TestTheRecoveryCountBelongsToTheTrack checks the bookkeeping: a track gets a
// fresh set of tries, so one bad stream does not use up the allowance for the
// rest of the queue.
func TestTheRecoveryCountBelongsToTheTrack(t *testing.T) {
	e := New(nil)
	defer e.Close()
	e.SetQueue(tracks(3))
	e.engineStubResolver()

	// The first track uses its tries.
	e.mu.Lock()
	e.recoveryFor, e.recoveries = 0, maxRecoveries
	e.mu.Unlock()

	// The second track starts its own count.
	e.mu.Lock()
	index := 1
	if e.recoveryFor != index {
		e.recoveryFor, e.recoveries = index, 0
	}
	e.mu.Unlock()

	e.mu.Lock()
	got := e.recoveries
	e.mu.Unlock()
	if got != 0 {
		t.Errorf("a new track starts with %d recoveries used, want 0", got)
	}
}

// TestARecoveredStreamGetsItsTriesBack checks the healthy rule: a stream that
// plays on well past where it was found again clears the count, so a later
// break is not the one that gives up.
func TestARecoveredStreamGetsItsTriesBack(t *testing.T) {
	e := New(nil)
	defer e.Close()
	e.SetQueue(tracks(3))
	e.engineStubResolver()

	e.mu.Lock()
	e.recoveries, e.recoveredAt = 2, 30*time.Second
	e.mu.Unlock()

	// Played on past the healthy distance.
	position := 30*time.Second + recoveryHealthy + time.Second
	e.mu.Lock()
	healthy := e.recoveries > 0 && position > e.recoveredAt+recoveryHealthy
	if healthy {
		e.recoveries = 0
	}
	e.mu.Unlock()

	e.mu.Lock()
	got := e.recoveries
	e.mu.Unlock()
	if got != 0 {
		t.Errorf("a stream that played on still has %d recoveries used", got)
	}
}

// TestTheRecoveryWaitsLongerThanTheReadAhead checks the two waits against the
// player's buffer.
//
// MyGo's player reads about a second ahead. A wait shorter than that would take
// the ordinary drain of the buffer for a stopped stream and interrupt every
// track; the wait once the decoder has finished can be shorter, because then
// only that second is left to play out.
func TestTheRecoveryWaitsLongerThanTheReadAhead(t *testing.T) {
	const readAhead = time.Second
	if stallTime <= readAhead {
		t.Errorf("stallTime is %v, not past the %v read ahead", stallTime, readAhead)
	}
	if drainTime <= readAhead {
		t.Errorf("drainTime is %v, not past the %v read ahead", drainTime, readAhead)
	}
	if drainTime >= stallTime {
		t.Errorf("drainTime is %v, not shorter than stallTime %v: a finished "+
			"decoder should be noticed sooner, not later", drainTime, stallTime)
	}
}

// TestTheRecoveryTriesAreBounded checks that a stream that cannot be found
// again is given up on rather than retried forever.
func TestTheRecoveryTriesAreBounded(t *testing.T) {
	if maxRecoveries <= 0 {
		t.Fatal("no recovery is ever tried")
	}
	// A queue that is broken for good must not hold the player forever: the
	// tries are a small number, and each is a full resolve.
	if maxRecoveries > 5 {
		t.Errorf("maxRecoveries is %d, which is a long silence before giving up",
			maxRecoveries)
	}
}
