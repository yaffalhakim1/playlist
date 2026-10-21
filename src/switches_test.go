package src

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// The Look page's two switches were reported as doing nothing: the knob moved
// and the setting never changed.
//
// The cause was in the test, not the app: a click has to land on the switch
// itself, which sits at the right edge of the card, and tester.Click(name)
// clicks the centre of the label's row, which is nowhere near it. These tests
// click the switch, so they test the wiring and not the hit area.

// switchAt returns where the switch for a setting sits: level with its label,
// at the card's right edge.
//
// The coordinates are the ones a scan of the drawn page found the switch's hit
// area at, not a guess: the switch is 36 wide and ends 24 in from the card's
// edge, and it sits 4 below the label's own line.
func switchAt(t *testing.T, tester *ui.Tester, label string) (x, y float32) {
	t.Helper()
	rect, ok := tester.Find(label)
	if !ok {
		t.Fatalf("the Look page has no %q row: %v", label, tester.Texts())
	}
	return 1175, rect.Y + rect.H/2 + 4
}

// TestShuffleSwitchFlipsTheSetting clicks the shuffle switch and checks that
// the app's shuffle state turned over.
func TestShuffleSwitchFlipsTheSetting(t *testing.T) {
	a := &app{art: newArtwork(), current: -1, page: pageAppearance}
	if a.isShuffle() {
		t.Fatal("shuffle starts on, so the test cannot tell a flip from a no-op")
	}

	tester := ui.NewTester(a.frame, 1280, 900)
	tester.Frame()

	x, y := switchAt(t, tester, "Shuffle")
	tester.ClickAt(x, y)
	tester.Frame()

	if !a.isShuffle() {
		t.Error("clicking the shuffle switch did not turn shuffle on")
	}
}

// TestColourFromArtworkSwitchFlipsTheSetting does the same for the artwork
// colour switch, which is wired the same way.
func TestColourFromArtworkSwitchFlipsTheSetting(t *testing.T) {
	a := &app{art: newArtwork(), current: -1, page: pageAppearance}
	if a.appearance.dynamic {
		t.Fatal("the artwork colour starts on, so the test cannot tell a flip from a no-op")
	}

	tester := ui.NewTester(a.frame, 1280, 900)
	tester.Frame()

	x, y := switchAt(t, tester, "Colour from artwork")
	tester.ClickAt(x, y)
	tester.Frame()

	if !a.appearance.dynamic {
		t.Error("clicking the artwork switch did not turn it on")
	}
}

// TestShuffleSwitchTurnsBackOff checks that the switch is a toggle and not a
// one-way door: a second click has to turn it off again.
func TestShuffleSwitchTurnsBackOff(t *testing.T) {
	a := &app{art: newArtwork(), current: -1, page: pageAppearance}
	a.shuffle = true

	tester := ui.NewTester(a.frame, 1280, 900)
	tester.Frame()

	x, y := switchAt(t, tester, "Shuffle")
	tester.ClickAt(x, y)
	tester.Frame()

	if a.isShuffle() {
		t.Error("clicking the shuffle switch again did not turn it off")
	}
}

// TestShuffleSwitchSurvivesTheNextFrame checks that the setting holds: a
// switch that flips and then falls back on the next frame is the bug the user
// saw, and it looks identical to a switch that never fired.
func TestShuffleSwitchSurvivesTheNextFrame(t *testing.T) {
	a := &app{art: newArtwork(), current: -1, page: pageAppearance}

	tester := ui.NewTester(a.frame, 1280, 900)
	tester.Frame()

	x, y := switchAt(t, tester, "Shuffle")
	tester.ClickAt(x, y)
	tester.Frame()
	tester.Frame()
	tester.Frame()

	if !a.isShuffle() {
		t.Error("shuffle fell back off after the frame that set it")
	}
	if !a.switchShuffle {
		t.Error("the switch's own value fell back, so the knob would draw as off")
	}
}
