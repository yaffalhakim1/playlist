package player

import (
	"testing"
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
