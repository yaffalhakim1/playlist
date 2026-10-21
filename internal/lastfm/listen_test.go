package lastfm

import (
	"testing"
	"time"
)

// Last.fm's counting rule, and the reason it is measured from positions rather
// than from the clock: a pause, a stall or a seek forward must not make a track
// qualify. Otherwise a track skipped to its end after two seconds would be
// scrobbled as fully heard.

// TestThresholdIsHalfOrFourMinutes checks the rule itself: half the track, or
// four minutes, whichever comes first, and nothing for a track of thirty
// seconds or less.
func TestThresholdIsHalfOrFourMinutes(t *testing.T) {
	cases := []struct {
		name  string
		total time.Duration
		want  time.Duration
		ok    bool
	}{
		{"a short track", 20 * time.Second, 0, false},
		{"exactly thirty seconds", 30 * time.Second, 0, false},
		{"a minute", time.Minute, 30 * time.Second, true},
		{"three minutes", 3 * time.Minute, 90 * time.Second, true},
		{"eight minutes", 8 * time.Minute, 4 * time.Minute, true},
		{"an hour", time.Hour, 4 * time.Minute, true},
		{"an unknown length", 0, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := Threshold(tc.total)
			if ok != tc.ok {
				t.Fatalf("Threshold(%v) ok = %v, want %v", tc.total, ok, tc.ok)
			}
			if ok && got != tc.want {
				t.Errorf("Threshold(%v) = %v, want %v", tc.total, got, tc.want)
			}
		})
	}
}

// TestAPlayQualifiesAtHalfTheTrack checks the ordinary path: samples of a track
// playing steadily, and the moment it qualifies.
func TestAPlayQualifiesAtHalfTheTrack(t *testing.T) {
	l := NewListen(Track{Artist: "A", Title: "T"}, true)
	total := 4 * time.Minute
	start := time.Now()

	// The first sample confirms playback.
	if got := l.Observe(start, 0, total); got != Started {
		t.Fatalf("the first sample gave %v, want Started", got)
	}
	// Half of four minutes is two.
	var qualifiedAt time.Duration
	for i := 1; i <= 240; i++ {
		pos := time.Duration(i) * time.Second
		at := start.Add(pos)
		if got := l.Observe(at, pos, total); got == Qualified {
			qualifiedAt = pos
			break
		}
	}
	if qualifiedAt == 0 {
		t.Fatal("the play never qualified")
	}
	if qualifiedAt < 2*time.Minute-time.Second || qualifiedAt > 2*time.Minute+time.Second {
		t.Errorf("the play qualified at %v, want about half of %v", qualifiedAt, total)
	}
}

// TestAPlayQualifiesOnceOnly checks that a play is submitted once, however long
// it goes on: scrobbling the same listen twice is worse than not at all.
func TestAPlayQualifiesOnceOnly(t *testing.T) {
	l := NewListen(Track{Artist: "A", Title: "T"}, true)
	total := 2 * time.Minute
	start := time.Now()
	l.Observe(start, 0, total)

	qualified := 0
	for i := 1; i <= 300; i++ {
		pos := time.Duration(i) * time.Second
		if l.Observe(start.Add(pos), pos, total) == Qualified {
			qualified++
		}
	}
	if qualified != 1 {
		t.Errorf("the play qualified %d times, want once", qualified)
	}
}

// TestASeekForwardDoesNotQualify checks the rule that matters most: dragging
// the place in the track to the end must not count as listening to it.
func TestASeekForwardDoesNotQualify(t *testing.T) {
	l := NewListen(Track{Artist: "A", Title: "T"}, true)
	total := 4 * time.Minute
	start := time.Now()
	l.Observe(start, 0, total)
	// A second of listening, then a jump to near the end.
	l.Observe(start.Add(time.Second), time.Second, total)
	if got := l.Observe(start.Add(2*time.Second), 3*time.Minute, total); got == Qualified {
		t.Error("a seek forward qualified the play")
	}
}

// TestAPauseDoesNotCount checks that a gap longer than a sample is not credited:
// the machine slept, or the player was held, and no music was heard.
func TestAPauseDoesNotCount(t *testing.T) {
	l := NewListen(Track{Artist: "A", Title: "T"}, true)
	total := 4 * time.Minute
	start := time.Now()
	l.Observe(start, 0, total)
	l.Observe(start.Add(time.Second), time.Second, total)
	// Ten minutes pass with the position standing still, then it resumes.
	l.Observe(start.Add(10*time.Minute), time.Second, total)
	if l.Listened() > 2*time.Second {
		t.Errorf("a ten minute pause counted as %v of listening", l.Listened())
	}
}

// TestBreakStopsCounting checks that a break, which the app reports while the
// player buffers or seeks, resets the sample so the next one does not credit the
// time in between.
func TestBreakStopsCounting(t *testing.T) {
	l := NewListen(Track{Artist: "A", Title: "T"}, true)
	total := 4 * time.Minute
	start := time.Now()
	l.Observe(start, 0, total)
	l.Observe(start.Add(time.Second), time.Second, total)
	before := l.Listened()

	l.Break()
	l.Observe(start.Add(30*time.Second), 30*time.Second, total)
	if grown := l.Listened() - before; grown > time.Second {
		t.Errorf("a break credited %v of listening", grown)
	}
}

// TestATrackWithNoArtistIsNeverFollowed checks the guard: a track Last.fm cannot
// be told about produces no events at all.
func TestATrackWithNoArtistIsNeverFollowed(t *testing.T) {
	l := NewListen(Track{Title: "T"}, true)
	if got := l.Observe(time.Now(), 0, time.Minute); got != Nothing {
		t.Errorf("a track with no artist gave %v", got)
	}
}

// TestAShortTrackIsNeverScrobbled checks that a track under thirty seconds
// produces no qualified event, however long it is listened to.
func TestAShortTrackIsNeverScrobbled(t *testing.T) {
	l := NewListen(Track{Artist: "A", Title: "T"}, true)
	total := 20 * time.Second
	start := time.Now()
	l.Observe(start, 0, total)
	for i := 1; i <= 40; i++ {
		pos := time.Duration(i) * time.Second
		if got := l.Observe(start.Add(pos), pos, total); got == Qualified {
			t.Fatalf("a twenty second track qualified at %v", pos)
		}
	}
}

// TestTheScrobbleKeepsTheStart checks that the submitted play carries the time
// playback began, not the time it qualified: that is the time Last.fm files it
// under.
func TestTheScrobbleKeepsTheStart(t *testing.T) {
	l := NewListen(Track{Artist: "A", Title: "T"}, true)
	total := 2 * time.Minute
	start := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	l.Observe(start, 0, total)
	l.Observe(start.Add(time.Minute), time.Minute, total)

	s := l.Scrobble(total)
	if !s.Started.Equal(start) {
		t.Errorf("the scrobble began at %v, want %v", s.Started, start)
	}
	if s.Track.Duration != total {
		t.Errorf("the scrobble length is %v, want %v", s.Track.Duration, total)
	}
}
