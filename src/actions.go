package src

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"youtube-playlist/internal/library"
	"youtube-playlist/internal/music"
	"youtube-playlist/internal/player"
)

// libraryPlay is a track that was played.
type libraryPlay = library.Play

// currentIndex returns the place of the track that plays, or -1.
func (a *app) currentIndex() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.current
}

// currentTrack returns the track that plays.
func (a *app) currentTrack() music.Track {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.current < 0 || a.current >= len(a.queue) {
		return music.Track{}
	}
	return a.queue[a.current]
}

// times returns how far the track has played, and how long it is.
func (a *app) times() (position, duration time.Duration) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.position, a.duration
}

// times2 returns how far the track has played.
func (a *app) times2() time.Duration {
	position, _ := a.times()
	return position
}

// durationOf returns the length of the track that plays.
func (a *app) durationOf() time.Duration {
	_, duration := a.times()
	return duration
}

// isPlaying tells whether a track plays.
func (a *app) isPlaying() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.playing
}

// isShuffle tells whether the queue plays in a random order.
func (a *app) isShuffle() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.shuffle
}

// repeatMode says what happens at the end of the queue.
func (a *app) repeatMode() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	switch a.repeat {
	case player.RepeatAll:
		return 1
	case player.RepeatOne:
		return 2
	default:
		return 0
	}
}

// volumeLevel returns the volume of the player, from 0 to 1.
func (a *app) volumeLevel() float64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.volume
}

// plays returns what was listened to.
func (a *app) plays() []libraryPlay {
	if a.store == nil {
		return nil
	}
	return a.store.Plays()
}

// playAt starts a track of the queue at a place in it.
func (a *app) playAt(index int, at time.Duration) {
	if a.engine == nil {
		return
	}
	// A track started from here was picked by the user, which Last.fm records.
	a.mu.Lock()
	a.chosenTrack = true
	a.mu.Unlock()
	a.engine.PlayAt(index, at)
}

// play starts a track of the queue.
func (a *app) play(index int) {
	if a.engine == nil {
		return
	}
	a.mu.Lock()
	a.chosenTrack = true
	a.mu.Unlock()
	a.engine.Play(index)
}

// playAll starts the queue from its first track.
func (a *app) playAll() {
	if len(a.queue) == 0 {
		return
	}
	a.mu.Lock()
	a.chosenTrack = true
	a.mu.Unlock()
	a.engine.Play(0)
}

// toggleShuffle turns the random order on or off.
func (a *app) toggleShuffle() {
	a.mu.Lock()
	a.shuffle = !a.shuffle
	shuffle := a.shuffle
	a.mu.Unlock()
	if a.engine != nil {
		a.engine.SetShuffle(shuffle)
	}
	if a.store != nil {
		a.store.SetShuffle(shuffle)
	}
}

// setShuffle turns the random order on or off, as the user asked.
func (a *app) setShuffle(shuffle bool) {
	a.mu.Lock()
	a.shuffle = shuffle
	a.mu.Unlock()
	if a.engine != nil {
		a.engine.SetShuffle(shuffle)
	}
	if a.store != nil {
		a.store.SetShuffle(shuffle)
	}
}

// setRepeat says what happens at the end of the queue, as the user asked.
func (a *app) setRepeat(mode int) {
	a.mu.Lock()
	switch mode {
	case 1:
		a.repeat = player.RepeatAll
	case 2:
		a.repeat = player.RepeatOne
	default:
		a.repeat = player.RepeatOff
	}
	repeat := a.repeat
	a.mu.Unlock()
	if a.engine != nil {
		a.engine.SetRepeat(repeat)
	}
	if a.store != nil {
		a.store.SetRepeat(nameOfRepeat(repeat))
	}
}

// shuffleAndPlay starts the queue at random.
func (a *app) shuffleAndPlay() {
	a.setShuffle(true)
	a.playAll()
}

// cycleRepeat moves through what happens at the end of the queue: nothing,
// the queue again, the same track again.
func (a *app) cycleRepeat() {
	a.mu.Lock()
	switch a.repeat {
	case player.RepeatOff:
		a.repeat = player.RepeatAll
	case player.RepeatAll:
		a.repeat = player.RepeatOne
	default:
		a.repeat = player.RepeatOff
	}
	repeat := a.repeat
	a.mu.Unlock()
	if a.engine != nil {
		a.engine.SetRepeat(repeat)
	}
	if a.store != nil {
		a.store.SetRepeat(nameOfRepeat(repeat))
	}
}

// rememberVolume keeps the volume for the next run.
func (a *app) rememberVolume(volume float64) {
	if a.store != nil {
		a.store.SetVolume(volume)
	}
}

// clearHistory forgets what was listened to.
func (a *app) clearHistory() {
	if a.store != nil {
		a.store.ClearPlays()
	}
}

// pasteLink reads what is on the clipboard and loads it.
func (a *app) pasteLink() {
	text := mygo.Clipboard.ReadText()
	if strings.TrimSpace(text) == "" {
		return
	}
	a.link = strings.TrimSpace(text)
	a.load()
}

// askDelete asks before a playlist is forgotten, because deleting is the one
// thing in the app that cannot be undone.
func (a *app) askDelete(link, title string) {
	if title == "" {
		title = link
	}
	a.deleting = pendingDelete{link: link, title: title, open: true}
}

// confirmDelete draws the question, and forgets the playlist when the answer
// is yes.
//
// It is called from the frame, because a dialog is drawn and the answer comes
// back a frame later. The buttons are "Delete" and "Cancel", in that order,
// with the destructive one second: a stray Enter should keep the playlist, not
// lose it.
func (a *app) confirmDelete(c *ui.Context) {
	if !a.deleting.open {
		return
	}
	open := true
	answer := ui.AlertDialog(c, &open,
		"Delete this playlist?",
		fmt.Sprintf("“%s” leaves the sidebar and the cache. What you listened to stays in your stats.",
			a.deleting.title),
		"Cancel", "Delete")
	if answer == 1 {
		a.deletePlaylist(a.deleting.link)
	}
	if answer >= 0 {
		a.deleting = pendingDelete{}
	}
}

// deletePlaylist forgets a playlist: the sidebar entry, the cached tracks, and
// the session if it was the one open.
func (a *app) deletePlaylist(link string) {
	if a.store != nil {
		a.store.ForgetPlaylist(link)
		a.recents = a.store.Recents()
	}

	a.mu.Lock()
	wasOpen := a.openLink == link
	if wasOpen {
		// The playlist that was open is gone, so nothing plays and the page
		// goes home rather than showing a playlist that no longer exists.
		a.playlist = nil
		a.queue = nil
		a.visible = nil
		a.current = -1
		a.position = 0
		a.duration = 0
		a.link = ""
		a.openLink = ""
		a.page = pageHome
	}
	a.mu.Unlock()

	if wasOpen && a.engine != nil {
		a.engine.SetQueue(nil)
	}
}
