package innertube

import (
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"youtube-playlist/internal/music"
)

// readAll reads a response body, with a limit so a broken service cannot
// fill the memory.
func readAll(resp *http.Response) ([]byte, error) {
	return io.ReadAll(io.LimitReader(resp.Body, 32<<20))
}

// parseLink reads a link, and reports whether the text is one at all.
func parseLink(ref string) (*url.URL, error) {
	if !strings.Contains(ref, "://") && !strings.Contains(ref, "?") {
		return nil, io.EOF
	}
	link, err := url.Parse(ref)
	if err != nil {
		return nil, err
	}
	if link.Host == "" {
		return nil, io.EOF
	}
	return link, nil
}

// parseHeader reads the name and the artwork of the playlist.
func parseHeader(root any) (title, thumbnail string) {
	walk(root, func(o *omap) bool {
		if o.object("musicResponsiveHeaderRenderer") != nil {
			title, thumbnail = headerOf(o.object("musicResponsiveHeaderRenderer"))
			return false
		}
		if o.object("musicDetailHeaderRenderer") != nil {
			title, thumbnail = headerOf(o.object("musicDetailHeaderRenderer"))
			return false
		}
		if o.object("playlistHeaderRenderer") != nil {
			title, thumbnail = headerOf(o.object("playlistHeaderRenderer"))
			return false
		}
		return true
	})
	return title, thumbnail
}

// headerOf reads the title and the artwork of a header renderer.
func headerOf(h *omap) (title, thumbnail string) {
	if h == nil {
		return "", ""
	}
	title = textOf(h.object("title"))
	thumb := h.object("thumbnail")
	if thumb == nil {
		return title, ""
	}
	if m := thumb.object("musicThumbnailRenderer"); m != nil {
		thumbnail = bestThumb(m.object("thumbnail"))
	} else {
		thumbnail = bestThumb(thumb)
	}
	return title, thumbnail
}

// parseTracks reads every track of the response, in the order they appear.
func parseTracks(root any) []music.Track {
	var tracks []music.Track
	walk(root, func(o *omap) bool {
		if item := o.object("musicResponsiveListItemRenderer"); item != nil {
			if track, ok := parseMusicItem(item); ok {
				tracks = append(tracks, track)
			}
			return false
		}
		if item := o.object("playlistVideoRenderer"); item != nil {
			if track, ok := parseVideoItem(item); ok {
				tracks = append(tracks, track)
			}
			return false
		}
		return true
	})
	return tracks
}

// parseMusicItem reads a track of a YouTube Music list.
func parseMusicItem(item *omap) (music.Track, bool) {
	track := music.Track{}
	if data := item.object("playlistItemData"); data != nil {
		track.ID = data.text("videoId")
	}
	if track.ID == "" {
		track.ID = playEndpointVideoID(item)
	}
	if track.ID == "" {
		return music.Track{}, false
	}
	columns := item.array("flexColumns")
	if len(columns) > 0 {
		track.Title = textOf(columnText(columns[0]))
	}
	if len(columns) > 1 {
		track.Artist, track.Album = splitByline(textOf(columnText(columns[1])))
	}
	track.Duration = parseDuration(textOf(fixedColumn(item)))
	if thumb := item.object("thumbnail"); thumb != nil {
		if m := thumb.object("musicThumbnailRenderer"); m != nil {
			track.Thumbnail = bestThumb(m.object("thumbnail"))
		} else {
			track.Thumbnail = bestThumb(thumb)
		}
	}
	return track, true
}

// parseVideoItem reads a track of an ordinary YouTube playlist.
func parseVideoItem(item *omap) (music.Track, bool) {
	track := music.Track{ID: item.text("videoId")}
	if track.ID == "" {
		return music.Track{}, false
	}
	track.Title = textOf(item.object("title"))
	track.Artist = textOf(item.object("shortBylineText"))
	if seconds, err := strconv.Atoi(item.text("lengthSeconds")); err == nil {
		track.Duration = time.Duration(seconds) * time.Second
	}
	if track.Duration == 0 {
		track.Duration = parseDuration(textOf(item.object("lengthText")))
	}
	if thumb := item.object("thumbnail"); thumb != nil {
		track.Thumbnail = bestThumb(thumb)
	}
	return track, true
}

