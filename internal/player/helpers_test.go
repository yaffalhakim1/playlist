package player

import (
	"time"

	"youtube-playlist/internal/music"
	"youtube-playlist/internal/ytdlp"
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

// engineStubResolver gives the engine a resolver, so a load starts rather than
// panicking on a nil one.
//
// The tests here check where the queue sits, not what it plays: the resolver
// is real but the video ids are not, so the load fails on its own and nothing
// is fetched.
func (e *Engine) engineStubResolver() {
	e.resolver = ytdlp.NewResolver()
}
