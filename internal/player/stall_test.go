package player

import (
	"testing"
	"time"
)

// The rule that ends a track whose stream died has to be safe for a track whose
// stream did not: MyGo's player decodes about a second ahead, so the decoder
// reaches its end while a second of audio is still waiting to be heard. Ending
// on "the decoder finished" alone would cut that second off every track — the
// very bug the length comparison was introduced to fix.

// TestADecoderThatFinishedIsNotYetTheEnd checks that a finished decoder alone
// does not end the track: the audio it read ahead is still playing.
func TestADecoderThatFinishedIsNotYetTheEnd(t *testing.T) {
	// The stream is over, the place in the track is still moving through the
	// audio read ahead.
	if streamEnded(true, false, 194*time.Second, 195*time.Second) {
		t.Error("a finished decoder ended the track while its audio still played")
	}
}

// TestAStallAloneIsNotTheEnd checks the other half: a place that has stopped
// moving while the decoder is still working is a slow network, not the end.
func TestAStallAloneIsNotTheEnd(t *testing.T) {
	if streamEnded(false, true, 60*time.Second, 195*time.Second) {
		t.Error("a stalled place ended the track while the decoder was still reading")
	}
}

// TestAFinishedDecoderThatStalledIsTheEnd checks the case that is the end: the
// decoder is done and the place has stopped, so nothing more is coming.
func TestAFinishedDecoderThatStalledIsTheEnd(t *testing.T) {
	if !streamEnded(true, true, 158*time.Second, 195*time.Second) {
		t.Error("a finished decoder and a stopped place were not taken for the end")
	}
}

// TestTheOrdinaryEndNeedsNoStall checks that a track that played to its length
// still ends by the length comparison, without waiting to stall.
func TestTheOrdinaryEndNeedsNoStall(t *testing.T) {
	duration := 195 * time.Second
	if !trackEnded(duration, duration) {
		t.Fatal("the length comparison does not claim the end")
	}
	// And the early-end rule stays out of it, so the end is reported once.
	if streamEnded(true, true, duration, duration) {
		t.Error("the early-end rule also claimed an ordinary end")
	}
}