// columnText reads the text of a column of a list item.
func columnText(column any) *omap {
	o, ok := column.(*omap)
	if !ok {
		return nil
	}
	for _, key := range []string{
		"musicResponsiveListItemFlexColumnRenderer",
		"musicResponsiveListItemFixedColumnRenderer",
	} {
		if renderer := o.object(key); renderer != nil {
			return renderer.object("text")
		}
	}
	return nil
}

// fixedColumn reads the column of a list item that holds its length. The
// column is a list in some responses and a single object in others.
func fixedColumn(item *omap) *omap {
	if items := item.array("fixedColumns"); len(items) > 0 {
		return columnText(items[0])
	}
	return columnText(item.object("fixedColumns"))
}

// playEndpointVideoID reads the video id out of the play button of an item.
func playEndpointVideoID(item *omap) string {
	overlay := item.object("overlay")
	if overlay == nil {
		return ""
	}
	button := overlay.object("musicItemThumbnailOverlayRenderer")
	if button == nil {
		return ""
	}
	content := button.object("content")
	if content == nil {
		return ""
	}
	play := content.object("musicPlayButtonRenderer")
	if play == nil {
		return ""
	}
	endpoint := play.object("playNavigationEndpoint")
	if endpoint == nil {
		return ""
	}
	watch := endpoint.object("watchEndpoint")
	if watch == nil {
		return ""
	}
	return watch.text("videoId")
}

// splitByline reads the artist and the album out of a byline, which reads
// "Artist • Album • Year".
func splitByline(byline string) (artist, album string) {
	parts := strings.Split(byline, "•")
	for i, part := range parts {
		part = strings.TrimSpace(part)
		switch {
		case i == 0 && part != "":
			artist = part
		case i == 1 && part != "":
			album = part
		}
	}
	return artist, album
}

// textOf reads the text of a text node, which holds either a list of runs or
// simple text.
func textOf(node *omap) string {
	if node == nil {
		return ""
	}
	if runs := node.array("runs"); len(runs) > 0 {
		var b strings.Builder
		for _, run := range runs {
			if o, ok := run.(*omap); ok {
				b.WriteString(o.text("text"))
			}
		}
		return b.String()
	}
	if s := node.text("simpleText"); s != "" {
		return s
	}
	if inner := node.object("text"); inner != nil {
		return textOf(inner)
	}
	return ""
}

// bestThumb returns the largest of a list of thumbnails.
func bestThumb(node *omap) string {
	if node == nil {
		return ""
	}
	list := node.array("thumbnails")
	best, bestArea := "", 0
	for _, entry := range list {
		o, ok := entry.(*omap)
		if !ok {
			continue
		}
		url := o.text("url")
		if url == "" {
			continue
		}
		width := o.number("width")
		height := o.number("height")
		if area := width * height; area > bestArea {
			best, bestArea = url, area
		}
	}
	return best
}

// parseDuration reads a length such as "3:42" or "1:02:03".
func parseDuration(text string) time.Duration {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0
	}
	parts := strings.Split(text, ":")
	var seconds int
	for _, part := range parts {
		value, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return 0
		}
		seconds = seconds*60 + value
	}
	return time.Duration(seconds) * time.Second
}

// findContinuation reads the token that asks for the next page of a list.
func findContinuation(root any) string {
	token := ""
	walk(root, func(o *omap) bool {
		item := o.object("continuationItemRenderer")
		if item == nil {
			return true
		}
		endpoint := item.object("continuationEndpoint")
		if endpoint == nil {
			return true
		}
		command := endpoint.object("continuationCommand")
		if command == nil {
			return true
		}
		token = command.text("token")
		return false
	})
	return token
}
