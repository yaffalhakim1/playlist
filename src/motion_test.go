package src

import (
	"image"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// The app moves a handful of things, and each of them has to be cheap enough to
// hold a frame on a fast display.
//
// The rule the tests below protect: **a thing that only needs to move is moved
// with a pose, not by rebuilding the view.** A pose runs in frames that only
// paint, so its cost is a transform and not a layout pass; a view that rebuilds
// per frame costs a layout per frame, and on a 240 Hz display that is the
// difference between smooth and stuttering.
//
// What does rebuild — a page arriving, a row recolouring — is a short spring
// that settles and can be interrupted, and it stills entirely when the desktop
// asks for less motion.

// TestMotionIsShort checks the durations: nothing the app moves runs long enough
// to feel like a wait.
func TestMotionIsShort(t *testing.T) {
	cases := []struct {
		name string
		a    ui.Animation
		max  time.Duration
	}{
		{"motionPress", motionPress, 250 * time.Millisecond},
		{"motionRise", motionRise, 300 * time.Millisecond},
		{"motionColor", motionColor, 200 * time.Millisecond},
		{"motionFade", motionFade, 300 * time.Millisecond},
	}
	for _, tc := range cases {
		// A spring carries its length in its own options; a curve in Duration.
		d := tc.a.Duration
		if d == 0 {
			d = tc.a.Spring.Duration
		}
		if d <= 0 || d > tc.max {
			t.Errorf("%s runs for %v: a control has to answer at once, so %v is the most",
				tc.name, d, tc.max)
		}
	}
}

// TestMotionStillsUnderReduceMotion checks the promise the desktop's setting
// asks for: a pose goes to its place at once rather than travelling there.
//
// The element is drawn with the setting off and on, and the two pictures must
// differ while the motion is on its way — and agree once it would have
// finished.
func TestMotionStillsUnderReduceMotion(t *testing.T) {
	view := func() (func(c *ui.Context), *bool) {
		lifted := false
		return func(c *ui.Context) {
			t := c.Theme()
			ui.Column(c).Fill().Padding(t.Space(4)).Background(t.Background).Children(func() {
				card := ui.Box(c).FillWidth().Height(t.Space(20)).
					Radius(radiusControl(t)).Background(t.Accent)
				// The pose is set on every frame, as the app sets it: a pose
				// starts where it is first given.
				pose := ui.Pose{}
				if lifted {
					pose = ui.Pose{Y: -60}
				}
				card.AnimatePose(pose, motionRise)
			})
		}, &lifted
	}

	view1, lift1 := view()
	moving := ui.NewTester(view1, 200, 160)
	moving.Frame()
	*lift1 = true
	moving.Advance(20 * time.Millisecond)
	moving.Frame()
	mid := moving.Image()

	view2, lift2 := view()
	still := ui.NewTester(view2, 200, 160)
	still.SetPreferences(ui.Preferences{ReduceMotion: true})
	still.Frame()
	*lift2 = true
	still.Advance(20 * time.Millisecond)
	still.Frame()
	atOnce := still.Image()

	if sameImage(mid, atOnce) {
		t.Error("the pose is drawn the same with Reduce Motion on: it did not still")
	}

	// Once the motion would have finished, both are at the same place.
	moving.Advance(time.Second)
	moving.Frame()
	still.Advance(time.Second)
	still.Frame()
	if !sameImage(moving.Image(), still.Image()) {
		t.Error("the element did not come to rest where the still one already was")
	}
}

// TestARowRecoloursWithoutMoving checks that the face of a row is a colour
// animation: a list of a thousand tracks must not relayout itself because the
// pointer moved one row down.
func TestARowRecoloursWithoutMoving(t *testing.T) {
	a := loaded(pageQueue)
	tester := ui.NewTester(a.frame, 1280, 840)
	tester.Frame()

	row, ok := tester.Find("Roar")
	if !ok {
		t.Fatalf("the list shows no row to move over: %v", tester.Texts())
	}
	before := tester.Image()

	tester.Move(row.X+row.W/2, row.Y+row.H/2)
	// A colour animates over a moment, so the frame that shows it is the one
	// after the pointer moved, not the one it moved in.
	tester.Advance(200 * time.Millisecond)
	tester.Frame()
	hovered := tester.Image()

	if sameImage(before, hovered) {
		t.Error("the row under the pointer did not change face")
	}
	// The row's box is the same: a colour change must not move the text.
	after, ok := tester.Find("Roar")
	if !ok {
		t.Fatal("the row is gone after the pointer rested on it")
	}
	if after.Y != row.Y || after.H != row.H {
		t.Errorf("the row moved from y=%.1f h=%.1f to y=%.1f h=%.1f: a face change relaid it out",
			row.Y, row.H, after.Y, after.H)
	}
}

// TestTheBarSaysWhatThePlaybackIsDoing checks the three states the bar has to
// tell apart. They are the states a user reads the app by: a track being found,
// a track that plays, and a track that failed.
func TestTheBarSaysWhatThePlaybackIsDoing(t *testing.T) {
	t.Run("finding", func(t *testing.T) {
		a := loaded(pageHome)
		a.buffering = true
		tester := ui.NewTester(a.frame, 1280, 840)
		tester.Frame()
		if !tester.HasText("Finding the track…") {
			t.Errorf("the bar does not say it is finding the track: %v", tester.Texts())
		}
	})
	t.Run("failed", func(t *testing.T) {
		a := loaded(pageHome)
		a.playing = false
		a.status = "yt-dlp found no audio stream"
		tester := ui.NewTester(a.frame, 1280, 840)
		tester.Frame()
		if !tester.HasText("That track will not play") {
			t.Errorf("the bar does not say the track failed: %v", tester.Texts())
		}
		if !tester.HasText("Try again") {
			t.Errorf("the bar offers no way to try the track again: %v", tester.Texts())
		}
	})
	t.Run("playing", func(t *testing.T) {
		a := loaded(pageHome)
		tester := ui.NewTester(a.frame, 1280, 840)
		tester.Frame()
		if tester.HasText("Finding the track…") || tester.HasText("That track will not play") {
			t.Error("a track that plays is drawn as one that is loading or broken")
		}
	})
}

// sameImage tells whether two frames are the same picture.
func sameImage(a, b *image.RGBA) bool {
	if a == nil || b == nil || len(a.Pix) != len(b.Pix) {
		return false
	}
	for i := range a.Pix {
		if a.Pix[i] != b.Pix[i] {
			return false
		}
	}
	return true
}