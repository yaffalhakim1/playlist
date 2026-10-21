package lastfm

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// The signature is the whole of Last.fm's authentication, and the one thing
// that is easy to get subtly wrong: the parameters are sorted by name, written
// as name then value with nothing between, and the secret is appended before
// the MD5. A single stray byte makes every call fail with "invalid method
// signature".

// TestSignMatchesLastFmExample checks the signature against the worked example
// in Last.fm's authentication specification.
//
// The spec says: for a call to auth.getSession with api_key "xxxxxxxxxx" and
// token "yyyyyy", and the secret "ilovecher", the signature is
// md5("api_keyxxxxxxxxxxmethodauth.getSessiontokenyyyyyyilovecher").
func TestSignMatchesLastFmExample(t *testing.T) {
	params := map[string]string{
		"api_key": "xxxxxxxxxx",
		"method":  "auth.getSession",
		"token":   "yyyyyy",
	}
	want := md5hex("api_keyxxxxxxxxxxmethodauth.getSessiontokenyyyyyyilovecher")
	if got := sign(params, "ilovecher"); got != want {
		t.Errorf("sign = %q, want %q", got, want)
	}
}

// TestSignSortsByName checks the ordering rule directly.
func TestSignSortsByName(t *testing.T) {
	params := map[string]string{
		"track":   "Song",
		"artist":  "Someone",
		"api_key": "key",
	}
	want := md5hex("api_keykeyartistSomeonetrackSongsecret")
	if got := sign(params, "secret"); got != want {
		t.Errorf("sign = %q, want %q", got, want)
	}
}

// TestSignLeavesOutFormatAndCallback checks the two parameters the spec says are
// not signed. Including them is a common cause of a signature mismatch.
func TestSignLeavesOutFormatAndCallback(t *testing.T) {
	base := map[string]string{"method": "x", "api_key": "k"}
	with := map[string]string{"method": "x", "api_key": "k", "format": "json", "callback": "cb"}
	if sign(base, "s") != sign(with, "s") {
		t.Error("format and callback changed the signature")
	}
}

// TestSignIsStable checks that the same call signs the same way every time,
// which a map iteration order bug would break now and then.
func TestSignIsStable(t *testing.T) {
	params := map[string]string{}
	for _, name := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		params[name] = name
	}
	first := sign(params, "s")
	for i := 0; i < 50; i++ {
		if got := sign(params, "s"); got != first {
			t.Fatalf("sign changed on run %d: %q then %q", i, first, got)
		}
	}
}

// TestASingleScrobbleSendsTheFields checks what goes over the wire: the artist,
// the track, the album, the length in whole seconds, and the time in UTC.
//
// A batch of one is still indexed: Last.fm's documented form uses the array
// notation, and the index is part of the signature.
func TestASingleScrobbleSendsTheFields(t *testing.T) {
	got, err := postCapture(t, func(c *Client) error {
		return c.Scrobble(context.Background(), "session", []Scrobble{{
			Track: Track{
				Artist:   "Justin Bieber",
				Title:    "One Time",
				Album:    "My World",
				Duration: 215*time.Second + 900*time.Millisecond,
			},
			Started: time.Date(2026, 10, 21, 12, 30, 0, 0, time.UTC),
		}})
	})
	if err != nil {
		t.Fatalf("scrobble: %v", err)
	}
	want := map[string]string{
		"method":       "track.scrobble",
		"artist[0]":    "Justin Bieber",
		"track[0]":     "One Time",
		"album[0]":     "My World",
		"duration[0]":  "215", // whole seconds, rounded down
		"timestamp[0]": "1792585800",
	}
	for name, value := range want {
		if got[name] != value {
			t.Errorf("%s = %q, want %q", name, got[name], value)
		}
	}
	if got["sk"] != "session" {
		t.Errorf("sk = %q, want the session key", got["sk"])
	}
	if got["api_sig"] == "" {
		t.Error("the call was sent without a signature")
	}
}

