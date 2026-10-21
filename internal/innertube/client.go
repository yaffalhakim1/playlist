// Package innertube reads YouTube Music metadata from Google's InnerTube API.
//
// InnerTube is the private API the YouTube Music web app talks to. It needs no
// API key and no account, and it is the only source that answers with the
// artist, the album and the artwork of every item of a playlist. The stream
// URLs it would hand out are signed and short-lived, so this package reads
// metadata only; internal/stream resolves the audio itself.
package innertube

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"youtube-playlist/internal/music"
)

const (
	// browseEndpoint is where InnerTube answers browse requests.
	browseEndpoint = "https://music.youtube.com/youtubei/v1/browse"
	// clientVersion is the version of the YouTube Music web app this client
	// claims to be. InnerTube answers differently for different versions.
	clientVersion = "1.20250219.01.00"
	// clientID is the numeric name of that client, in the header YouTube
	// Music sends.
	clientID = "67"
	// userAgent is the browser the web app pretends to be.
	userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36"
	// pagePause is the wait between two pages of a long playlist, so the
	// service is not hammered.
	pagePause = 250 * time.Millisecond
)

// Client reads YouTube Music metadata.
type Client struct {
	http *http.Client
}

// New returns a client that talks to InnerTube.
func New() *Client {
	return &Client{http: &http.Client{Timeout: 30 * time.Second}}
}

// Playlist reads the playlist a link or an id points at.
//
// The link may be a playlist page, a watch page with a list parameter, or the
// bare playlist id. Every page of the playlist is read, so a long playlist
// arrives whole.
func (c *Client) Playlist(ctx context.Context, ref string) (*music.Playlist, error) {
	id, err := ParsePlaylistID(ref)
	if err != nil {
		return nil, err
	}
	playlist := &music.Playlist{ID: id}
	continuation := ""
	for page := 0; ; page++ {
		root, err := c.browse(ctx, id, continuation)
		if err != nil {
			return nil, err
		}
		if page == 0 {
			playlist.Title, playlist.Thumbnail = parseHeader(root)
		}
		playlist.Tracks = append(playlist.Tracks, parseTracks(root)...)
		continuation = findContinuation(root)
		if continuation == "" {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pagePause):
		}
	}
	if len(playlist.Tracks) == 0 {
		return nil, errors.New("that playlist holds no tracks, or YouTube would not show it")
	}
	return playlist, nil
}

// browse asks InnerTube for one page of a playlist.
func (c *Client) browse(ctx context.Context, playlistID, continuation string) (any, error) {
	body := map[string]any{
		"context": map[string]any{
			"client": map[string]any{
				"hl":            "en",
				"gl":            "US",
				"clientName":    "WEB_REMIX",
				"clientVersion": clientVersion,
			},
		},
		"isAudioOnly": true,
	}
	if continuation == "" {
		body["browseId"] = "VL" + playlistID
	} else {
		body["continuation"] = continuation
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("innertube: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, browseEndpoint, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("innertube: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Origin", "https://music.youtube.com")
	req.Header.Set("Referer", "https://music.youtube.com/")
	req.Header.Set("X-YouTube-Client-Name", clientID)
	req.Header.Set("X-YouTube-Client-Version", clientVersion)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("innertube: %w", err)
	}
	defer resp.Body.Close()
	data, err := readAll(resp)
	if err != nil {
		return nil, fmt.Errorf("innertube: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("innertube: the service answered %s", resp.Status)
	}
	root, err := decodeOrdered(data)
	if err != nil {
		return nil, fmt.Errorf("innertube: %w", err)
	}
	return root, nil
}

// ParsePlaylistID reads the playlist id out of a link or an id.
//
// A watch link with a list parameter, a playlist page and a bare id all name
// the same playlist.
func ParsePlaylistID(ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", errors.New("no playlist link given")
	}
	if link, err := parseLink(ref); err == nil {
		if id := link.Query().Get("list"); id != "" {
			return id, nil
		}
		return "", errors.New("that link holds no playlist: it has no list= parameter")
	}
	if id, ok := strings.CutPrefix(ref, "VL"); ok && looksLikeID(id) {
		return id, nil
	}
	if looksLikeID(ref) {
		return ref, nil
	}
	return "", fmt.Errorf("%q is not a playlist link", ref)
}

// looksLikeID tells a playlist id from a video id, which is much shorter.
//
// The mixes YouTube generates have ids of twelve characters ("RD" and ten
// more), and a video id always has eleven.
func looksLikeID(id string) bool {
	if len(id) < 12 || len(id) > 64 {
		return false
	}
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
		default:
			return false
		}
	}
	return true
}

// mixPrefix is how YouTube Music names the mix it generates from a track.
//
// A mix is not a playlist that exists: YouTube builds it when it is asked for,
// from the track the listener is on. The id carries that track, which is why a
// mix needs a video to start from and a plain playlist does not.
const mixPrefix = "RDAMVM"

// MixID is the id of the mix YouTube generates from a video.
func MixID(videoID string) string {
	videoID = strings.TrimSpace(videoID)
	if videoID == "" {
		return ""
	}
	return mixPrefix + videoID
}

// MixLink is the link that plays the mix generated from a video.
//
// The watch id is part of it: YouTube will not serve a mix from the list id
// alone, because the list is built around the track being watched.
func MixLink(videoID string) string {
	videoID = strings.TrimSpace(videoID)
	if videoID == "" {
		return ""
	}
	return "https://www.youtube.com/watch?v=" + url.QueryEscape(videoID) +
		"&list=" + url.QueryEscape(MixID(videoID))
}

// MixTitle is what to call the mix built from a track, for the sidebar and the
// page it opens on.
func MixTitle(trackTitle string) string {
	trackTitle = strings.TrimSpace(trackTitle)
	if trackTitle == "" {
		return "Mix"
	}
	return "Mix - " + trackTitle
}
