package src

import (
	"fmt"
	"image"
)

// dominantColor reads the colour of a picture: the colour that stands out
// most in it, as a hex string.
//
// The colours that are almost grey are left out: an album cover is often
// mostly black or white, and the colour of the app should not be that.
func dominantColor(img image.Image) string {
	bounds := img.Bounds()
	step := 1
	if bounds.Dx() > 96 {
		step = bounds.Dx() / 96
	}
	counts := map[uint32]int{}
	for y := bounds.Min.Y; y < bounds.Max.Y; y += step {
		for x := bounds.Min.X; x < bounds.Max.X; x += step {
			r, g, b, _ := img.At(x, y).RGBA()
			red, green, blue := int(r>>8), int(g>>8), int(b>>8)
			high, low := red, red
			if green > high {
				high = green
			}
			if blue > high {
				high = blue
			}
			if green < low {
				low = green
			}
			if blue < low {
				low = blue
			}
			if high-low < 24 {
				continue // almost grey
			}
			key := uint32(red>>4)<<8 | uint32(green>>4)<<4 | uint32(blue>>4)
			counts[key]++
		}
	}
	if len(counts) == 0 {
		return ""
	}
	best, bestScore := uint32(0), 0
	for key, count := range counts {
		red := int(key>>8&0xf) << 4
		green := int(key>>4&0xf) << 4
		blue := int(key&0xf) << 4
		score := count * (maxOf(red, green, blue) - minOf(red, green, blue))
		if score > bestScore {
			best, bestScore = key, score
		}
	}
	red := int(best>>8&0xf) << 4
	green := int(best>>4&0xf) << 4
	blue := int(best&0xf) << 4
	return fmt.Sprintf("#%02x%02x%02x", red, green, blue)
}

// fitAccent makes a colour fit to be the accent of the app: it keeps enough
// colour in it, and the brightness that the appearance asks for.
func fitAccent(red, green, blue int, dark bool) string {
	target := 0.42
	if dark {
		target = 0.76
	}
	lum := (0.2126*float64(red) + 0.7152*float64(green) + 0.0722*float64(blue)) / 255
	if lum > 0.01 {
		factor := target / lum
		red = clamp255(int(float64(red) * factor))
		green = clamp255(int(float64(green) * factor))
		blue = clamp255(int(float64(blue) * factor))
	}
	return fmt.Sprintf("#%02x%02x%02x", red, green, blue)
}

// clamp255 keeps a colour inside the range of a colour.
func clamp255(v int) int {
	switch {
	case v < 0:
		return 0
	case v > 255:
		return 255
	default:
		return v
	}
}

// markOn returns the ink that reads on a filled colour: white on a dark fill
// and near-black on a light one, by the fill's relative luminance, so a mark
// never disappears into the swatch it sits on.
func markOn(fill string) string {
	red, green, blue := hexToRGB(fill)
	lum := (0.2126*float64(red) + 0.7152*float64(green) + 0.0722*float64(blue)) / 255
	if lum > 0.55 {
		return "#1a1a18"
	}
	return "#ffffff"
}

// maxOf returns the largest of three numbers.
func maxOf(a, b, c int) int {
	if b > a {
		a = b
	}
	if c > a {
		a = c
	}
	return a
}

// minOf returns the smallest of three numbers.
func minOf(a, b, c int) int {
	if b < a {
		a = b
	}
	if c < a {
		a = c
	}
	return a
}
