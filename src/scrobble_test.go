package src

import (
	"path/filepath"
	"testing"
	"time"

	"youtube-playlist/internal/lastfm"
	"youtube-playlist/internal/music"
)

// The scrobbler's job is to turn the player's positions into a play Last.fm
// accepts, exactly once, and to keep it if it cannot be delivered. These tests
// cover the wiring between the engine's reports and that rule.

// aTrack is what the queue holds in these tests.
func aTrack() music.Track {
	return music.Track{
		ID:       "abc",
		Title:    "One Time",
		Artist:   "Justin Bieber",
		Album:    "My World",
		Duration: 4 * time.Minute,
	}
}

// connected returns a scrobbler that is signed in, with an outbox that keeps
// nothing on disk.
func connected(t *testing.T) *scrobbler {
	t.Helper()
	s := newScrobbler()
	s.configure("key", "secret")
	s.user = "someone"
	s.session = "session"
	return s
}

// TestNothingIsSentWithoutAnAccount checks the guard that matters most: an app
// that was never connected must not follow a play, so nothing is ever queued.
func TestNothingIsSentWithoutAnAccount(t *testing.T) {
	s := newScrobbler()
	s.configure("key", "secret")
	s.begin(aTrack(), true)
	if s.listen != nil {
		t.Error("a play was followed without an account")
	}
}

// TestAPlayIsQueuedOnceItQualifies checks the whole path: a track playing
// steadily is added to the outbox when half of it has been heard, and only then.
func TestAPlayIsQueuedOnceItQualifies(t *testing.T) {
	s := connected(t)
	track := aTrack()
	s.begin(track, true)

	start := time.Now()
	// A quarter of the way in: not yet.
	for i := 0; i <= 60; i++ {
		s.observe(start.Add(time.Duration(i)*time.Second), time.Duration(i)*time.Second, track.Duration)
	}
	if got := s.outbox.Len(); got != 0 {
		t.Fatalf("a quarter of the track queued %d plays", got)
	}

	// Past the halfway mark: one play.
	for i := 61; i <= 130; i++ {
		s.observe(start.Add(time.Duration(i)*time.Second), time.Duration(i)*time.Second, track.Duration)
	}
	if got := s.outbox.Len(); got != 1 {
		t.Fatalf("half the track queued %d plays, want 1", got)
	}
}

// TestAPlayIsNeverQueuedTwice checks that a track left playing for hours is one
// play, not one every time the threshold is crossed.
func TestAPlayIsNeverQueuedTwice(t *testing.T) {
	s := connected(t)
	track := aTrack()
	s.begin(track, true)

	start := time.Now()
	for i := 0; i <= 400; i++ {
		// The position wraps, as a repeated track does.
		pos := time.Duration(i%240) * time.Second
		s.observe(start.Add(time.Duration(i)*time.Second), pos, track.Duration)
	}
	if got := s.outbox.Len(); got > 1 {
		t.Errorf("one play queued %d scrobbles", got)
	}
}

// TestPausingStopsFollowing checks that turning scrobbling off leaves the play
// out entirely: it was not followed from its start, so it cannot be submitted.
func TestPausingStopsFollowing(t *testing.T) {
	s := connected(t)
	track := aTrack()
	s.begin(track, true)

	s.setPaused(true)
	if s.listen != nil {
		t.Error("a play was still followed while scrobbling was paused")
	}

	// And nothing qualifies while paused.
	start := time.Now()
	for i := 0; i <= 300; i++ {
		s.observe(start.Add(time.Duration(i)*time.Second), time.Duration(i)*time.Second, track.Duration)
	}
	if got := s.outbox.Len(); got != 0 {
		t.Errorf("a paused scrobbler queued %d plays", got)
	}
}

// TestABreakStopsCounting checks the app's own guard: while the track is held,
// the time does not count towards the threshold.
func TestABreakStopsCounting(t *testing.T) {
	s := connected(t)
	track := aTrack()
	s.begin(track, true)

	start := time.Now()
	s.observe(start, 0, track.Duration)
	s.observe(start.Add(time.Second), time.Second, track.Duration)
	// Held for ten minutes.
	s.breakListen()
	s.observe(start.Add(10*time.Minute), time.Second, track.Duration)

	if got := s.listen.Listened(); got > 2*time.Second {
		t.Errorf("a hold counted as %v of listening", got)
	}
}

