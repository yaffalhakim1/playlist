package src

import (
	"log"
	"strings"
	"sync"
	"time"

	"youtube-playlist/internal/discord"
	"youtube-playlist/internal/music"
)

// discordClientID is the application id the presence is registered under.
//
// Discord draws the panel with the name and pictures of this application, so it
// has to exist in the Discord developer portal before presence shows anything.
// The panel reads "Axolotl" because that is what this application is called
// there; renaming it in the portal renames the panel, and needs no rebuild.
const discordClientID = "1562293136137523412"

// presence shows what plays in Discord.
//
// Discord is often not running, its socket may be closed, and the application id
// may not be set. None of that is a reason for the music to stop, so every
// failure is logged once and the feature stays quietly off.
type presence struct {
	mu      sync.Mutex
	client  *discord.Client
	enabled bool
	// failed remembers that a connection was tried and refused, so the app does
	// not open a socket on every track.
	failed bool
	// shown is the track last sent, so the same track is not sent twice.
	shown string
	// started is when the track that plays began, for Discord's counter.
	started time.Time
}

// enable turns the integration on or off.
func (p *presence) enable(on bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.enabled = on
	if !on {
		p.clearLocked()
		p.failed = false
		return
	}
	// A new enable is a new attempt, even if the last one failed.
	p.failed = false
}

// enabledNow tells whether the integration is on.
func (p *presence) enabledNow() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.enabled
}

// show puts a track on the panel, or clears it when the track is empty.
//
// It runs off the thread that draws: a socket write must not hold up a frame.
func (p *presence) show(track music.Track, playing bool) {
	p.mu.Lock()
	if !p.enabled {
		p.mu.Unlock()
		return
	}
	if track.Title == "" {
		p.clearLocked()
		p.mu.Unlock()
		return
	}
	// The same track and the same state needs no frame.
	key := track.ID + "|" + track.Artist
	if !playing {
		key += "|paused"
	}
	if key == p.shown {
		p.mu.Unlock()
		return
	}
	p.shown = key
	started := p.started
	if playing && started.IsZero() {
		started = time.Now()
		p.started = started
	}
	p.mu.Unlock()

	go p.send(track, playing, started)
}
// send connects if it must, and sets the activity.
func (p *presence) send(track music.Track, playing bool, started time.Time) {
	p.mu.Lock()
	client := p.client
	failed := p.failed
	p.mu.Unlock()

	if client == nil {
		if failed {
			return
		}
		c, err := discord.Connect(discordClientID)
		if err != nil {
			// Discord not running is the ordinary case, so it is noted once and
			// not retried until the integration is turned on again.
			log.Printf("discord: %v", err)
			p.mu.Lock()
			p.failed = true
			p.mu.Unlock()
			return
		}
		p.mu.Lock()
		p.client = c
		p.mu.Unlock()
		client = c
	}

	activity := discord.Activity{
		Details: track.Title,
		// Discord shows `details` and `state` as the two lines under the name,
		// and a newline inside `state` is drawn as a line of its own. That is
		// how the artist and the album both get a line: the protocol has no
		// third text field.
		State: stateLine(track),
		// Type 2 is Listening. Discord labels that "Listening to <application>",
		// which is why the application is named Axolotl rather than "Music":
		// the word comes from the application, not from this field.
		Type: 2,
	}
	// Both ends of the timestamp are needed for the progress bar: Discord draws
	// the elapsed time against the total only when it knows where the track
	// ends. With a start alone it shows a counter and no bar.
	if !started.IsZero() && playing && track.Duration > 0 {
		activity.Timestamps = &discord.Timestamps{
			Start: started.Unix(),
			End:   started.Add(track.Duration).Unix(),
		}
	} else if !started.IsZero() && playing {
		// A length the app does not know: the counter still runs, without a bar.
		activity.Timestamps = &discord.Timestamps{Start: started.Unix()}
	}
	if track.Thumbnail != "" {
		activity.Assets = &discord.Assets{
			LargeImage: track.Thumbnail,
			LargeText:  track.Album,
		}
	}
	if err := client.SetActivity(activity); err != nil {
		log.Printf("discord: %v", err)
		p.mu.Lock()
		p.client = nil
		p.failed = true
		p.mu.Unlock()
		client.Close()
	}
}

// stateLine is the second line of the panel: the artist, and the album under
// it when the track knows one.
//
// The protocol has two text fields, not three, but Discord draws a newline
// inside `state` as its own line. That is the only way the album gets a line of
// its own, as the reference screenshots show.
//
// Newlines inside the artist or the album are flattened first: a title with a
// line break in it would otherwise push the panel into three or four lines, and
// the one newline this sends is the one it means.
func stateLine(track music.Track) string {
	artist := oneLine(track.Artist)
	album := oneLine(track.Album)
	if album == "" || album == artist {
		return artist
	}
	if artist == "" {
		return album
	}
	return artist + "\n" + album
}

// oneLine flattens the line breaks in a string, so a title cannot break the
// panel's layout.
func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return strings.TrimSpace(s)
}

// clearLocked takes the panel down. The caller holds the lock.
func (p *presence) clearLocked() {
	p.shown = ""
	p.started = time.Time{}
	client := p.client
	p.client = nil
	if client != nil {
		go func() {
			_ = client.Clear()
			_ = client.Close()
		}()
	}
}

// close ends the connection, which takes the panel down.
func (p *presence) close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.clearLocked()
}

// resetTrack forgets when the track started, so the next one counts from its own
// beginning rather than carrying the last one's clock.
func (p *presence) resetTrack() {
	p.mu.Lock()
	p.started = time.Time{}
	p.mu.Unlock()
}
