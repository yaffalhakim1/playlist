package src

import (
	_ "embed"
	"log"

	"github.com/egoist/mygo/ui"
)

// Plus Jakarta Sans is the app's typeface: a geometric sans with a tall
// x-height, drawn for interfaces, and the face the design was made in.
//
// Four weights are all the app uses: regular for text, medium for labels and
// controls, semibold for names, bold for the one title a page leads with. The
// font is embedded so the app looks the same on a machine that never installed
// it.

//go:embed assets/fonts/PlusJakartaSans-Regular.ttf
var jakartaRegular []byte

//go:embed assets/fonts/PlusJakartaSans-Medium.ttf
var jakartaMedium []byte

//go:embed assets/fonts/PlusJakartaSans-SemiBold.ttf
var jakartaSemiBold []byte

//go:embed assets/fonts/PlusJakartaSans-Bold.ttf
var jakartaBold []byte

// familySans is the name the theme asks for.
const familySans = "Plus Jakarta Sans"

// jakartaReady tells whether the desktop took the font. When it did not, the
// theme keeps the system's own instead of a name nothing answers to.
var jakartaReady bool

func init() { registerFonts() }

// registerFonts adds the four weights under one family, so a weight of the
// theme's picks the face that carries it.
func registerFonts() {
	for _, font := range [][]byte{jakartaRegular, jakartaMedium, jakartaSemiBold, jakartaBold} {
		if err := ui.RegisterFont(font, familySans); err != nil {
			log.Printf("font: the app keeps the system's own: %v", err)
			return
		}
	}
	jakartaReady = true
}

// appIcon is the app's mark, the same drawing mygo build embeds in the
// executable.
//
// It is embedded here as well so the window can set it while it runs: the
// taskbar reads the icon of the window, and the one in the executable's
// resources is only read when nothing else has claimed it. Setting it makes the
// taskbar right even when the app was started in a way that skipped the
// resources, as `go run` and the dev server do.
//
//go:embed assets/app-icon.png
var appIcon []byte
