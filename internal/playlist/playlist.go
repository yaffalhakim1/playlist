// Package playlist loads a playlist from a link the user pasted.
//
// Two readers know how to read YouTube playlists. InnerTube answers with the
// artist, the album and the artwork of every track, but it cannot read the
// mixes YouTube generates on the fly. yt-dlp reads those, but knows only the
// title of a track. The reader that knows more wins, and the other is the
// way out.
package playlist

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"youtube-playlist/internal/innertube"
	"youtube-playlist/internal/music"
	"youtube-playlist/internal/ytdlp"
)

// Load reads the playlist a link points at.
func Load(ctx context.Context, ref string) (*music.Playlist, error) {
	rich, richErr := innertube.New().Playlist(ctx, ref)
	if richErr == nil && len(rich.Tracks) > 0 {
		return rich, nil
	}
	plain, plainErr := ytdlp.Playlist(ctx, link(ref))
	if plainErr == nil {
		return plain, nil
	}
	if richErr == nil {
		richErr = errors.New("YouTube would not show that playlist")
	}
	return nil, fmt.Errorf("%w (yt-dlp says: %v)", richErr, plainErr)
}

// link turns what the user pasted into a link yt-dlp understands.
//
// A link is kept as it is, with the tokens it carries: the mixes YouTube
// generates are only viewable with the token that comes with the link.
func link(ref string) string {
	ref = strings.TrimSpace(ref)
	if strings.Contains(ref, "://") {
		return ref
	}
	if id, err := innertube.ParsePlaylistID(ref); err == nil {
		return "https://music.youtube.com/playlist?list=" + url.QueryEscape(id)
	}
	return ref
}
