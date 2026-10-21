package src

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// testTimeout is how long a test waits for work that runs off the thread that
// draws.
const testTimeout = 5 * time.Second

// waitUntil polls cond until it is true, or the timeout passes.
func waitUntil(cond func() bool, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return cond()
}

// lastfmStub is a Last.fm that answers a token request, and fails with apiError
// when it is set.
//
// The stub is local, so a test that saves a key never touches the internet: the
// first run of this feature's tests scrobbled to the real API with a made-up
// key and got a 403, which is what this exists to prevent.
func lastfmStub(t *testing.T, apiError string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if apiError != "" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": apiErrorNumber(apiError), "message": "refused",
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"token": "a-token"})
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

// apiErrorNumber turns the digits of an error code into the number Last.fm
// would answer with.
func apiErrorNumber(digits string) int {
	n := 0
	for _, r := range digits {
		if r < '0' || r > '9' {
			continue
		}
		n = n*10 + int(r-'0')
	}
	return n
}
