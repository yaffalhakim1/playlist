package src

import (
	"strings"
	"testing"
)

// Saving a Last.fm key used to look like nothing happened: the button was
// pressed, the client was configured, and the page went on drawing the same
// fields because the frame had already read the state. These tests pin the
// behaviour that replaces it: the key is checked, the result is shown, and the
// secret leaves the page once it is kept.

// TestAShortKeyIsRefusedBeforeACall checks the cheap guard: a pasted key of the
// wrong shape is caught here rather than in a round trip to Last.fm.
func TestAShortKeyIsRefusedBeforeACall(t *testing.T) {
	cases := []struct {
		name   string
		key    string
		secret string
		want   string
	}{
		{"nothing at all", "", "", "Both the key and the secret are needed."},
		{"only a key", strings.Repeat("a", 32), "", "Both the key and the secret are needed."},
		{"only a secret", "", strings.Repeat("a", 32), "Both the key and the secret are needed."},
		{"too short", "abc", strings.Repeat("a", 32), "32 characters"},
		{"a secret too long", strings.Repeat("a", 32), strings.Repeat("b", 40), "32 characters"},
		{"spaces instead", "   ", "   ", "Both the key and the secret are needed."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := &app{scrobble: newScrobbler()}
			a.lastfmKey, a.lastfmSecret = tc.key, tc.secret
			a.saveLastFMCredentials()

			if !strings.Contains(a.lastfmNotice, tc.want) {
				t.Errorf("notice = %q, want it to mention %q", a.lastfmNotice, tc.want)
			}
			// A refused key is not configured, and nothing is being checked.
			if a.lastfmSaving {
				t.Error("a refused key started a check")
			}
			if a.scrobble.configured() {
				t.Error("a refused key was configured")
			}
		})
	}
}

// TestAKeyOfTheRightShapeIsCheckedAndThenKept checks the ordinary path: the
// key is configured, the check runs, and once it passes the fields are cleared
// so the secret is not left on screen.
func TestAKeyOfTheRightShapeIsCheckedAndThenKept(t *testing.T) {
	// A stub that accepts the key, as Last.fm does for a real one.
	srv := lastfmStub(t, "")
	s := newScrobbler()
	s.configure("key", "secret")
	s.client.Root = srv

	a := &app{scrobble: s}
	a.lastfmKey, a.lastfmSecret = strings.Repeat("a", 32), strings.Repeat("b", 32)
	a.saveLastFMCredentials()

	// The check runs off the thread that draws; the test waits for it.
	if !waitUntil(func() bool { return !a.lastfmSaving }, testTimeout) {
		t.Fatal("the check never finished")
	}
	if a.lastfmNotice == "" {
		t.Fatal("nothing was said about the save")
	}
	if strings.Contains(a.lastfmNotice, "did not accept") {
		t.Fatalf("a good key was refused: %s", a.lastfmNotice)
	}
	if a.lastfmKey != "" || a.lastfmSecret != "" {
		t.Error("the fields still hold the credentials after they were kept")
	}
	if !a.lastfmSaved {
		t.Error("the page does not know the key was kept")
	}
}

// TestAKeyLastFmRefusesIsNotKept checks the failure path: the fields stay, so
// the user can correct them, and the reason is shown.
func TestAKeyLastFmRefusesIsNotKept(t *testing.T) {
	srv := lastfmStub(t, "10") // invalid API key
	s := newScrobbler()
	s.configure("key", "secret")
	s.client.Root = srv

	a := &app{scrobble: s}
	a.lastfmKey, a.lastfmSecret = strings.Repeat("a", 32), strings.Repeat("b", 32)
	a.saveLastFMCredentials()

	if !waitUntil(func() bool { return !a.lastfmSaving }, testTimeout) {
		t.Fatal("the check never finished")
	}
	if !strings.Contains(a.lastfmNotice, "did not accept") {
		t.Errorf("notice = %q, want it to say the key was refused", a.lastfmNotice)
	}
	if a.lastfmKey == "" || a.lastfmSecret == "" {
		t.Error("the fields were cleared even though the key was refused")
	}
	if a.lastfmSaved {
		t.Error("a refused key was marked as saved")
	}
	if a.scrobble.configured() {
		t.Error("a refused key is still configured, so every call would fail")
	}
}
