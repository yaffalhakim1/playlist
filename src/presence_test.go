package src

import (
	"strings"
	"testing"
	"time"

	"youtube-playlist/internal/music"
)

// The Discord presence shows what plays, and it has to stay out of the way when
// it cannot: no Discord running, no application id, a socket that closed. None
// of those may stop the music or hold up a frame.

// TestPresenceIsOffUntilItIsTurnedOn checks the default: nothing is sent unless
// the user asked for it.
func TestPresenceIsOffUntilItIsTurnedOn(t *testing.T) {
	p := &presence{}
	if p.enabledNow() {
		t.Error("presence starts on")
	}
	// A track shown while it is off must not connect anywhere.
	p.show(music.Track{ID: "one", Title: "It Will Rain"}, true)
	if p.shown != "" {
		t.Errorf("presence recorded %q while it was off", p.shown)
	}
}

// TestPresenceRemembersTheTrackItShowed checks that the same track is not sent
// twice: the engine reports the same track on every state change, and a socket
// write per report would be noise.
func TestPresenceRemembersTheTrackItShowed(t *testing.T) {
	p := &presence{}
	p.enabled = true

	track := music.Track{ID: "one", Title: "It Will Rain", Artist: "Bruno Mars"}
	// The send is asynchronous and will fail to connect in a test, which is
	// fine: what matters is the key the presence records.
	p.show(track, true)
	first := p.shown

	p.show(track, true)
	if p.shown != first {
		t.Errorf("the same track was recorded twice: %q, then %q", first, p.shown)
	}

	// Pausing is a different state, so it is a new frame.
	p.show(track, false)
	if p.shown == first {
		t.Error("pausing did not count as a change")
	}
}

// TestPresenceClearsWhenNothingPlays checks that an empty track takes the panel
// down rather than leaving the last one up.
func TestPresenceClearsWhenNothingPlays(t *testing.T) {
	p := &presence{}
	p.enabled = true
	p.shown = "something"

	p.show(music.Track{}, false)

	if p.shown != "" {
		t.Errorf("an empty track left %q on the panel", p.shown)
	}
}

// TestPresenceCountsFromTheStartOfTheTrack checks the clock: Discord counts from
// when the track began, and a new track resets it rather than inheriting the
// last one's.
func TestPresenceCountsFromTheStartOfTheTrack(t *testing.T) {
	p := &presence{}
	p.enabled = true

	p.show(music.Track{ID: "one", Title: "A"}, true)
	if p.started.IsZero() {
		t.Fatal("the first track did not start a clock")
	}
	first := p.started

	p.resetTrack()
	if !p.started.IsZero() {
		t.Error("resetting the track left the old clock")
	}

	// A second later, so the two clocks can be told apart.
	time.Sleep(2 * time.Millisecond)
	p.show(music.Track{ID: "two", Title: "B"}, true)
	if p.started.Equal(first) {
		t.Error("the second track kept the first track's clock")
	}
}

// TestTurningPresenceOffClearsIt checks that turning the switch off takes the
// panel down and forgets the track, so turning it back on shows the current one.
func TestTurningPresenceOffClearsIt(t *testing.T) {
	p := &presence{}
	p.enabled = true
	p.shown = "something"
	p.started = time.Now()

	p.enable(false)

	if p.enabledNow() {
		t.Error("presence is still on")
	}
	if p.shown != "" {
		t.Errorf("turning it off left %q on the panel", p.shown)
	}
	if !p.started.IsZero() {
		t.Error("turning it off left the clock running")
	}
}

// TestPresenceWithNoClientIDStaysOff checks the case a build without an
// application id hits: the switch is on in the settings, but nothing connects.
func TestPresenceWithNoClientIDStaysOff(t *testing.T) {
	if discordClientID != "" {
		t.Skip("this build has a Discord application id")
	}
	a := &app{art: newArtwork()}
	a.presence = &presence{}
	a.setDiscord(true)

	if a.presence.enabledNow() {
		t.Error("presence turned on with no application id to register it under")
	}
	if !a.appearance.discord {
		t.Error("the choice was not kept, so the switch would not stay on")
	}
}

// TestTheDiscordSettingIsRemembered checks that the choice survives a restart.
func TestTheDiscordSettingIsRemembered(t *testing.T) {
	store := openStore(t)
	a := &app{art: newArtwork(), store: store, presence: &presence{}}

	a.setDiscord(true)
	if !store.Discord() {
		t.Error("turning Discord on was not kept")
	}

	a.setDiscord(false)
	if store.Discord() {
		t.Error("turning Discord off was not kept")
	}
}

// TestDiscordHintSaysWhyItCannotBeTurnedOn checks the words: a build without an
// application id has to say so, rather than offering a switch that does nothing.
func TestDiscordHintSaysWhyItCannotBeTurnedOn(t *testing.T) {
	a := &app{art: newArtwork()}
	hint := a.discordHint()
	if discordClientID == "" && !strings.Contains(hint, "application id") {
		t.Errorf("the hint does not say why the switch cannot work: %q", hint)
	}
	if discordClientID != "" && strings.Contains(hint, "cannot") {
		t.Errorf("the hint says it cannot work, but this build has an id: %q", hint)
	}
}


