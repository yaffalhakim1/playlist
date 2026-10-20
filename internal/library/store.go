// Package library keeps what the app remembers between runs: the playlists
// that were played, and the settings of the player.
package library

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
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
	Plays   []Play   `json:"plays"`
	Volume  float64  `json:"volume"`
	Shuffle bool     `json:"shuffle"`
	Repeat  string   `json:"repeat"`
	Mode    string   `json:"mode"`
	Accent  string   `json:"accent"`
	Dynamic bool     `json:"dynamic"`
}

// maxRecents is how many playlists are remembered.
const maxRecents = 20

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
func (s *Store) Appearance() (mode, accent string, dynamic bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data.Mode, s.data.Accent, s.data.Dynamic
}

// SetAppearance remembers what the user chose about the look of the app.
func (s *Store) SetAppearance(mode, accent string, dynamic bool) {
	s.mu.Lock()
	s.data.Mode = mode
	s.data.Accent = accent
	s.data.Dynamic = dynamic
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
