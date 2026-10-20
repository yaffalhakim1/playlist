package player

import (
	"time"

	"youtube-playlist/internal/music"
)

// tracks returns a queue of the given length.
func tracks(count int) []music.Track {
	list := make([]music.Track, count)
	for i := range list {
		list[i] = music.Track{
			ID:       "id" + string(rune('a'+i)),
			Title:    "Track",
			Duration: time.Duration(i+1) * time.Minute,
		}
	}
	return list
}
