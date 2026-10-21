package player

import (
	"testing"
	"time"
)

// A track used to be cut off before its last second: the engine went by the
// player's Playing flag, which drops as the mixer reaches the end of what it
// has decoded, and also drops while the decoder waits on the network. It goes
// by the place in the track now, against its length.

func TestTrackEndsAtItsLength(t *testing.T) {
	cases := []struct {
		name     string
		position time.Duration
		duration time.Duration
		want     bool
	}{
		{"in the middle", 90 * time.Second, 200 * time.Second, false},
		{"near the end", 197 * time.Second, 200 * time.Second, false},
		{"a hair short", 199*time.Second + 700*time.Millisecond, 200 * time.Second, false},
		{"within the slack", 199*time.Second + 950*time.Millisecond, 200 * time.Second, true},
		{"at the end", 200 * time.Second, 200 * time.Second, true},
		{"past the end", 201 * time.Second, 200 * time.Second, true},
		{"no length known", 90 * time.Second, 0, false},
		{"nothing at all", 0, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := trackEnded(tc.position, tc.duration); got != tc.want {
				t.Errorf("trackEnded(%v, %v) = %v, want %v",
					tc.position, tc.duration, got, tc.want)
			}
		})
	}
}

// TestTrackEndedAllowsForTheDrain checks the tolerance: the reported place
// runs a little past the length as the buffer drains, and that is the end.
func TestTrackEndedAllowsForTheDrain(t *testing.T) {
	duration := 200 * time.Second
	if !trackEnded(duration+200*time.Millisecond, duration) {
		t.Error("a track 200ms past its length was not called ended")
	}
	if trackEnded(duration-2*time.Second, duration) {
		t.Error("a track 2s short of its length was called ended")
	}
}

// TestATrackOfUnknownLengthIsNotCutShort checks the case that hurt most: a
// stream whose length the app does not know is never called ended by its
// place, because there is nothing to compare against.
func TestATrackOfUnknownLengthIsNotCutShort(t *testing.T) {
	for _, position := range []time.Duration{0, time.Second, time.Hour} {
		if trackEnded(position, 0) {
			t.Errorf("a track of unknown length was called ended at %v", position)
		}
	}
}
