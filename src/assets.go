package src

import (
	_ "embed"
	"log"

	"github.com/egoist/mygo/ui"
)

// Inter is the app's typeface: a humanist sans drawn for screens, with a
// tall x-height that holds up at 12 DIPs and tabular figures for numbers
// that change while the music plays.
//
// Three weights are all the app uses: regular for text, medium for labels
// and controls, semibold for names. The font is embedded so the app looks
// the same on a machine that never installed it.

//go:embed assets/fonts/Inter-Regular.ttf
var interRegular []byte

//go:embed assets/fonts/Inter-Medium.ttf
var interMedium []byte

//go:embed assets/fonts/Inter-SemiBold.ttf
var interSemiBold []byte

// familySans is the name the theme asks for.
const familySans = "Inter"

// interReady tells whether the desktop took the font. When it did not, the
// theme keeps the system's own instead of a name nothing answers to.
var interReady bool

func init() { registerInter() }

// registerInter adds the three weights under one family, so a weight of
// the theme's picks the face that carries it.
func registerInter() {
	for _, font := range [][]byte{interRegular, interMedium, interSemiBold} {
		if err := ui.RegisterFont(font, familySans); err != nil {
			log.Printf("font: the app keeps the system's own: %v", err)
			return
		}
	}
	interReady = true
}
