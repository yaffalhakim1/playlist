package library

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// The store is what survives a restart, so the settings that matter have to
// come back: a connected Last.fm account that is forgotten every launch would
// ask the user to connect again each time.

// TestLastFMSettingsSurviveARestart checks the round trip through the file.
func TestLastFMSettingsSurviveARestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")

	first, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	first.SetLastFMCredentials("the-key", "the-secret")
	first.SetLastFMAccount("someone", true)

	second, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	key, secret, user, paused := second.LastFM()
	if key != "the-key" {
		t.Errorf("the key came back as %q", key)
	}
	if secret != "the-secret" {
		t.Errorf("the secret came back as %q", secret)
	}
	if user != "someone" {
		t.Errorf("the account came back as %q", user)
	}
	if !paused {
		t.Error("the pause choice was lost")
	}
}

// TestAFreshStoreHasNoLastFMAccount checks the default: nothing is connected
// until the user says so.
func TestAFreshStoreHasNoLastFMAccount(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "settings.json"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	_, _, user, paused := store.LastFM()
	if user != "" {
		t.Errorf("a fresh store names the account %q", user)
	}
	if paused {
		t.Error("a fresh store reports scrobbling paused")
	}
}

// TestTheSessionKeyIsNotInTheSettings checks the one thing that must not be
// there: the session key is a credential, and the settings are ordinary.
//
// It reads the struct's own tags rather than a written file, because an empty
// field is left out of the JSON and would slip past a check of the bytes.
func TestTheSessionKeyIsNotInTheSettings(t *testing.T) {
	typ := reflect.TypeOf(Data{})
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if strings.EqualFold(name, "sk") || strings.Contains(strings.ToLower(name), "sessionkey") {
			t.Fatalf("Data.%s is %q: the session key belongs in its own file, "+
				"not the settings", field.Name, name)
		}
	}
}
