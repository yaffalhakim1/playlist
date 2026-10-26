package src

import (
	"bytes"
	"image"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
)

// artwork keeps the pictures of the albums, so a track is drawn the same way
// every time and nothing is fetched twice.
type artwork struct {
	mu      sync.Mutex
	bitmaps map[string]*ui.Bitmap
	// colors is the hue each picture casts, for the artwork accent. It is read
	// from the same bytes as the bitmap, so a cover is decoded once.
	colors  map[string]string
	loading map[string]bool
	onReady func()
}

// artLimit bounds how much a picture may weigh.
const artLimit = 8 << 20

// newArtwork returns an empty cache.
func newArtwork() *artwork {
	return &artwork{
		bitmaps: map[string]*ui.Bitmap{},
		colors:  map[string]string{},
		loading: map[string]bool{},
	}
}

// get returns the picture of an address, or nil when it is not there yet.
func (a *artwork) get(url string) *ui.Bitmap {
	if url == "" {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.bitmaps[url]
}

// color returns the hue a picture casts, as a hex string, or "" when the picture
// is not there yet or has no colour worth taking.
func (a *artwork) color(url string) string {
	if url == "" {
		return ""
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.colors[url]
}

// want asks for a picture, and fetches it the first time.
func (a *artwork) want(url string) {
	if url == "" {
		return
	}
	a.mu.Lock()
	if a.bitmaps[url] != nil || a.loading[url] {
		a.mu.Unlock()
		return
	}
	a.loading[url] = true
	a.mu.Unlock()

	go func() {
		bitmap, colour, err := fetchArtwork(url)
		a.mu.Lock()
		delete(a.loading, url)
		if err == nil {
			a.bitmaps[url] = bitmap
			a.colors[url] = colour
		}
		ready := a.onReady
		a.mu.Unlock()
		if ready != nil {
			ready()
		}
	}()
}

// fetchArtwork reads a picture, turns it into something the interface can draw,
// and reads the hue of it.
func fetchArtwork(url string) (*ui.Bitmap, string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(biggerArtwork(url))
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", &netError{status: resp.Status}
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, artLimit))
	if err != nil {
		return nil, "", err
	}
	bitmap, err := ui.DecodeBitmap(data)
	if err != nil {
		return nil, "", err
	}
	colour := ""
	if img, _, err := image.Decode(bytes.NewReader(data)); err == nil {
		colour = dominantColor(img)
	}
	return bitmap, colour, nil
}

// netError reports an address that answered with an error.
type netError struct {
	status string
}

func (e *netError) Error() string { return "the picture answered " + e.status }

// biggerArtwork asks for the largest picture of a track.
//
// The addresses of YouTube come in several sizes; the one of an item in a list is
// small, and the big one is worth asking for when a track plays.
func biggerArtwork(url string) string {
	const prefix = "https://i.ytimg.com/vi/"
	if !strings.HasPrefix(url, prefix) {
		return url
	}
	rest := strings.TrimPrefix(url, prefix)
	id, tail, ok := strings.Cut(rest, "/")
	if !ok || id == "" {
		return url
	}
	name, _, _ := strings.Cut(tail, "?")
	switch name {
	case "hqdefault.jpg", "mqdefault.jpg", "default.jpg", "hq720.jpg":
		return prefix + id + "/maxresdefault.jpg"
	}
	return url
}
