package src

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// The Look page's switches were reported as doing nothing: the knob moved and
// the setting never changed.
//
// Two things were wrong. MyGo flips a switch's bound value as the control is
// built, so the value has to outlive the frame rather than being a fresh local
// each time; and a click has to land on the switch, which sits at the right of
// its row, not on the middle of the label, which is what tester.Click(name)
// does.
//
// These tests drive the switch through the same binding the page uses, so they
// check the wiring rather than the hit area.

// flipDiscord turns the Discord switch the way a click does, through the binding
// the page gives it, and reports the value the page would read back.
func flipDiscord(a *app) bool {
	// appearancePage seeds the field from the real state as it draws, then reads
	// it back after. This is the same round trip, with the switch's own flip in
	// between.
	a.switchDiscord = a.appearance.discord
	a.switchDiscord = !a.switchDiscord
	if a.switchDiscord != a.appearance.discord {
		a.setDiscord(a.switchDiscord)
	}
	return a.appearance.discord
}

// TestTheSwitchCarriesItsValueAcrossFrames checks the read-back: the page seeds
// the field, a click flips it, and the page applies what changed.
func TestTheSwitchCarriesItsValueAcrossFrames(t *testing.T) {
	a := &app{art: newArtwork(), current: -1, page: pageAppearance}

	if !flipDiscord(a) {
		t.Error("the Discord switch did not turn the presence on")
	}
}

// TestTheSwitchTogglesBack checks that it is a switch and not a one-way door: a
// second flip turns it off again.
func TestTheSwitchTogglesBack(t *testing.T) {
	a := &app{art: newArtwork(), current: -1, page: pageAppearance}

	flipDiscord(a)
	if flipDiscord(a) {
		t.Error("the Discord switch did not turn the presence off again")
	}
}

// TestTheSwitchSurvivesTheNextFrame checks the bug the user saw: a switch that
// flips and then falls back. After a flip, drawing the page must not undo it.
func TestTheSwitchSurvivesTheNextFrame(t *testing.T) {
	a := &app{art: newArtwork(), current: -1, page: pageAppearance}
	flipDiscord(a)

	// Three frames of the real page: it seeds the switch field from the state
	// at the top and reads it back at the bottom, so a wrong seed shows up here
	// as a setting that flips itself off.
	tester := ui.NewTester(a.frame, 1280, 900)
	for range 3 {
		tester.Frame()
	}

	if !a.appearance.discord {
		t.Error("the Discord presence fell back off after the page drew again")
	}
}

// TestTheSwitchIsDrawnWithItsState checks that the page draws the switch from
// the real state, so the knob matches the setting.
func TestTheSwitchIsDrawnWithItsState(t *testing.T) {
	a := &app{art: newArtwork(), current: -1, page: pageAppearance}
	a.appearance.discord = true

	tester := ui.NewTester(a.frame, 1280, 900)
	tester.Frame()

	if !a.switchDiscord {
		t.Error("the page seeded the Discord switch off while it is on")
	}
}

// TestShuffleIsNotOnTheAppearancePage checks that shuffle has no setting: it is a
// thing the transport does, and it has a button there. A second switch for it was
// two controls for one setting.
func TestShuffleIsNotOnTheAppearancePage(t *testing.T) {
	a := &app{art: newArtwork(), current: -1, page: pageAppearance}
	tester := ui.NewTester(a.frame, 1280, 1200)
	tester.Frame()

	if tester.HasText("Play the queue in a random order.") {
		t.Errorf("the Appearance page still offers a Shuffle setting: %v", tester.Texts())
	}
	if !tester.HasText("Interface theme") {
		t.Error("the Appearance page lost its theme section too")
	}
}

// TestLastFMIsNotOnTheLookPage checks that the page no longer asks for a Last.fm
// key. Scrobbling still runs for an install that has one in its settings.
func TestLastFMIsNotOnTheLookPage(t *testing.T) {
	a := &app{art: newArtwork(), current: -1, page: pageAppearance, scrobble: newScrobbler()}
	tester := ui.NewTester(a.frame, 1280, 1400)
	tester.Frame()

	for _, text := range tester.Texts() {
		if text == "API key" || text == "Shared secret" || text == "Scrobbling" {
			t.Fatalf("the Look page still asks for Last.fm credentials: %v", tester.Texts())
		}
	}
}

// TestThereIsNoPaletteOrAccentPicker checks the design's rule: the window is
// paper and ink, so the Look page offers no colour to pick.
func TestThereIsNoPaletteOrAccentPicker(t *testing.T) {
	a := &app{art: newArtwork(), current: -1, page: pageAppearance}
	tester := ui.NewTester(a.frame, 1280, 1400)
	tester.Frame()

	for _, text := range tester.Texts() {
		if text == "Palette" || text == "Accent" || text == "Colour from artwork" {
			t.Fatalf("the Look page still offers a colour picker: %v", tester.Texts())
		}
	}
}
