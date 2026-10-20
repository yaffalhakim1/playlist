package player

import (
	"testing"
	"time"
)

func TestShuffledOrderKeepsEveryTrack(t *testing.T) {
	for _, count := range []int{0, 1, 2, 25} {
		current := -1
		if count > 0 {
			current = count / 2
		}
		order := shuffledOrder(count, current)
		if len(order) != count {
			t.Fatalf("shuffledOrder(%d, %d) has %d tracks, want %d", count, current, len(order), count)
		}
		seen := map[int]bool{}
		for _, item := range order {
			if seen[item] {
				t.Fatalf("shuffledOrder(%d, %d) holds %d twice", count, current, item)
			}
			seen[item] = true
		}
		if current >= 0 && order[0] != current {
			t.Fatalf("shuffledOrder(%d, %d) starts at %d, want the track that plays", count, current, order[0])
		}
	}
}

func TestQueueOrder(t *testing.T) {
	e := New(nil)
	e.SetQueue(tracks(5))
	if got := len(e.order); got != 5 {
		t.Fatalf("the order holds %d tracks, want 5", got)
	}
	for i, item := range e.order {
		if item != i {
			t.Fatalf("the order holds %d at %d, want %d", item, i, i)
		}
	}
	if got := e.Current(); got != -1 {
		t.Fatalf("nothing plays yet, but Current is %d", got)
	}
}

func TestShuffleKeepsTheTrackThatPlays(t *testing.T) {
	e := New(nil)
	e.SetQueue(tracks(6))
	e.order = []int{2, 0, 1, 3, 4, 5}
	e.cursor = 0
	e.shuffle = true
	e.shuffleLocked()
	if e.order[0] != 2 {
		t.Fatalf("the order starts at %d, want the track that plays (2)", e.order[0])
	}
	if len(e.order) != 6 {
		t.Fatalf("the order holds %d tracks, want 6", len(e.order))
	}
}

// TestPlayAtKeepsThePlace checks that a track asked for at a place is played
// from there: this is what makes a track carry on across a restart instead of
// starting the queue again.
//
// The engine is built without a resolver, so a load cannot start: the place is
// settled before anything is resolved, which is what is being checked.
func TestPlayAtKeepsThePlace(t *testing.T) {
	e := New(nil)
	e.SetQueue(tracks(3))
	e.engineStubResolver()

	e.PlayAt(1, 42*time.Second)

	e.mu.Lock()
	cursor := e.cursor
	e.mu.Unlock()
	if cursor != 1 {
		t.Fatalf("the queue sits at %d, want the track asked for (1)", cursor)
	}
}

// TestPlayStartsAtTheBeginning checks that the plain Play still starts a track
// at its beginning, so the place is only carried over when it is asked for.
func TestPlayStartsAtTheBeginning(t *testing.T) {
	e := New(nil)
	e.SetQueue(tracks(3))
	e.engineStubResolver()

	e.Play(2)

	e.mu.Lock()
	cursor := e.cursor
	e.mu.Unlock()
	if cursor != 2 {
		t.Fatalf("the queue sits at %d, want the track asked for (2)", cursor)
	}
}
