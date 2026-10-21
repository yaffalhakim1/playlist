package src

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The tests must not reach Last.fm. They run on a build machine with a
// made-up key, and a call that escapes makes the suite slow, flaky and
// dependent on the internet: the first CI run of this feature failed because a
// test scrobbled to the real API and got a 403.

// TestTheScrobblerTestsStayOffline checks that no test file in this package
// builds a client that is not pointed at a stub.
//
// A client whose Root is empty signs for the real endpoint. Every test that
// needs a client has to go through the helper that points one at a closed local
// port, so this reads the source for the one thing that would break that.
func TestTheScrobblerTestsStayOffline(t *testing.T) {
	files, err := filepath.Glob("*_test.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range files {
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		text := string(src)
		// lastfm.New(...) builds a client at the real endpoint unless the
		// caller then sets Root.
		if strings.Contains(text, "lastfm.New(") && !strings.Contains(text, ".Root =") {
			t.Errorf("%s builds a Last.fm client without pointing it at a stub: "+
				"the test would call the real API", name)
		}
		// A scrobbler's own client is configured through configure, which
		// replaces it. A test that configures one and then calls verify or
		// flush would reach the real endpoint unless it pointed the client at
		// a stub first, so it must set Root after configuring.
		if strings.Contains(text, ".configure(") && !strings.Contains(text, ".Root =") {
			t.Errorf("%s configures a scrobbler without pointing its client at "+
				"a stub: the check would reach the real API", name)
		}
	}
}
