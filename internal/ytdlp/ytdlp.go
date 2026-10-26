// Package ytdlp reads YouTube with yt-dlp.
//
// yt-dlp is the program that knows how YouTube's streams are signed and how
// its challenges are solved. The app asks it for two things: the audio of one
// track, and the list of a playlist that InnerTube cannot read.
package ytdlp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"youtube-playlist/internal/music"
	"youtube-playlist/internal/proc"
)

// userAgent is the browser the web app pretends to be.
const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
	"(KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36"

// resolveTimeout bounds one run of yt-dlp.
const resolveTimeout = 90 * time.Second

// playlistTimeout bounds the reading of a whole playlist.
const playlistTimeout = 3 * time.Minute

// The cache of resolved URLs. A URL carries an expiry of its own, so it is
// only kept while it lasts.
const (
	urlTTL   = 30 * time.Minute
	urlLimit = 8
)

// ErrToolMissing reports that yt-dlp is not installed.
type ErrToolMissing struct {
	Err error
}

func (e *ErrToolMissing) Error() string {
	return fmt.Sprintf("yt-dlp is not installed, so playback cannot work: %v", e.Err)
}

func (e *ErrToolMissing) Unwrap() error { return e.Err }

// LookTool finds yt-dlp, and says so when it is missing.
func LookTool() (string, error) {
	path, err := exec.LookPath("yt-dlp")
	if err != nil {
		return "", &ErrToolMissing{Err: err}
	}
	return path, nil
}

// Resolver turns a video into the direct URL of its audio.
type Resolver struct {
	mu    sync.Mutex
	cache map[string]cachedURL
	order []string
}

type cachedURL struct {
	url   string
	until time.Time
}

// NewResolver returns a resolver with an empty cache.
func NewResolver() *Resolver {
	return &Resolver{cache: map[string]cachedURL{}}
}

// Resolve returns the direct URL of the audio of a video.
//
// The URL is signed by YouTube and stops working after a few hours, so it is
// kept in a small cache and asked for again when it goes stale.
func (r *Resolver) Resolve(ctx context.Context, videoID string) (string, error) {
	if url := r.cached(videoID); url != "" {
		return url, nil
	}
	url, err := r.resolve(ctx, videoID)
	if err != nil {
		return "", err
	}
	r.remember(videoID, url)
	return url, nil
}

// resolve runs yt-dlp, first with the client that needs no JavaScript, and
// with one when the service insists on it.
func (r *Resolver) resolve(ctx context.Context, videoID string) (string, error) {
	if _, err := LookTool(); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, resolveTimeout)
	defer cancel()

	watch := "https://music.youtube.com/watch?v=" + videoID
	attempts := [][]string{
		{"-f", "bestaudio", "-g", "--no-playlist", "--no-warnings",
			"--extractor-args", "youtube:skip=hls,dash;player_client=visionos", watch},
	}
	if runtime := jsRuntime(); runtime != "" {
		attempts = append(attempts, []string{"-f", "bestaudio", "-g", "--no-playlist", "--no-warnings",
			"--extractor-args", "youtube:skip=hls,dash", "--js-runtimes", runtime, watch})
	}

	var lastErr error
	for _, args := range attempts {
		out, err := run(ctx, args...)
		if err != nil {
			lastErr = err
			continue
		}
		url, _, _ := strings.Cut(out, "\n")
		url = strings.TrimSpace(url)
		if url == "" {
			lastErr = errors.New("yt-dlp found no stream")
			continue
		}
		return url, nil
	}
	return "", fmt.Errorf("yt-dlp found no audio stream for %s: %w", videoID, lastErr)
}

// run runs yt-dlp and returns what it printed.
func run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	// Without this, Windows opens a console window for yt-dlp, which the user
	// sees flash beside the player on every track.
	proc.Hide(cmd)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) && strings.TrimSpace(stderr.String()) != "" {
			return "", fmt.Errorf("yt-dlp: %s", strings.TrimSpace(stderr.String()))
		}
		return "", fmt.Errorf("yt-dlp: %w", err)
	}
	return string(out), nil
}

// jsRuntime finds a JavaScript runtime for yt-dlp, which needs one for some
// challenges.
func jsRuntime() string {
	for _, name := range []string{"node", "bun"} {
		if path, err := exec.LookPath(name); err == nil {
			return name + ":" + path
		}
	}
	return ""
}

// cached returns a URL that is still fresh.
func (r *Resolver) cached(videoID string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	entry, ok := r.cache[videoID]
	if !ok || time.Now().After(entry.until) {
		return ""
	}
	return entry.url
}

// Forget drops a cached URL, so the next resolve asks yt-dlp again.
//
// A stream that broke off mid-track broke with the link it was given, and that
// link is usually the reason: it expired, or the server stopped serving it. A
// recovery that took the cached answer would fail the same way, so the caller
// forgets it first.
func (r *Resolver) Forget(videoID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.cache[videoID]; !ok {
		return
	}
	delete(r.cache, videoID)
	for i, id := range r.order {
		if id == videoID {
			r.order = append(r.order[:i], r.order[i+1:]...)
			break
		}
	}
}

