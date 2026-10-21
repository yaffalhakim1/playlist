package player

import (
	"testing"
	"time"
)

// A track can end two ways, and the queue only knew one of them.
//
// The first is the place in the track reaching its length. The second is the
// stream stopping before that — a dropped connection, an expired link, audio
// shorter than the metadata claims. The place then stops at 2:38 of 3:15 and
// never gets to the comparison, so nothing starts the next track and the music
// simply stops. That is the bug this file pins.

// TestAStreamThatStopsEarlyEndsTheTrack checks the case from the screenshot.
func TestAStreamThatStopsEarlyEndsTheTrack(t *testing.T) {
	cases := []struct {
		name      string
		exhausted bool
		position  time.Duration
		duration  time.Duration
		want      bool
	}{
		{
			name:      "stopped at 2:38 of 3:15",
			exhausted: true,
			position:  158 * time.Second,
			duration:  195 * time.Second,
			want:      true,
		},
		{
			name:      "stopped a second in",
			exhausted: true,
			position:  time.Second,
			duration:  195 * time.Second,
			want:      true,
		},
		{
			name:      "stopped with no length known",
			exhausted: true,
			position:  90 * time.Second,
			duration:  0,
			want:      true,
		},
		{
			name:      "stopped at the very start",
			exhausted: true,
			position:  0,
			duration:  195 * time.Second,
			want:      true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := streamEnded(tc.exhausted, true, tc.position, tc.duration); got != tc.want {
				t.Errorf("streamEnded(%v, %v, %v) = %v, want %v",
					tc.exhausted, tc.position, tc.duration, got, tc.want)
			}
		})
	}
}

// TestAStreamThatIsStillDecodingIsNotTheEnd checks the other half, which is the
// bug the Playing flag caused: a stream waiting on the network is not a track
// that finished.
//
// Without this, every network pause would skip a track.
func TestAStreamThatIsStillDecodingIsNotTheEnd(t *testing.T) {
	for _, position := range []time.Duration{0, time.Second, 158 * time.Second} {
		if streamEnded(false, true, position, 195*time.Second) {
			t.Errorf("a stream still decoding was called ended at %v", position)
		}
	}
}

// TestTheOrdinaryEndIsLeftToTheLengthComparison checks that the two rules do
// not both claim the same moment.
//
// A stream that reaches its length ends by trackEnded. This rule stays out of
// the way there, so the end of a track is reported once, not twice.
func TestTheOrdinaryEndIsLeftToTheLengthComparison(t *testing.T) {
	duration := 195 * time.Second
	for _, position := range []time.Duration{
		duration - 50*time.Millisecond,
		duration,
		duration + time.Second,
	} {
		if !trackEnded(position, duration) {
			t.Fatalf("the length comparison does not claim %v of %v", position, duration)
		}
		if streamEnded(true, true, position, duration) {
			t.Errorf("both rules claim the end at %v of %v", position, duration)
		}
	}
}

// TestAnEarlyEndIsReportedWhileTheTrackIsStillShort checks the boundary: a
// stream that stopped a moment before its length is still an early stop, and
// has to be reported by the rule that handles early stops.
func TestAnEarlyEndIsReportedWhileTheTrackIsStillShort(t *testing.T) {
	duration := 195 * time.Second
	position := duration - 2*time.Second
	if trackEnded(position, duration) {
		t.Fatal("the length comparison claimed a track two seconds short")
	}
	if !streamEnded(true, true, position, duration) {
		t.Error("a stream that stopped two seconds early was not reported ended")
	}
}