// TestChosenIsSent checks chosenByUser, which tells Last.fm the listener picked
// the track rather than the queue moving on to it.
func TestChosenIsSent(t *testing.T) {
	got, err := postCapture(t, func(c *Client) error {
		return c.Scrobble(context.Background(), "s", []Scrobble{
			{Track: Track{Artist: "A", Title: "One"}, Chosen: true},
			{Track: Track{Artist: "B", Title: "Two"}, Chosen: false},
		})
	})
	if err != nil {
		t.Fatalf("scrobble: %v", err)
	}
	if got["chosenByUser[0]"] != "1" {
		t.Errorf("chosenByUser[0] = %q, want 1", got["chosenByUser[0]"])
	}
	if got["chosenByUser[1]"] != "0" {
		t.Errorf("chosenByUser[1] = %q, want 0", got["chosenByUser[1]"])
	}
}

// TestABatchIsIndexed checks that a batch numbers every field, because that is
// how Last.fm pairs them.
func TestABatchIsIndexed(t *testing.T) {
	got, err := postCapture(t, func(c *Client) error {
		return c.Scrobble(context.Background(), "session", []Scrobble{
			{Track: Track{Artist: "A", Title: "One"}, Started: time.Unix(1000, 0)},
			{Track: Track{Artist: "B", Title: "Two"}, Started: time.Unix(2000, 0)},
		})
	})
	if err != nil {
		t.Fatalf("scrobble: %v", err)
	}
	want := map[string]string{
		"artist[0]":    "A",
		"track[0]":     "One",
		"timestamp[0]": "1000",
		"artist[1]":    "B",
		"track[1]":     "Two",
		"timestamp[1]": "2000",
	}
	for name, value := range want {
		if got[name] != value {
			t.Errorf("%s = %q, want %q", name, got[name], value)
		}
	}
}

// TestNowPlayingHasNoTimestamp checks that the now-playing call does not carry
// the time the track began.
func TestNowPlayingHasNoTimestamp(t *testing.T) {
	got, err := postCapture(t, func(c *Client) error {
		return c.UpdateNowPlaying(context.Background(), "session",
			Track{Artist: "A", Title: "T", Duration: time.Minute})
	})
	if err != nil {
		t.Fatalf("now playing: %v", err)
	}
	if got["method"] != "track.updateNowPlaying" {
		t.Errorf("method = %q", got["method"])
	}
	if _, ok := got["timestamp"]; ok {
		t.Error("the now-playing call carried a timestamp")
	}
}

// TestAnEmptyAlbumIsLeftOut checks that an optional field is omitted rather
// than sent empty, which Last.fm would take as the album name.
func TestAnEmptyAlbumIsLeftOut(t *testing.T) {
	got, err := postCapture(t, func(c *Client) error {
		return c.UpdateNowPlaying(context.Background(), "session", Track{Artist: "A", Title: "T"})
	})
	if err != nil {
		t.Fatalf("now playing: %v", err)
	}
	if _, ok := got["album"]; ok {
		t.Errorf("an empty album was sent: %q", got["album"])
	}
	if _, ok := got["duration"]; ok {
		t.Error("an unknown length was sent")
	}
}

// TestAnInvalidSessionIsReportedAsSuch checks that the error the app has to act
// on is recognisable: only connecting again cures it.
func TestAnInvalidSessionIsReportedAsSuch(t *testing.T) {
	for _, code := range []int{4, 9} {
		c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": code, "message": "Invalid session key",
			})
		})
		err := c.UpdateNowPlaying(context.Background(), "s", Track{Artist: "A", Title: "T"})
		if err == nil {
			t.Fatalf("error %d was not reported", code)
		}
		if !InvalidSession(err) {
			t.Errorf("error %d gave %v, which is not an auth error", code, err)
		}
	}
}

// TestABadKeyIsNotRetried checks that a configuration fault is not treated as a
// transient failure, which would retry it forever.
func TestABadKeyIsNotRetried(t *testing.T) {
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"error": 10, "message": "Invalid API key"})
	})
	err := c.UpdateNowPlaying(context.Background(), "s", Track{Artist: "A", Title: "T"})
	if err == nil {
		t.Fatal("a bad key was not reported")
	}
	if InvalidSession(err) {
		t.Error("a bad key was reported as a session problem")
	}
	if Transient(err) {
		t.Error("a bad key was reported as a transient failure")
	}
	if !strings.Contains(err.Error(), "API key") {
		t.Errorf("the error does not name the key: %v", err)
	}
}