// remember keeps a URL until it goes stale, and forgets the oldest when there
// are too many.
func (r *Resolver) remember(videoID, url string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.cache[videoID]; !ok {
		r.order = append(r.order, videoID)
	}
	r.cache[videoID] = cachedURL{url: url, until: time.Now().Add(urlTTL)}
	for len(r.order) > urlLimit {
		oldest := r.order[0]
		r.order = r.order[1:]
		delete(r.cache, oldest)
	}
}

// flatPlaylist is what yt-dlp answers for a playlist read without opening
// every track.
type flatPlaylist struct {
	Title      string `json:"title"`
	Thumbnails []struct {
		URL    string `json:"url"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	} `json:"thumbnails"`
	Entries []flatEntry `json:"entries"`
}

// flatEntry is one track of a playlist.
type flatEntry struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	URL        string  `json:"url"`
	Duration   float64 `json:"duration"`
	Channel    string  `json:"channel"`
	Uploader   string  `json:"uploader"`
	Thumbnails []struct {
		URL    string `json:"url"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	} `json:"thumbnails"`
}

// Playlist reads a playlist with yt-dlp.
//
// It is the way to read the playlists InnerTube cannot: the mixes YouTube
// generates ("RD...") are only viewable with the token that comes with the
// link. The tracks have no artist and no album here, only their title.
func Playlist(ctx context.Context, ref string) (*music.Playlist, error) {
	if _, err := LookTool(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, playlistTimeout)
	defer cancel()

	out, err := run(ctx, "--flat-playlist", "-J", "--no-warnings", ref)
	if err != nil {
		return nil, err
	}
	var flat flatPlaylist
	if err := json.Unmarshal([]byte(out), &flat); err != nil {
		return nil, fmt.Errorf("yt-dlp: %w", err)
	}
	playlist := &music.Playlist{Title: flat.Title, Thumbnail: bestThumb(flat.Thumbnails)}
	for _, entry := range flat.Entries {
		if entry.ID == "" {
			continue
		}
		title, artist := splitTitle(entry.Title)
		// yt-dlp knows the channel that published a track, which for a mix is
		// the only artist there is. It is used when the title does not name one,
		// because "Artist - Song" is more precise than the channel a label or a
		// topic page publishes under.
		if artist == "" {
			artist = artistFromChannel(entry.Channel, entry.Uploader, title)
		}
		playlist.Tracks = append(playlist.Tracks, music.Track{
			ID:        entry.ID,
			Title:     title,
			Artist:    artist,
			Duration:  time.Duration(entry.Duration * float64(time.Second)),
			Thumbnail: bestThumb(entry.Thumbnails),
		})
	}
	if len(playlist.Tracks) == 0 {
		return nil, errors.New("yt-dlp found no tracks in that playlist")
	}
	if playlist.Thumbnail == "" {
		// A mix has no artwork of its own: the cover of its first track
		// stands in for it.
		playlist.Thumbnail = playlist.Tracks[0].Thumbnail
	}
	return playlist, nil
}

// splitTitle reads the artist out of the title of a track.
//
// yt-dlp knows no artist of its own, but the titles of the tracks on YouTube
// usually read "Artist - Song", and that is where the artist is.
func splitTitle(title string) (name, artist string) {
	before, after, ok := strings.Cut(title, " - ")
	if !ok || before == "" || len(before) > 60 {
		return title, ""
	}
	return strings.TrimSpace(after), strings.TrimSpace(before)
}

// artistFromChannel returns the artist a track's channel names, for the tracks
// whose title does not.
//
// The channel is not always the performer: a label, a topic page and a lyric
// channel all publish other people's music. So it is only used when it looks
// like a performer — a name, not "Various Artists", not the "- Topic" channel
// YouTube generates, and not a name that is really a label. What it buys is a
// mix's tracks having an artist at all, which is what the queue shows under the
// title and what Last.fm needs to record a play.
func artistFromChannel(channel, uploader, title string) string {
	name := strings.TrimSpace(channel)
	if name == "" {
		name = strings.TrimSpace(uploader)
	}
	name = strings.TrimSuffix(name, " - Topic")
	name = strings.TrimSpace(name)
	if name == "" || name == title {
		return ""
	}
	if strings.Contains(strings.ToLower(name), "vevo") {
		return name
	}
	for _, generic := range []string{"various artists", "youtube", "topic", "records", "music"} {
		if strings.EqualFold(name, generic) {
			return ""
		}
	}
	return name
}

// bestThumb returns the largest of a list of thumbnails.
func bestThumb(thumbs []struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}) string {
	best, area := "", 0
	for _, thumb := range thumbs {
		if thumb.URL == "" {
			continue
		}
		if size := thumb.Width * thumb.Height; size > area {
			best, area = thumb.URL, size
		}
	}
	return best
}

// DurationFromURL reads the length of a track out of its signed URL, which
// carries it.
func DurationFromURL(raw string) time.Duration {
	query := raw
	if i := strings.IndexByte(query, '?'); i >= 0 {
		query = query[i+1:]
	} else {
		return 0
	}
	for _, part := range strings.Split(query, "&") {
		key, value, ok := strings.Cut(part, "=")
		if !ok || key != "dur" {
			continue
		}
		seconds, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return 0
		}
		return time.Duration(seconds * float64(time.Second))
	}
	return 0
}
