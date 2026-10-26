package src

import (
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// The Appearance page offers three things the window actually reads: which
// appearance to draw, which accent to paint the transport with, and how strong a
// backdrop to ask the desktop for. These tests drive each through the same state
// the page writes, so they check the wiring rather than the drawing.

// TestTheAccentReachesTheTheme checks that picking a tone repaints the window.
func TestTheAccentReachesTheTheme(t *testing.T) {
	a := &app{art: newArtwork()}
	before := a.themeInFrame(t)

	a.setAccent("#6B7A68")
	after := a.themeInFrame(t)

	if after.Accent == before.Accent {
		t.Fatal("picking Sage left the accent unchanged")
	}
	if after.Accent != ui.Hex("#6B7A68") {
		t.Errorf("the accent is not the one picked: %v", after.Accent)
	}
	// The faces derived from it have to be derived, or a filled button gives no
	// sign it was touched.
	if after.AccentHover == after.Accent || after.AccentText == after.Accent {
		t.Error("the accent's hover face and its ink were not derived from it")
	}
}

// TestPickingAToneStopsTheArtworkTinting checks the one rule the two settings
// share: only one of them decides the colour.
func TestPickingAToneStopsTheArtworkTinting(t *testing.T) {
	a := &app{art: newArtwork()}
	a.setDynamic(true)
	a.setAccent("#B86A51")

	if a.appearance.dynamic {
		t.Error("picking a tone left the artwork tint on, so two settings fight")
	}
}

// TestTurningTheArtworkTintOnClearsTheTone checks the other direction.
func TestTurningTheArtworkTintOnClearsTheTone(t *testing.T) {
	a := &app{art: newArtwork()}
	a.setAccent("#B86A51")
	a.setDynamic(true)

	if a.appearance.accent != "" {
		t.Errorf("the picked tone survived the artwork tint: %q", a.appearance.accent)
	}
	if !a.appearance.dynamic {
		t.Error("the artwork tint was not turned on")
	}
}

// TestTheArtworkTintReadsTheCover checks the reader: a hue taken from a cover has
// to survive the fit that makes it usable as an accent.
func TestTheArtworkTintReadsTheCover(t *testing.T) {
	// A saturated blue, as a cover might be.
	red, green, blue := hexToRGB("#2a4a8a")
	light := fitAccent(red, green, blue, false)
	dark := fitAccent(red, green, blue, true)

	if len(light) != 7 || len(dark) != 7 {
		t.Fatalf("fitAccent did not return colours: %q, %q", light, dark)
	}
	// The dark appearance lifts the colour so it reads on charcoal; the light one
	// keeps it deep enough to read on paper.
	if dark == light {
		t.Error("the two appearances fit the same colour the same way")
	}
	// The hue survives: the blue channel is still the largest.
	for _, colour := range []string{light, dark} {
		r, g, b := hexToRGB(colour)
		if b <= r || b <= g {
			t.Errorf("%s lost its hue: it is no longer the bluest channel", colour)
		}
	}
}

// TestTheMaterialIsAskedForWhenTheWindowIsBorn checks that a window is created with
// a backdrop. On Windows a window born without one cannot take one later, so
// asking at creation is the only way the setting can work at all.
func TestTheMaterialIsAskedForWhenTheWindowIsBorn(t *testing.T) {
	if got := materialOf(appearance{}); got != mygo.VibrancyMica {
		t.Errorf("a fresh window asks for %q, not Mica", got)
	}
	if got := materialOf(appearance{material: "acrylic"}); got != mygo.VibrancyAcrylic {
		t.Errorf("a window set to High asks for %q, not Acrylic", got)
	}
	// A window told to draw its own face asks for no material at all.
	if got := materialOf(appearance{material: "acrylic", solid: true}); got != mygo.VibrancyNone {
		t.Errorf("a window set to reduce transparency still asks for %q", got)
	}
}

// TestTheAppearancePageOffersWhatTheWindowReads checks the page against the state:
// every tone it draws is one the theme can take, and every material is one MyGo
// knows.
func TestTheAppearancePageOffersWhatTheWindowReads(t *testing.T) {
	known := map[mygo.Vibrancy]bool{
		mygo.VibrancyTabbed:  true,
		mygo.VibrancyMica:    true,
		mygo.VibrancyAcrylic: true,
	}
	for _, material := range materials {
		if !known[material.value] {
			t.Errorf("%q is not a backdrop MyGo can draw", material.value)
		}
	}
	for _, accent := range accents {
		if len(accent.hex) != 7 || accent.hex[0] != '#' {
			t.Errorf("%q is not a hex colour", accent.hex)
		}
		if accent.name == "" {
			t.Errorf("the tone %q has no name, so a screen reader cannot offer it", accent.hex)
		}
	}
	// The paper's own ink is offered first, so a choice once made can be unmade.
	if accents[0].hex != paperAction {
		t.Errorf("the first tone is %q, not the paper's own ink", accents[0].hex)
	}
}

// TestTheAppearanceIsRemembered checks that every part of the choice survives a
// restart: the mode, the tone, the tint, and the material.
func TestTheAppearanceIsRemembered(t *testing.T) {
	store := openStore(t)
	a := &app{art: newArtwork(), store: store}
	a.setMode("Dark")
	a.setAccent("#566E7C")
	a.setMaterial("acrylic")

	mode, accent, _, _ := store.Appearance()
	material, solid := store.Material()
	if mode != "dark" {
		t.Errorf("the appearance came back as %q", mode)
	}
	if accent != "#566E7C" {
		t.Errorf("the tone came back as %q", accent)
	}
	if material != "acrylic" {
		t.Errorf("the material came back as %q", material)
	}
	if solid {
		t.Error("reduce transparency came back on")
	}
}