// TestABusyServiceIsTransient checks the errors worth retrying.
func TestABusyServiceIsTransient(t *testing.T) {
	for _, code := range []int{11, 16, 29} {
		c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{"error": code, "message": "busy"})
		})
		err := c.UpdateNowPlaying(context.Background(), "s", Track{Artist: "A", Title: "T"})
		if !Transient(err) {
			t.Errorf("error %d gave %v, which is not transient", code, err)
		}
	}
}

// TestAnUnapprovedTokenIsRecognisable checks the ordinary state while the user
// is still on Last.fm's page: the token exists but is not approved yet.
func TestAnUnapprovedTokenIsRecognisable(t *testing.T) {
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": 14, "message": "Unauthorized Token - This token has not been issued",
		})
	})
	_, err := c.Session(context.Background(), "tok")
	if !NotApproved(err) {
		t.Errorf("an unapproved token gave %v", err)
	}
}

// TestGetTokenAndSession checks the two-step sign-in.
func TestGetTokenAndSession(t *testing.T) {
	var sawGetSession bool
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		switch r.Form.Get("method") {
		case "auth.getToken":
			_ = json.NewEncoder(w).Encode(map[string]string{"token": "tok"})
		case "auth.getSession":
			sawGetSession = true
			if r.Form.Get("token") != "tok" {
				t.Errorf("getSession got token %q", r.Form.Get("token"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"session": map[string]string{"name": "someone", "key": "sk"},
			})
		}
	})

	token, err := c.Token(context.Background())
	if err != nil {
		t.Fatalf("get token: %v", err)
	}
	if token != "tok" {
		t.Errorf("token = %q", token)
	}
	u := c.AuthURL(token)
	if !strings.Contains(u, "api_key=key") || !strings.Contains(u, "token=tok") {
		t.Errorf("the auth url is wrong: %s", u)
	}

	session, err := c.Session(context.Background(), token)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if !sawGetSession {
		t.Fatal("auth.getSession was never called")
	}
	if session.Key != "sk" || session.Name != "someone" {
		t.Errorf("session = %q, user = %q", session.Key, session.Name)
	}
}

// TestAnUnconfiguredClientSaysSo checks the build without a key: it must fail
// clearly rather than sign with an empty secret.
func TestAnUnconfiguredClientSaysSo(t *testing.T) {
	c := New("", "")
	if c.Configured() {
		t.Fatal("a client with no key reports itself configured")
	}
	err := c.UpdateNowPlaying(context.Background(), "s", Track{Artist: "A", Title: "T"})
	if err == nil || !strings.Contains(err.Error(), "application key") {
		t.Errorf("an unconfigured client gave %v", err)
	}
}

// TestATrackNeedsATitleAndAnArtist checks the one rule that decides whether a
// track can be sent at all.
func TestATrackNeedsATitleAndAnArtist(t *testing.T) {
	cases := []struct {
		track Track
		want  bool
	}{
		{Track{Artist: "A", Title: "T"}, true},
		{Track{Artist: "A"}, false},
		{Track{Title: "T"}, false},
		{Track{Artist: " ", Title: "T"}, false},
		{Track{}, false},
	}
	for _, tc := range cases {
		if got := tc.track.Valid(); got != tc.want {
			t.Errorf("Valid(%q, %q) = %v, want %v",
				tc.track.Artist, tc.track.Title, got, tc.want)
		}
	}
}

// postCapture runs one call against a stub and returns the form it posted.
func postCapture(t *testing.T, fn func(*Client) error) (map[string]string, error) {
	t.Helper()
	var got map[string]string
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		got = map[string]string{}
		for name := range r.Form {
			got[name] = r.Form.Get(name)
		}
		_, _ = w.Write([]byte(`{"scrobbles":{"accepted":"1","ignored":"0"}}`))
	})
	return got, fn(c)
}

// testClient returns a client pointed at a stub server.
func testClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := New("key", "secret")
	c.endpoint = srv.URL
	return c, srv
}

func md5hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}
