// Package library keeps what the app remembers between runs: the playlists
// that were read, the track that played last, and the settings of the player.
package library

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"youtube-playlist/internal/music"
)

// Recent is a playlist that was played.
type Recent struct {
	// Link is what the user pasted.
	Link string `json:"link"`
	// Title is the name of the playlist, as it was read.
	Title string `json:"title"`
	// Thumbnail is the artwork of the playlist.
	Thumbnail string `json:"thumbnail"`
	// At is when it was played.
	At time.Time `json:"at"`
}

// Cached is a playlist the app has read, kept so that opening it again costs
// no network at all: the tracks, their artwork and their lengths are here.
type Cached struct {
	// Link is the address the playlist was read from, which is the key.
	Link string `json:"link"`
	// Playlist is what was read: its name, its cover and its tracks.
	Playlist music.Playlist `json:"playlist"`
	// At is when it was read, so the oldest can be let go first.
	At time.Time `json:"at"`
}

// Session is where the player was when the app was last closed: the playlist
// that was open, the track that played, and how far into it.
type Session struct {
	// Link is the playlist that was open.
	Link string `json:"link"`
	// TrackID is the track that played.
	TrackID string `json:"trackId"`
	// Position is how far into the track it had played.
	Position time.Duration `json:"position"`
}

// Play is a track that was played.
type Play struct {
	// TrackID is the video the track plays.
	TrackID string `json:"trackId"`
	// Title is the name of the track.
	Title string `json:"title"`
	// Artist is the performer, when it is known.
	Artist string `json:"artist"`
	// At is when it started.
	At time.Time `json:"at"`
	// Listened is how long it played.
	Listened time.Duration `json:"listened"`
}

// Data is what the store keeps.
type Data struct {
	Recents []Recent `json:"recents"`
	Cached  []Cached `json:"cached"`
	Session Session  `json:"session"`
	Plays   []Play   `json:"plays"`
	Volume  float64  `json:"volume"`
	Shuffle bool     `json:"shuffle"`
	Repeat  string   `json:"repeat"`
	Mode    string   `json:"mode"`
	Accent  string   `json:"accent"`
	Dynamic bool     `json:"dynamic"`
	Palette string   `json:"palette"`
}

// maxRecents is how many playlists are remembered.
const maxRecents = 20

// maxCached is how many playlists keep their tracks, so opening one again
// needs no network.
//
// The limit is on size, not count: a mix with a thousand tracks is a few
// hundred kilobytes, and ten of those would be a slow file to read and write on
// every position report.
const (
	maxCached      = 10
	maxCachedBytes = 4 << 20
)

// MaxCached returns how many playlists keep their tracks.
func MaxCached() int { return maxCached }

// maxPlays is how many plays are remembered.
const maxPlays = 2000

// Store keeps the data in a file, and writes it back when it changes.
type Store struct {
	mu   sync.Mutex
	path string
	data Data
}

// Open reads the store in a file, and writes it back when it changes.
//
// A store that cannot be read is an empty one: the app still works, it just
// forgets.
func Open(path string) (*Store, error) {
	s := &Store{path: path, data: Data{Volume: 0.8}}
	data, err := os.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(data, &s.data); err != nil {
			return s, err
		}
	} else if !os.IsNotExist(err) {
		return s, err
	}
	return s, nil
}

// Recents returns the playlists that were played, the newest first.
func (s *Store) Recents() []Recent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Recent(nil), s.data.Recents...)
}

// Volume returns the volume of the player.
func (s *Store) Volume() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data.Volume
}

// Shuffle tells whether the queue plays in a random order.
func (s *Store) Shuffle() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data.Shuffle
}

// Repeat returns what happens at the end of the queue, as its name.
func (s *Store) Repeat() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data.Repeat
}

// SetVolume remembers the volume of the player.
func (s *Store) SetVolume(volume float64) {
	s.mu.Lock()
	s.data.Volume = volume
	s.mu.Unlock()
	s.save()
}

// SetShuffle remembers whether the queue plays in a random order.
func (s *Store) SetShuffle(shuffle bool) {
	s.mu.Lock()
	s.data.Shuffle = shuffle
	s.mu.Unlock()
	s.save()
}

// SetRepeat remembers what happens at the end of the queue.
func (s *Store) SetRepeat(repeat string) {
	s.mu.Lock()
	s.data.Repeat = repeat
	s.mu.Unlock()
	s.save()
}

