package src

import (
	"testing"
	"time"

	"youtube-playlist/internal/music"
	"youtube-playlist/internal/player"
)

// TestApplyingATrackDoesNotDeadlock is the regression test for a hang that
// froze the window the moment a track started.
//
// The engine reports a track, the window applies it while holding the app's
// lock, and the tray is told about it. The tray used to ask the app what was
// playing, which locked the lock the caller already held: Go's mutex is not
// reentrant, so the thread that draws waited on itself and the window stopped
// responding. This test runs that exact path on a deadline, because a
// deadlocked test would otherwise hang the suite instead of failing it.
func TestApplyingATrackDoesNotDeadlock(t *testing.T) {
	a := &app{art: newArtwork(), current: -1}
	a.queue = []music.Track{
		{ID: "one", Title: "It Will Rain", Artist: "Bruno Mars"},
	}
	a.tray = &tray{app: a}

	done := make(chan struct{})
	go func() {
		defer close(done)
		a.apply(player.Event{
			Kind:  player.KindTrack,
			Index: 0,
			Track: a.queue[0],
			State: player.StatePlaying,
		})
		a.apply(player.Event{Kind: player.KindState, State: player.StatePaused})
		a.apply(player.Event{Kind: player.KindState, State: player.StatePlaying})
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("applying a track did not finish: the app's lock is held twice and the window would freeze")
	}

	if a.current != 0 {
		t.Errorf("the track that plays is %d, not 0", a.current)
	}
	if !a.playing {
		t.Error("the app does not think the track plays")
	}
}

// TestTraySaysWhatPlays checks that the tray is told the track and the state,
// and that it keeps what it was told: a tray with no icon still knows what it
// would show.
func TestTraySaysWhatPlays(t *testing.T) {
	a := &app{art: newArtwork(), current: 0}
	a.queue = []music.Track{{ID: "one", Title: "It Will Rain"}}
	a.tray = &tray{app: a}

	a.apply(player.Event{Kind: player.KindTrack, Index: 0, Track: a.queue[0], State: player.StatePlaying})
	if got, want := a.tray.tooltip, "Playlist — It Will Rain"; got != want {
		t.Errorf("the tray says %q, want %q", got, want)
	}

	a.apply(player.Event{Kind: player.KindState, State: player.StatePaused})
	if got, want := a.tray.tooltip, "Playlist — paused"; got != want {
		t.Errorf("a paused track shows %q, want %q", got, want)
	}
}

// TestTrayWithNoIconIsHarmless checks that an app with no tray, and a tray with
// no icon, both take the state without a panic: the notification area is not
// there on every desktop.
func TestTrayWithNoIconIsHarmless(t *testing.T) {
	a := &app{art: newArtwork(), current: 0}
	a.queue = []music.Track{{ID: "one", Title: "It Will Rain"}}

	// No tray at all.
	a.tray = nil
	a.apply(player.Event{Kind: player.KindTrack, Index: 0, Track: a.queue[0], State: player.StatePlaying})

	// A tray the desktop refused to make.
	a.tray = &tray{app: a}
	a.apply(player.Event{Kind: player.KindTrack, Index: 0, Track: a.queue[0], State: player.StatePlaying})

	if a.tray.tooltip == "" {
		t.Error("the tray did not record what plays")
	}
}

// TestTrackLockedMatchesCurrentTrack checks the lock-free reader against the
// locking one: they must name the same track, or the tray and the window would
// disagree about what plays.
func TestTrackLockedMatchesCurrentTrack(t *testing.T) {
	cases := []struct {
		name    string
		current int
		queue   []music.Track
	}{
		{"nothing playing", -1, []music.Track{{ID: "one", Title: "A"}}},
		{"an empty queue", 0, nil},
		{"past the end", 5, []music.Track{{ID: "one", Title: "A"}}},
		{"the first track", 0, []music.Track{{ID: "one", Title: "A"}, {ID: "two", Title: "B"}}},
		{"the second track", 1, []music.Track{{ID: "one", Title: "A"}, {ID: "two", Title: "B"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := &app{art: newArtwork(), current: tc.current, queue: tc.queue}
			a.mu.Lock()
			locked := a.trackLocked()
			a.mu.Unlock()
			if got := a.currentTrack(); got != locked {
				t.Errorf("trackLocked names %+v, currentTrack names %+v", locked, got)
			}
		})
	}
}
