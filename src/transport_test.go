package src

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// The transport's two toggles — shuffle and repeat — were both broken in ways that
// looked like nothing at all.
//
// Repeat drew a second button: the glyph was chosen by reassigning the element
// after it had already been made a child of the row, so cycling to "repeat one"
// added a button beside the first rather than changing the one there. Clicking it
// again added a third.
//
// Shuffle never looked on: its colour was set on the button, and the mark inside it
// is a child with a colour of its own, so the colour never arrived.
//
// These tests count the buttons and read the colour of the mark, which is what a
// person sees.

// transportTexts returns what the transport of the bar holds, for a page that has
// one, so a count is about the bar and not about the rail or the page.
func transportTexts(t *testing.T, a *app) []string {
	t.Helper()
	tester := ui.NewTester(a.frame, 1280, 840)
	tester.Frame()
	return tester.Texts()
}

// count returns how many times a string appears.
func count(list []string, want string) int {
	n := 0
	for _, item := range list {
		if item == want {
			n++
		}
	}
	return n
}

// TestRepeatIsOneButtonInThreeStates checks the bug: cycling repeat changes the one
// button rather than adding another.
func TestRepeatIsOneButtonInThreeStates(t *testing.T) {
	a := loaded(pageQueue)

	// Off: one repeat button, named for what it would do.
	texts := transportTexts(t, a)
	if got := count(texts, "Repeat") + count(texts, "Repeat the queue") + count(texts, "Repeat this track"); got != 1 {
		t.Fatalf("the transport draws %d repeat buttons at rest, not 1: %v", got, texts)
	}

	// The queue again.
	a.cycleRepeat()
	texts = transportTexts(t, a)
	if got := count(texts, "Repeat the queue"); got != 1 {
		t.Errorf("with the queue repeating, the transport draws %d repeat buttons, not 1: %v", got, texts)
	}
	if count(texts, "Repeat this track") != 0 {
		t.Errorf("the queue-repeating button is joined by the one-track button: %v", texts)
	}

	// The one track again.
	a.cycleRepeat()
	texts = transportTexts(t, a)
	if got := count(texts, "Repeat this track"); got != 1 {
		t.Errorf("with one track repeating, the transport draws %d repeat buttons, not 1: %v", got, texts)
	}
	if count(texts, "Repeat the queue") != 0 {
		t.Errorf("the one-track button is joined by the queue-repeating one: %v", texts)
	}

	// And back to off.
	a.cycleRepeat()
	texts = transportTexts(t, a)
	if got := count(texts, "Repeat"); got != 1 {
		t.Errorf("back at off, the transport draws %d repeat buttons, not 1: %v", got, texts)
	}
}

// TestRepeatNeverGrowsTheTransport checks the shape of the bar: three cycles of
// repeat must not make the transport any wider than it started.
func TestRepeatNeverGrowsTheTransport(t *testing.T) {
	a := loaded(pageQueue)
	tester := ui.NewTester(a.frame, 1280, 840)
	tester.Frame()

	before, ok := tester.Find("Shuffle")
	if !ok {
		t.Fatal("the transport has no shuffle button")
	}
	for range 3 {
		a.cycleRepeat()
		tester.Frame()
	}
	after, ok := tester.Find("Shuffle")
	if !ok {
		t.Fatal("the shuffle button is gone after cycling repeat")
	}
	// The shuffle button is the first of the transport, so its position is where
	// the row starts: a row that grew would push it.
	if after.X != before.X {
		t.Errorf("the transport moved from x=%.0f to x=%.0f: it grew", before.X, after.X)
	}
}

// TestShuffleLightsUpWhenOn checks the other bug: the mark changes colour with the
// setting, so the toggle says it is on.
func TestShuffleLightsUpWhenOn(t *testing.T) {
	// A frame with the shuffle off, and one with it on: the two pictures have to
	// differ in the mark's own pixels.
	off := shuffleMarkPixels(t, false)
	on := shuffleMarkPixels(t, true)

	if samePixels(off, on) {
		t.Error("the shuffle mark is drawn the same on and off: the toggle says nothing")
	}
}

// shuffleMarkPixels returns the pixels of the shuffle button.
func shuffleMarkPixels(t *testing.T, shuffle bool) []uint8 {
	t.Helper()
	a := loaded(pageQueue)
	a.shuffle = shuffle
	tester := ui.NewTester(a.frame, 1280, 840)
	tester.Frame()
	rect, ok := tester.Find("Shuffle")
	if !ok {
		t.Fatal("the transport has no shuffle button")
	}
	img := tester.Image()
	if img == nil {
		t.Fatal("nothing drew")
	}
	var out []uint8
	for y := int(rect.Y); y < int(rect.Y+rect.H); y++ {
		for x := int(rect.X); x < int(rect.X+rect.W); x++ {
			i := img.PixOffset(x, y)
			out = append(out, img.Pix[i:i+4]...)
		}
	}
	return out
}

// samePixels tells whether two pixel runs are identical.
func samePixels(a, b []uint8) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestTheTogglesAreNamedForWhatTheyDo checks the accessible names, which are what a
// screen reader reads and what a test finds the button by.
func TestTheTogglesAreNamedForWhatTheyDo(t *testing.T) {
	a := loaded(pageQueue)
	a.setRepeat(2)

	texts := transportTexts(t, a)
	if count(texts, "Repeat this track") != 1 {
		t.Errorf("a repeating track is not named for it: %v", texts)
	}
}