// AddRecent remembers a playlist that was played, and puts it first.
func (s *Store) AddRecent(link, title, thumbnail string) {
	s.mu.Lock()
	recents := []Recent{{Link: link, Title: title, Thumbnail: thumbnail, At: time.Now()}}
	for _, recent := range s.data.Recents {
		if recent.Link == link {
			continue
		}
		recents = append(recents, recent)
	}
	if len(recents) > maxRecents {
		recents = recents[:maxRecents]
	}
	s.data.Recents = recents
	s.mu.Unlock()
	s.save()
}

// CachedPlaylist returns the tracks of a playlist that was read before, and
// whether it is there at all.
//
// A cached playlist is what makes opening one again instant: the tracks come
// from here, and nothing goes to the network.
func (s *Store) CachedPlaylist(link string) (music.Playlist, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, cached := range s.data.Cached {
		if cached.Link == link {
			return cached.Playlist, true
		}
	}
	return music.Playlist{}, false
}

// CachePlaylist keeps the tracks of a playlist, so opening it again needs no
// network, and puts it first.
//
// A playlist too large to keep is left out rather than crowding the file: the
// next open of it reads it again, which is the right trade for a mix of a
// thousand tracks.
func (s *Store) CachePlaylist(link string, list music.Playlist) {
	size := 0
	for _, track := range list.Tracks {
		size += len(track.ID) + len(track.Title) + len(track.Artist) +
			len(track.Album) + len(track.Thumbnail) + 64
	}
	if size > maxCachedBytes {
		return
	}
	s.mu.Lock()
	cached := []Cached{{Link: link, Playlist: list, At: time.Now()}}
	for _, item := range s.data.Cached {
		if item.Link == link {
			continue
		}
		cached = append(cached, item)
	}
	if len(cached) > maxCached {
		cached = cached[:maxCached]
	}
	s.data.Cached = cached
	s.mu.Unlock()
	s.save()
}

// Session returns where the player was when the app was last closed.
func (s *Store) Session() Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data.Session
}

// SetSession remembers where the player was, so the next run carries on from
// there rather than starting the playlist again.
func (s *Store) SetSession(session Session) {
	s.mu.Lock()
	s.data.Session = session
	s.mu.Unlock()
	s.save()
}

// ForgetPlaylist removes a playlist from everything the app remembers about
// it: the recent list, the cached tracks, and the session, if it was the one
// open.
//
// The plays are left alone: the stats are a record of what was listened to,
// and forgetting a playlist is not a reason to forget that it was heard.
func (s *Store) ForgetPlaylist(link string) {
	s.mu.Lock()
	recents := s.data.Recents[:0]
	for _, recent := range s.data.Recents {
		if recent.Link != link {
			recents = append(recents, recent)
		}
	}
	s.data.Recents = recents

	cached := s.data.Cached[:0]
	for _, item := range s.data.Cached {
		if item.Link != link {
			cached = append(cached, item)
		}
	}
	s.data.Cached = cached

	if s.data.Session.Link == link {
		s.data.Session = Session{}
	}
	s.mu.Unlock()
	s.save()
}

// Plays returns what was played, oldest first.
func (s *Store) Plays() []Play {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Play(nil), s.data.Plays...)
}

// AddPlay remembers that a track played.
func (s *Store) AddPlay(play Play) {
	s.mu.Lock()
	s.data.Plays = append(s.data.Plays, play)
	if len(s.data.Plays) > maxPlays {
		s.data.Plays = s.data.Plays[len(s.data.Plays)-maxPlays:]
	}
	s.mu.Unlock()
	s.save()
}

// FinishLastPlay writes how long the last track played.
func (s *Store) FinishLastPlay(listened time.Duration) {
	s.mu.Lock()
	if len(s.data.Plays) > 0 {
		s.data.Plays[len(s.data.Plays)-1].Listened = listened
	}
	s.mu.Unlock()
	s.save()
}

// ClearPlays forgets what was listened to.
func (s *Store) ClearPlays() {
	s.mu.Lock()
	s.data.Plays = nil
	s.mu.Unlock()
	s.save()
}

// Appearance returns what the user chose about the look of the app.
func (s *Store) Appearance() (mode, accent string, dynamic bool, palette string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data.Mode, s.data.Accent, s.data.Dynamic, s.data.Palette
}

// SetAppearance remembers what the user chose about the look of the app.
func (s *Store) SetAppearance(mode, accent string, dynamic bool, palette string) {
	s.mu.Lock()
	s.data.Mode = mode
	s.data.Accent = accent
	s.data.Dynamic = dynamic
	s.data.Palette = palette
	s.mu.Unlock()
	s.save()
}

// save writes the data back, and says so when it cannot.
func (s *Store) save() {
	s.mu.Lock()
	data, err := json.MarshalIndent(s.data, "", "  ")
	path := s.path
	s.mu.Unlock()
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o644)
}