// TestATrackWithNoArtistIsNotFollowed checks that a playlist without artists
// does not queue plays Last.fm would reject.
func TestATrackWithNoArtistIsNotFollowed(t *testing.T) {
	s := connected(t)
	track := aTrack()
	track.Artist = ""
	s.begin(track, false)

	start := time.Now()
	for i := 0; i <= 300; i++ {
		s.observe(start.Add(time.Duration(i)*time.Second), time.Duration(i)*time.Second, track.Duration)
	}
	if got := s.outbox.Len(); got != 0 {
		t.Errorf("a track with no artist queued %d plays", got)
	}
}

// TestTheSessionIsKeptApartFromTheSettings checks where the credential goes: a
// file of its own, not the settings, which are ordinary and copied about.
func TestTheSessionIsKeptApartFromTheSettings(t *testing.T) {
	dir := t.TempDir()
	s := newScrobbler()
	s.open(dir)

	if s.store.Path == "" {
		t.Fatal("the session has nowhere to go")
	}
	if got, want := filepath.Base(s.store.Path), "lastfm-session"; got != want {
		t.Errorf("the session is kept as %q, want %q", got, want)
	}
	if s.store.Path == filepath.Join(dir, "settings.json") {
		t.Error("the session is kept in the settings")
	}
}

// TestTheSessionSurvivesARestart checks that a connected account stays connected:
// the key is read back on the next run.
func TestTheSessionSurvivesARestart(t *testing.T) {
	dir := t.TempDir()
	first := newScrobbler()
	first.open(dir)
	first.configure("key", "secret")
	if err := first.store.Save("the-session"); err != nil {
		t.Fatalf("save the session: %v", err)
	}

	second := newScrobbler()
	second.open(dir)
	second.configure("key", "secret")
	second.restore("someone", false)

	second.mu.Lock()
	session, user := second.session, second.user
	second.mu.Unlock()
	if session != "the-session" {
		t.Errorf("the session came back as %q", session)
	}
	if user != "someone" {
		t.Errorf("the account came back as %q", user)
	}
}

// TestAMissingKeyAsksToReconnect checks the case of settings that say connected
// while the key file is gone: the app has to ask, not send calls that fail.
func TestAMissingKeyAsksToReconnect(t *testing.T) {
	dir := t.TempDir()
	s := newScrobbler()
	s.open(dir)
	s.configure("key", "secret")

	s.restore("someone", false)

	s.mu.Lock()
	reconnect := s.reconnect
	s.mu.Unlock()
	if !reconnect {
		t.Error("a missing session key did not ask to reconnect")
	}
}

// TestDisconnectingForgetsEverything checks the whole of it: the key leaves the
// disk, the account leaves the settings, and the plays waiting for it are
// dropped rather than sent as whoever connects next.
func TestDisconnectingForgetsEverything(t *testing.T) {
	dir := t.TempDir()
	s := connected(t)
	s.open(dir)
	if err := s.store.Save("the-session"); err != nil {
		t.Fatal(err)
	}
	s.outbox.Add(lastfm.Scrobble{
		Track:   lastfm.Track{Artist: "A", Title: "T"},
		Started: time.Now(),
	})

	var savedUser string
	var savedPaused bool
	s.onSave = func(user string, paused bool) {
		savedUser, savedPaused = user, paused
	}

	s.disconnect()

	if key, _ := s.store.Load(); key != "" {
		t.Error("the session key is still on disk after disconnecting")
	}
	if got := s.outbox.Len(); got != 0 {
		t.Errorf("%d plays were left for the next account", got)
	}
	if savedUser != "" {
		t.Errorf("the settings still name %q", savedUser)
	}
	_ = savedPaused
	if s.scrobblingNow() {
		t.Error("the scrobbler still reports itself as connected")
	}
}

// TestAnInvalidSessionAsksToReconnect checks the one failure the app acts on.
func TestAnInvalidSessionAsksToReconnect(t *testing.T) {
	s := connected(t)
	s.failed(lastfm.ErrInvalidSession)

	s.mu.Lock()
	reconnect, notice := s.reconnect, s.notice
	s.mu.Unlock()
	if !reconnect {
		t.Error("a refused session did not ask to reconnect")
	}
	if notice == "" {
		t.Error("no notice was raised")
	}
}

// TestAnOrdinaryFailureDoesNotAskToReconnect checks the other side: a busy
// service or a lost network is not a reason to make the user sign in again.
func TestAnOrdinaryFailureDoesNotAskToReconnect(t *testing.T) {
	s := connected(t)
	s.failed(errTransientForTest())

	s.mu.Lock()
	reconnect := s.reconnect
	s.mu.Unlock()
	if reconnect {
		t.Error("a transient failure asked to reconnect")
	}
}

// errTransientForTest is a failure that may pass on its own.
func errTransientForTest() error {
	// A network error, as the client reports one.
	return &netErr{}
}

type netErr struct{}

func (*netErr) Error() string { return "the network is down" }
