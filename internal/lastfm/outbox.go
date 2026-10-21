package lastfm

import (
	"cmp"
	"encoding/json"
	"errors"
	"log"
	"os"
	"slices"
	"sync"
	"time"
)

const (
	// MaxPending is how many undelivered plays are kept; the oldest go first.
	MaxPending = 2000
	// MaxAge is how old a play may be and still be sent: Last.fm ignores
	// scrobbles from more than two weeks ago.
	MaxAge = 13 * 24 * time.Hour
	// MaxBatch is how many plays Last.fm takes in one call.
	MaxBatch = 50
)

// stored is a pending play as it is written to disk.
type stored struct {
	Artist   string `json:"artist"`
	Title    string `json:"title"`
	Album    string `json:"album,omitempty"`
	Duration int    `json:"duration,omitempty"` // seconds
	Started  int64  `json:"started"`            // Unix seconds
	Chosen   bool   `json:"chosen,omitempty"`
}

func store(s Scrobble) stored {
	return stored{
		Artist:   s.Track.Artist,
		Title:    s.Track.Title,
		Album:    s.Track.Album,
		Duration: int(s.Track.Duration / time.Second),
		Started:  s.Started.Unix(),
		Chosen:   s.Chosen,
	}
}

func (s stored) scrobble() Scrobble {
	return Scrobble{
		Track: Track{
			Artist:   s.Artist,
			Title:    s.Title,
			Album:    s.Album,
			Duration: time.Duration(s.Duration) * time.Second,
		},
		Started: time.Unix(s.Started, 0),
		Chosen:  s.Chosen,
	}
}

// Outbox keeps the plays that qualified until Last.fm has them, on disk, so
// they survive a restart and a night offline. Each keeps the time its playback
// began, because that is what Last.fm files it under.
//
// Changes are in memory at once and written by a goroutine of the outbox's own,
// so a caller never waits on the disk. It is safe for concurrent use. The zero
// value holds plays in memory only.
type Outbox struct {
	path string

	mu      sync.Mutex
	items   []stored
	dirty   bool
	running bool
	done    chan struct{}
}

// OpenOutbox reads the outbox kept at path. A file that is missing or not
// readable is an empty outbox.
func OpenOutbox(path string) *Outbox {
	o := &Outbox{path: path}
	if path == "" {
		return o
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Printf("last.fm: reading pending scrobbles: %v", err)
		}
		return o
	}
	if err := json.Unmarshal(data, &o.items); err != nil {
		log.Printf("last.fm: pending scrobbles in %s are not readable, starting fresh: %v", path, err)
		o.items = nil
	}
	return o
}

// Add keeps a play to send.
func (o *Outbox) Add(s Scrobble) {
	o.mu.Lock()
	o.items = append(o.items, store(s))
	if over := len(o.items) - MaxPending; over > 0 {
		o.items = slices.Delete(o.items, 0, over)
	}
	o.changedLocked()
	o.mu.Unlock()
}

// Len is how many plays wait.
func (o *Outbox) Len() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return len(o.items)
}

// Batch is the oldest plays waiting, up to Last.fm's limit, to send now. Plays
// too old for Last.fm to take are dropped first.
//
// The batch is ordered by the time each play began, oldest first, rather than by
// the order they were added. A queue is filled by whatever played, but the
// plays about to expire are the ones worth sending before they are lost.
func (o *Outbox) Batch(now time.Time) []Scrobble {
	o.mu.Lock()
	defer o.mu.Unlock()
	cutoff := now.Add(-MaxAge).Unix()
	if n := len(o.items); n > 0 {
		o.items = slices.DeleteFunc(o.items, func(s stored) bool { return s.Started < cutoff })
		if len(o.items) != n {
			o.changedLocked()
		}
	}
	slices.SortStableFunc(o.items, func(a, b stored) int {
		return cmp.Compare(a.Started, b.Started)
	})
	batch := make([]Scrobble, 0, min(len(o.items), MaxBatch))
	for _, s := range o.items[:min(len(o.items), MaxBatch)] {
		batch = append(batch, s.scrobble())
	}
	return batch
}

// Done removes the n oldest plays: they were delivered, or can never be.
func (o *Outbox) Done(n int) {
	o.mu.Lock()
	defer o.mu.Unlock()
	n = min(n, len(o.items))
	if n <= 0 {
		return
	}
	o.items = slices.Delete(o.items, 0, n)
	o.changedLocked()
}

// Clear drops every waiting play.
func (o *Outbox) Clear() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if len(o.items) == 0 {
		return
	}
	o.items = nil
	o.changedLocked()
}

// changedLocked schedules a write. o.mu is held.
func (o *Outbox) changedLocked() {
	if o.path == "" {
		return
	}
	o.dirty = true
	if o.running {
		return
	}
	o.running, o.done = true, make(chan struct{})
	go o.drain(o.done)
}

func (o *Outbox) drain(done chan struct{}) {
	defer close(done)
	for {
		o.mu.Lock()
		if !o.dirty {
			o.running = false
			o.mu.Unlock()
			return
		}
		items := slices.Clone(o.items)
		o.dirty = false
		o.mu.Unlock()
		if items == nil {
			items = []stored{} // "[]", not "null"
		}
		data, err := json.Marshal(items)
		if err == nil {
			err = writeFileAtomic(o.path, data, 0o600)
		}
		if err != nil {
			log.Printf("last.fm: saving pending scrobbles: %v", err)
		}
	}
}

// Wait returns once what changed so far is on disk.
func (o *Outbox) Wait() {
	o.mu.Lock()
	done := o.done
	running := o.running
	o.mu.Unlock()
	if running {
		<-done
	}
}
