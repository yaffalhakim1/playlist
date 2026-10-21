package src

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"youtube-playlist/internal/lastfm"
)

// Delivery is where a scrobbler earns its keep: a play must be sent once, must
// survive a service that is briefly down, and must not be retried forever when
// Last.fm will never take it.

// stubLastFM is a Last.fm that answers as the test says.
type stubLastFM struct {
	// failFirst is how many calls fail transiently before one succeeds.
	failFirst int32
	// alwaysError, when set, is the API error number every call answers with.
	alwaysError int
	// calls counts the calls that arrived.
	calls int32
	// accepted counts the plays accepted.
	accepted int32
}

func (s *stubLastFM) server(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&s.calls, 1)
		_ = r.ParseForm()
		if s.alwaysError != 0 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": s.alwaysError, "message": "no",
			})
			return
		}
		if left := atomic.AddInt32(&s.failFirst, -1); left >= 0 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		// Count one accepted play per indexed artist.
		for i := 0; ; i++ {
			if r.Form.Get("artist["+itoa(i)+"]") == "" {
				break
			}
			atomic.AddInt32(&s.accepted, 1)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"scrobbles": map[string]any{"accepted": "1", "ignored": "0"},
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

// scrobblerFor returns a connected scrobbler pointed at a stub.
func scrobblerFor(t *testing.T, srv *httptest.Server) *scrobbler {
	t.Helper()
	s := connected(t)
	s.client = lastfm.New("key", "secret")
	s.client.Root = srv.URL
	return s
}

// waitFor polls until cond is true, or the deadline passes.
func waitFor(t *testing.T, timeout time.Duration, cond func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// TestAPlayIsDeliveredAndLeavesTheOutbox checks the ordinary path.
func TestAPlayIsDeliveredAndLeavesTheOutbox(t *testing.T) {
	stub := &stubLastFM{}
	s := scrobblerFor(t, stub.server(t))
	s.outbox.Add(lastfm.Scrobble{
		Track:   lastfm.Track{Artist: "A", Title: "T"},
		Started: time.Now(),
	})

	s.flush()

	if got := s.outbox.Len(); got != 0 {
		t.Errorf("the outbox holds %d plays after delivery", got)
	}
	if got := atomic.LoadInt32(&stub.accepted); got != 1 {
		t.Errorf("the stub accepted %d plays, want 1", got)
	}
}

// TestAPlaySurvivesAServiceThatIsBrieflyDown checks the retry: a play that
// cannot be delivered stays in the outbox rather than being lost.
func TestAPlaySurvivesAServiceThatIsBrieflyDown(t *testing.T) {
	stub := &stubLastFM{failFirst: 1000} // down for the whole test
	s := scrobblerFor(t, stub.server(t))
	s.outbox.Add(lastfm.Scrobble{
		Track:   lastfm.Track{Artist: "A", Title: "T"},
		Started: time.Now(),
	})

	s.flush()

	if got := s.outbox.Len(); got != 1 {
		t.Errorf("a play was lost to a service that was down: %d left", got)
	}
}

// TestAPlayLastFmRefusesIsDropped checks the other side: a play Last.fm will
// never take must not be retried forever.
func TestAPlayLastFmRefusesIsDropped(t *testing.T) {
	// 8 is "operation failed", which is not one worth retrying.
	stub := &stubLastFM{alwaysError: 8}
	s := scrobblerFor(t, stub.server(t))
	s.outbox.Add(lastfm.Scrobble{
		Track:   lastfm.Track{Artist: "A", Title: "T"},
		Started: time.Now(),
	})

	s.flush()

	if got := s.outbox.Len(); got != 0 {
		t.Errorf("a refused play is still queued: %d left", got)
	}
}

// TestABadPlayDoesNotHoldBackTheRest checks the batch fallback: one play Last.fm
// refuses is dropped, and the others are still sent.
func TestABadPlayDoesNotHoldBackTheRest(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		_ = r.ParseForm()
		// A batch is refused; a single good play is accepted.
		if r.Form.Get("artist[1]") != "" {
			_ = json.NewEncoder(w).Encode(map[string]any{"error": 8, "message": "bad batch"})
			return
		}
		if r.Form.Get("track[0]") == "Bad" {
			_ = json.NewEncoder(w).Encode(map[string]any{"error": 8, "message": "bad play"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"scrobbles": map[string]any{"accepted": "1"},
		})
	}))
	defer srv.Close()

	s := scrobblerFor(t, srv)
	now := time.Now()
	s.outbox.Add(lastfm.Scrobble{Track: lastfm.Track{Artist: "A", Title: "Good"}, Started: now})
	s.outbox.Add(lastfm.Scrobble{Track: lastfm.Track{Artist: "B", Title: "Bad"}, Started: now})

	s.flush()

	if got := s.outbox.Len(); got != 0 {
		t.Errorf("the outbox holds %d plays after the fallback", got)
	}
	if got := atomic.LoadInt32(&calls); got < 3 {
		t.Errorf("the batch was not retried play by play: %d calls", got)
	}
}

// TestARefusedSessionStopsDeliveryAndAsksToReconnect checks the one failure the
// user has to act on.
func TestARefusedSessionStopsDeliveryAndAsksToReconnect(t *testing.T) {
	stub := &stubLastFM{alwaysError: 9} // invalid session key
	s := scrobblerFor(t, stub.server(t))
	s.outbox.Add(lastfm.Scrobble{
		Track:   lastfm.Track{Artist: "A", Title: "T"},
		Started: time.Now(),
	})

	s.flush()

	s.mu.Lock()
	reconnect := s.reconnect
	s.mu.Unlock()
	if !reconnect {
		t.Error("a refused session did not ask to reconnect")
	}
	// The play waits: it is still good once the user reconnects.
	if got := s.outbox.Len(); got != 1 {
		t.Errorf("the play was dropped instead of kept: %d left", got)
	}
}

// TestNothingIsDeliveredWithoutASession checks the guard: with no session key
// the plays wait rather than being sent unsigned.
func TestNothingIsDeliveredWithoutASession(t *testing.T) {
	stub := &stubLastFM{}
	s := scrobblerFor(t, stub.server(t))
	s.session = ""
	s.outbox.Add(lastfm.Scrobble{
		Track:   lastfm.Track{Artist: "A", Title: "T"},
		Started: time.Now(),
	})

	s.flush()

	if got := atomic.LoadInt32(&stub.calls); got != 0 {
		t.Errorf("%d calls were made with no session", got)
	}
	if got := s.outbox.Len(); got != 1 {
		t.Errorf("the play was lost: %d left", got)
	}
}
