package src

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// The rail can be hidden, and the button that brings it back is the only way to
// ask for it again.
//
// It was once drawn inside the rail's own row, so closing the rail closed the
// button too: the window kept a hidden sidebar and no way to unhide it, and the
// only fix was to quit. These tests pin the button to the window rather than to
// the rail.

// TestTheRailToggleSurvivesTheRailBeingClosed checks the bug: after the rail is
// closed, the window still offers a way to open it.
func TestTheRailToggleSurvivesTheRailBeingClosed(t *testing.T) {
	a := loaded(pageHome)
	tester := ui.NewTester(a.frame, 1280, 840)
	tester.Frame()

	// Close the rail the way the button does.
	a.railClosed = true
	// The rail animates its width shut, so the frame it is drawn in is the one
	// after the animation settles.
	tester.Advance(railMotion)
	tester.Frame()

	if !tester.HasText("Show or hide the sidebar") {
		t.Fatalf("the toggle is gone with the rail: %v", tester.Texts())
	}
	// And the rail really is closed, so the test is about the closed case: the
	// width it animated to. A spring leaves a sub-pixel residue, which is why
	// this is a tolerance and not a comparison with zero.
	if a.sidebarPixels > 1 {
		t.Errorf("the rail is still %.0f wide, so the toggle was never tested closed", a.sidebarPixels)
	}
}

// TestTheRailToggleIsClickableWhileClosed checks that it can actually be pressed:
// an element drawn off the window, or under the header, would be found by the
// test and still be unreachable.
func TestTheRailToggleIsClickableWhileClosed(t *testing.T) {
	a := loaded(pageHome)
	a.railClosed = true
	tester := ui.NewTester(a.frame, 1280, 840)
	tester.Advance(railMotion)
	tester.Frame()

	if err := tester.Click("Show or hide the sidebar"); err != nil {
		t.Fatalf("the toggle cannot be clicked while the rail is closed: %v", err)
	}
	tester.Frame()
	if a.railClosed {
		t.Error("clicking the toggle did not reopen the rail")
	}
}

// TestTheRailReopensWhenTheToggleIsPressed checks the whole round trip: close it,
// open it, close it again.
func TestTheRailReopensWhenTheToggleIsPressed(t *testing.T) {
	a := loaded(pageHome)
	tester := ui.NewTester(a.frame, 1280, 840)
	tester.Frame()

	// It starts open; each click flips it.
	for i, wantClosed := range []bool{true, false, true} {
		if err := tester.Click("Show or hide the sidebar"); err != nil {
			t.Fatalf("click %d: %v", i, err)
		}
		tester.Advance(railMotion)
		tester.Frame()
		if a.railClosed != wantClosed {
			t.Errorf("after click %d the rail is closed=%v, want %v",
				i, a.railClosed, wantClosed)
		}
	}
}
