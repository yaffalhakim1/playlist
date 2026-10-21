package src

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// The Look page offers Material 3 colour schemes. They change colour and
// nothing else: the app derives its hover, pressed, selection and focus faces
// from the accent, so a palette is a choice of surfaces, ink and one accent.

// TestEveryPaletteHasBothAppearances checks that no scheme is half done: a
// palette with a light side and no dark side would go wrong at dusk.
func TestEveryPaletteHasBothAppearances(t *testing.T) {
	for _, p := range palettes {
		if p.name == "" {
			t.Errorf("a palette has no name")
		}
		for _, s := range []struct {
			what   string
			scheme scheme
		}{
			{"light", p.light},
			{"dark", p.dark},
		} {
			if len(s.scheme.accent) != 7 {
				t.Errorf("%s (%s) has no accent", p.name, s.what)
			}
			if len(s.scheme.surface) != 7 {
				t.Errorf("%s (%s) has no surface", p.name, s.what)
			}
			if len(s.scheme.raised) != 7 {
				t.Errorf("%s (%s) has no raised surface", p.name, s.what)
			}
			if len(s.scheme.text) != 7 {
				t.Errorf("%s (%s) has no text colour", p.name, s.what)
			}
			if len(s.scheme.muted) != 7 {
				t.Errorf("%s (%s) has no muted colour", p.name, s.what)
			}
		}
	}
}

// TestPaletteIDsAreUnique checks that the id a palette is saved under names one
// palette, or the Look page would select the wrong one after a restart.
func TestPaletteIDsAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, p := range palettes {
		if seen[p.id] {
			t.Errorf("two palettes share the id %q", p.id)
		}
		seen[p.id] = true
	}
	// The empty id is Onevra's own, and it has to exist: it is what a fresh
	// install has, and what the store writes when nothing was chosen.
	if _, ok := paletteByID(""); !ok {
		t.Error("there is no default palette under the empty id")
	}
}

// TestAPaletteChangesTheTheme checks that choosing a palette reaches the theme
// the window installs.
func TestAPaletteChangesTheTheme(t *testing.T) {
	plain := &app{art: newArtwork()}
	before := plain.themeInFrame(t)

	violet := &app{art: newArtwork()}
	violet.appearance.palette = "violet"
	after := violet.themeInFrame(t)

	if before.Accent == after.Accent {
		t.Error("choosing Violet left the accent unchanged")
	}
	if before.Background == after.Background {
		t.Error("choosing Violet left the page colour unchanged")
	}
	if before.Text == after.Text {
		t.Error("choosing Violet left the ink unchanged")
	}
}

// TestAPaletteKeepsTheShape checks the promise that a palette changes colour
// and nothing else: spacing, radius and font size are the theme's own.
func TestAPaletteKeepsTheShape(t *testing.T) {
	plain := &app{art: newArtwork()}
	before := plain.themeInFrame(t)

	for _, p := range palettes {
		t.Run(p.name, func(t *testing.T) {
			a := &app{art: newArtwork()}
			a.appearance.palette = p.id
			after := a.themeInFrame(t)

			if after.Spacing != before.Spacing {
				t.Errorf("%s changed the spacing: %.1f, was %.1f",
					p.name, after.Spacing, before.Spacing)
			}
			if after.Radius != before.Radius {
				t.Errorf("%s changed the corner radius: %.1f, was %.1f",
					p.name, after.Radius, before.Radius)
			}
			if after.FontSize != before.FontSize {
				t.Errorf("%s changed the font size: %.1f, was %.1f",
					p.name, after.FontSize, before.FontSize)
			}
		})
	}
}

// TestAnAccentSitsOverAPalette checks that the two settings compose: the
// palette supplies the surfaces and the accent replaces the colour alone.
func TestAnAccentSitsOverAPalette(t *testing.T) {
	a := &app{art: newArtwork()}
	a.appearance.palette = "green"
	a.appearance.accent = "#6750a4"
	theme := a.themeInFrame(t)

	violet, _ := paletteByID("violet")
	if got, want := theme.Accent, ui.Hex(violet.light.accent); got != want {
		t.Errorf("the accent is not the one laid over the palette")
	}
	// The surface is still Green's, not Violet's.
	green, _ := paletteByID("green")
	if theme.Background != ui.Hex(green.light.surface) {
		t.Error("laying an accent over the palette replaced the palette's surfaces too")
	}
}

// TestPickingAPaletteClearsTheAccent checks that the two do not fight: a
// palette brings its own accent, so a colour laid over the last one goes.
func TestPickingAPaletteClearsTheAccent(t *testing.T) {
	a := &app{art: newArtwork()}
	a.appearance.accent = "#6750a4"
	a.appearance.dynamic = true

	a.setPalette("teal")

	if a.appearance.accent != "" {
		t.Errorf("the old accent survived: %q", a.appearance.accent)
	}
	if a.appearance.dynamic {
		t.Error("the artwork colour survived, so it would fight the palette")
	}
	if a.appearance.palette != "teal" {
		t.Errorf("the palette is %q, not teal", a.appearance.palette)
	}
}

// TestAPaletteIsRemembered checks that the choice survives a restart.
func TestAPaletteIsRemembered(t *testing.T) {
	store := openStore(t)
	a := &app{art: newArtwork(), store: store}
	a.setPalette("amber")

	_, _, _, palette := store.Appearance()
	if palette != "amber" {
		t.Errorf("the palette came back as %q, not amber", palette)
	}
}
