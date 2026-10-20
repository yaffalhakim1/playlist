// Package music holds the domain model the app plays: tracks and playlists.
//
// The types are the app's own language: the UI, the playback engine and the
// metadata readers all speak them, so no layer has to know about YouTube's
// JSON shapes.
package music

import "time"

// Track is one playable item: a YouTube video heard for its audio.
type Track struct {
	// ID is the YouTube video id.
	ID string
	// Title is the name of the track.
	Title string
	// Artist is the performer, empty when the source does not say.
	Artist string
	// Album is the release the track belongs to, empty when unknown.
	Album string
	// Duration is how long the track plays, zero when unknown.
	Duration time.Duration
	// Thumbnail is the URL of the artwork.
	Thumbnail string
}

// Playlist is an ordered set of tracks with a name and artwork of its own.
type Playlist struct {
	// ID is the YouTube playlist id.
	ID string
	// Title is the name of the playlist.
	Title string
	// Thumbnail is the URL of the playlist artwork.
	Thumbnail string
	// Tracks are the items, in the order the playlist lists them.
	Tracks []Track
}
