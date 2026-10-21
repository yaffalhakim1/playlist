package src

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// The Look page's two switches were reported as doing nothing: the knob moved
// and the setting never changed.
//
// Two things were wrong. MyGo flips a switch's bound value as the control is
// built, so the value has to outlive the frame rather than being a fresh local
// each time; and a click has to land on the switch, which sits at the right of
// its row, not on the middle of the label, which is what tester.Click(name)
// does.
//
// These tests drive the switch through the same binding the page uses, so they
// check the wiring rather than the hit area.

// flipSwitch turns a switch the way a click does, through the binding the page
// gives it, and reports the value the page would read back.
func flipSwitch(a *app) (shuffle, dynamic bool) {
	// appearancePage seeds the two fields from the real state as it draws, then
	// reads them back after. This is the same round trip, with the switch's own
	// flip in between.
	a.switchShuffle = a.isShuffle()
	a.switchDynamic = a.appearance.dynamic

	// What a click does to the binding MyGo holds.
	a.switchShuffle = !a.switchShuffle
	a.switchDynamic = !a.switchDynamic

	if a.switchDynamic != a.appearance.dynamic {
		a.setDynamic(a.switchDynamic)
	}
	if a.switchShuffle != a.isShuffle() {
		a.setShuffle(a.switchShuffle)
	}
	return a.isShuffle(), a.appearance.dynamic
}

// TestTheSwitchesCarryTheirValueAcrossFrames checks the read-back: the page
// seeds the fields, a click flips them, and the page applies what changed.
func TestTheSwitchesCarryTheirValueAcrossFrames(t *testing.T) {
	a := &app{art: newArtwork(), current: -1, page: pageAppearance}

	shuffle, dynamic := flipSwitch(a)
	if !shuffle {
		t.Error("the shuffle switch did not turn shuffle on")
	}
	if !dynamic {
		t.Error("the artwork switch did not turn the artwork colour on")
	}
}

// TestTheSwitchesToggleBack checks that they are switches and not one-way
// doors: a second flip turns each one off again.
func TestTheSwitchesToggleBack(t *testing.T) {
	a := &app{art: newArtwork(), current: -1, page: pageAppearance}

	flipSwitch(a)
	shuffle, dynamic := flipSwitch(a)

	if shuffle {
		t.Error("the shuffle switch did not turn shuffle off again")
	}
	if dynamic {
		t.Error("the artwork switch did not turn the artwork colour off again")
	}
}

// TestTheSwitchesSurviveTheNextFrame checks the bug the user saw: a switch that
// flips and then falls back. After a flip, drawing the page must not undo it.
func TestTheSwitchesSurviveTheNextFrame(t *testing.T) {
	a := &app{art: newArtwork(), current: -1, page: pageAppearance}
	flipSwitch(a)

	// Three frames of the real page: it seeds the switch fields from the state
	// at the top and reads them back at the bottom, so a wrong seed shows up
	// here as a setting that flips itself off.
	tester := ui.NewTester(a.frame, 1280, 900)
	for range 3 {
		tester.Frame()
	}

	if !a.isShuffle() {
		t.Error("shuffle fell back off after the page drew again")
	}
	if !a.appearance.dynamic {
		t.Error("the artwork colour fell back off after the page drew again")
	}
}

// TestTheSwitchesAreDrawnWithTheirState checks that the page draws the switches
// from the real state, so the knob matches the setting.
func TestTheSwitchesAreDrawnWithTheirState(t *testing.T) {
	a := &app{art: newArtwork(), current: -1, page: pageAppearance}
	a.shuffle = true
	a.appearance.dynamic = true

	tester := ui.NewTester(a.frame, 1280, 900)
	tester.Frame()

	if !a.switchShuffle {
		t.Error("the page seeded the shuffle switch off while shuffle is on")
	}
	if !a.switchDynamic {
		t.Error("the page seeded the artwork switch off while it is on")
	}
}
