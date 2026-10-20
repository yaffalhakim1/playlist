// Package src is the interface of the app: one native window, drawn by MyGo.
//
// The window is a rail of destinations at the left and a sheet at the right
// that holds the page. The view is a function of the state, and the state is
// only ever changed on the thread that draws: the playback engine reports
// what happens on a channel, and the window applies it with win.Update.
package src

import (
	"context"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/shells/sidebar"

	"youtube-playlist/internal/library"
	"youtube-playlist/internal/music"
	"youtube-playlist/internal/player"
	"youtube-playlist/internal/playlist"
	"youtube-playlist/internal/ytdlp"
)

// loadTimeout bounds the reading of a playlist.
const loadTimeout = 3 * time.Minute

// sessionInterval is how often the place in the track is written down: often
// enough that a crash costs a second, rare enough that the file is not
// rewritten five times a second.
const sessionInterval = 5 * time.Second

// The pages of the app.
const (
	pageHome       = "home"
	pagePlaylist   = "playlist"
	pagePlayer     = "player"
	pageQueue      = "queue"
	pageStats      = "stats"
	pageAppearance = "appearance"
)

// notice is a message the app shows once, in the next frame.
type notice struct {
	title string
	body  string
	kind  string
}

// app is the whole interface: the state it shows and the actions it takes.
type app struct {
	mu sync.Mutex

	page     string
	link     string
	filter   string
	playlist *music.Playlist
	queue    []music.Track
	visible  []int
	status   string
	loading  bool

	current   int
	position  time.Duration
	duration  time.Duration
	playing   bool
	volume    float64
	shuffle   bool
	repeat    player.Repeat
	seekValue float64
	scrubbing bool
	scrubAt   time.Time
	// sessionAt is when the place in the track was last written down.
	sessionAt time.Time

	recents []library.Recent
	art     *artwork

	engine     *player.Engine
	win        *mygo.Window
	store      *library.Store
	notice     *notice
	generation int
	tray       *tray

	appearance appearance

	list          components.ListState
	queueL        components.ListState
	sidebarPixels float32
	closed        bool
}

// appearance holds what the user chose about the look of the app.
type appearance struct {
	// mode is "system", "light" or "dark".
	mode string
	// accent is the colour of the accent, as a hex string, empty for the
	// colour of the design.
	accent string
	// dynamic takes the colour from the artwork of the track that plays.
	dynamic bool
}

// Run opens the window and runs the app until it closes.
func Run() {
	a := &app{art: newArtwork(), current: -1, page: pageHome}
	mygo.App.WhenReady(func() {
		a.setup()
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

// setup opens the store, the engine and the window.
func (a *app) setup() {
	if path, err := mygo.App.Path(mygo.PathUserData); err == nil {
		store, err := library.Open(filepath.Join(path, "settings.json"))
		if err != nil {
			log.Printf("settings: %v", err)
		}
		if store != nil {
			a.store = store
			a.recents = store.Recents()
			a.volume = store.Volume()
			a.shuffle = store.Shuffle()
			a.repeat = repeatByName(store.Repeat())
			mode, accent, dynamic := store.Appearance()
			a.appearance = appearance{mode: mode, accent: accent, dynamic: dynamic}
		}
	}
	if a.appearance.mode == "" {
		a.appearance.mode = "system"
	}

	a.engine = player.New(ytdlp.NewResolver())
	a.engine.SetVolume(a.volume)
	a.engine.SetShuffle(a.shuffle)
	a.engine.SetRepeat(a.repeat)

	shell := sidebar.NewShell(sidebar.ShellOptions{
		Title:        "Playlist",
		Width:        1280,
		Height:       840,
		MinWidth:     960,
		MinHeight:    620,
		StateKey:     "playlist",
		Theme:        a.theme,
		Left:         a.sidebar,
		Content:      a.content,
		ContentTitle: a.contentTitle,
		LeftWidth:    236,
		LeftMinWidth: 208,
		LeftMaxWidth: 320,
	})
	a.win = shell.Window()
	a.win.OnClose(func(e *mygo.CloseEvent) {
		// The music goes on: the window hides, and the icon in the
		// notification area brings it back.
		if a.tray == nil {
			return
		}
		e.PreventDefault()
		a.win.Hide()
	})
	a.win.OnClosed(func() {
		a.mu.Lock()
		a.closed = true
		a.mu.Unlock()
		a.rememberSession()
		a.engine.Close()
	})
	a.art.onReady = func() {
		a.update(func() {})
	}
	a.newTray()
	a.registerKeys()
	a.restoreSession()
	go a.listen()
}

// restoreSession opens the playlist that was open when the app last closed,
// from the cache, and puts the player where it left off.
//
// This is the difference between an app and a web page: the playlist is there
// at once, and the track that was playing carries on from where it stopped
// rather than starting the queue again.
func (a *app) restoreSession() {
	restored, ok := a.restoredState()
	if !ok {
		return
	}
	if a.engine != nil {
		a.engine.SetQueue(restored.list.Tracks)
		a.playAt(restored.index, restored.position)
	}
}

// restored is where the last run left the app.
type restored struct {
	link     string
	list     music.Playlist
	index    int
	position time.Duration
}

// restoredState reads the last session and the playlist it names, and puts the
// page in the state the next run should start from. It returns false when
// there is nothing to restore.
//
// It touches no engine and no window, so what the app would open on can be
// checked without either.
func (a *app) restoredState() (restored, bool) {
	if a.store == nil {
		return restored{}, false
	}
	session := a.store.Session()
	if session.Link == "" {
		return restored{}, false
	}
	list, ok := a.store.CachedPlaylist(session.Link)
	if !ok || len(list.Tracks) == 0 {
		// Nothing was kept for it, so the link waits in the field for the
		// user to open it again.
		a.link = session.Link
		return restored{}, false
	}

	a.link = session.Link
	a.playlist = &list
	a.queue = list.Tracks
	a.visible = visibleTracks(list.Tracks, "")
	a.page = pagePlaylist

	// Carry on at the track that played, at the place in it. A track that is
	// no longer in the playlist starts the queue instead.
	index := 0
	for i, track := range list.Tracks {
		if track.ID == session.TrackID {
			index = i
			break
		}
	}
	a.current = index
	a.position = session.Position
	return restored{link: session.Link, list: list, index: index, position: session.Position}, true
}

// rememberSession keeps the playlist that is open and where the player is, so
// the next run carries on from there. It takes the lock itself, for the places
// that are not already holding it.
func (a *app) rememberSession() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.saveSessionLocked()
}

// listen takes what the engine reports and applies it on the thread that
// draws.
func (a *app) listen() {
	for event := range a.engine.Events() {
		a.update(func() {
			a.apply(event)
		})
	}
}

// update runs a function on the thread that draws.
func (a *app) update(fn func()) {
	a.mu.Lock()
	closed := a.closed
	a.mu.Unlock()
	if closed || a.win == nil {
		return
	}
	a.win.Update(fn)
}

// apply takes what the engine reported and puts it in the state.
//
// It holds the app's lock, so it takes what it needs for the tray from the
// state it is writing: asking the app again would lock the lock it holds, and
// the thread that draws would wait on itself. That is why the tray is told
// what plays rather than asked.
func (a *app) apply(event player.Event) {
	a.mu.Lock()
	defer a.mu.Unlock()
	switch event.Kind {
	case player.KindTrack:
		if event.Err == nil {
			if a.store != nil {
				a.store.FinishLastPlay(a.position)
				a.store.AddPlay(library.Play{
					TrackID: event.Track.ID,
					Title:   event.Track.Title,
					Artist:  event.Track.Artist,
					At:      time.Now(),
				})
			}
			a.current = event.Index
			if event.Track.Duration > 0 {
				a.duration = event.Track.Duration
			}
			a.tray.refresh(event.Track, event.State == player.StatePlaying)
			// A new track is where the next run carries on from.
			a.saveSessionLocked()
		}
		a.playing = event.State == player.StatePlaying
	case player.KindState:
		a.playing = event.State == player.StatePlaying
		a.tray.refresh(a.trackLocked(), a.playing)
	case player.KindPosition:
		a.position = event.Position
		if event.Duration > 0 {
			a.duration = event.Duration
		}
		if !a.scrubbing {
			a.seekValue = event.Position.Seconds()
		}
		// The place in the track is written now and then, not on every report:
		// the engine reports five times a second, and a file write that often
		// costs more than the second of accuracy it buys.
		if time.Since(a.sessionAt) > sessionInterval {
			a.saveSessionLocked()
		}
	case player.KindError:
		a.status = event.Err.Error()
		a.notice = &notice{title: "That track will not play", body: event.Err.Error(), kind: "error"}
	}
}

// trackLocked returns the track that plays. The caller holds the app's lock.
func (a *app) trackLocked() music.Track {
	if a.current < 0 || a.current >= len(a.queue) {
		return music.Track{}
	}
	return a.queue[a.current]
}

// saveSessionLocked keeps where the player is. The caller holds the app's
// lock.
//
// It is written as the track plays, now and then, and again when the track
// changes: a second of the place in a track is worth keeping, five writes a
// second is not.
func (a *app) saveSessionLocked() {
	if a.store == nil || a.link == "" {
		return
	}
	track := a.trackLocked()
	a.store.SetSession(library.Session{
		Link:     a.link,
		TrackID:  track.ID,
		Position: a.position,
	})
	a.sessionAt = time.Now()
}

// contentTitle draws what sits in the shell header beside the window
// controls.
//
// The page draws its own title below, so this holds the actions of the page
// rather than repeating it. The header keeps the same inset as a page, so an
// action ends where the page's own column does.
func (a *app) contentTitle(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).FillWidth().Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
		ui.Spacer(c)
		switch a.page {
		case pageHome, pagePlaylist:
			button := components.Button(c, "Paste link", components.ButtonSecondary)
			if button.Clicked() {
				a.pasteLink()
			}
		case pageStats:
			button := components.Button(c, "Clear history", components.ButtonGhost)
			if button.Clicked() {
				a.clearHistory()
			}
		}
	})
}

// frame draws the window the way the app shows it: the shell's layout, the
// page, the shortcuts, and the messages.
//
// The window itself uses Onevra's sidebar shell, which owns the layout. This
// is the same view without a window, so a test draws what the window shows.
func (a *app) frame(c *ui.Context) {
	a.shortcuts(c)
	a.shellView(c)
} // load opens the playlist of the link and starts it.
//
// A playlist that was read before comes from the cache, so opening it again
// costs no network at all; only a link the app has never seen is read from
// YouTube.
func (a *app) load() {
	link := strings.TrimSpace(a.link)
	if link == "" {
		return
	}

	if a.store != nil {
		if list, ok := a.store.CachedPlaylist(link); ok && len(list.Tracks) > 0 {
			a.showPlaylist(link, list)
			return
		}
	}

	a.mu.Lock()
	a.loading = true
	a.status = "Reading the playlist…"
	a.generation++
	generation := a.generation
	a.mu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), loadTimeout)
		defer cancel()
		result, err := playlist.Load(ctx, link)
		a.update(func() {
			a.mu.Lock()
			stale := generation != a.generation
			a.mu.Unlock()
			if stale {
				return // a newer link won
			}
			if err != nil {
				a.mu.Lock()
				a.loading = false
				a.status = err.Error()
				a.notice = &notice{title: "That playlist would not open", body: err.Error(), kind: "error"}
				a.mu.Unlock()
				return
			}
			a.showPlaylist(link, *result)
		})
	}()
}

// showPlaylist puts a playlist on the page and starts it, and keeps it for the
// next time it is opened.
func (a *app) showPlaylist(link string, list music.Playlist) {
	a.mu.Lock()
	a.loading = false
	a.status = ""
	a.playlist = &list
	a.queue = list.Tracks
	a.visible = visibleTracks(list.Tracks, "")
	a.filter = ""
	a.current = 0
	a.position = 0
	a.page = pagePlaylist
	a.mu.Unlock()

	if a.engine != nil {
		a.engine.SetQueue(list.Tracks)
	}
	if a.store != nil {
		a.store.AddRecent(link, list.Title, list.Thumbnail)
		a.store.CachePlaylist(link, list)
		a.recents = a.store.Recents()
	}
	a.rememberSession()
	if len(list.Tracks) > 0 {
		a.play(0)
	}
}

// visibleTracks returns the places of the tracks whose title or artist holds
// the text.
func visibleTracks(tracks []music.Track, filter string) []int {
	filter = strings.ToLower(strings.TrimSpace(filter))
	if filter == "" {
		indexes := make([]int, len(tracks))
		for i := range tracks {
			indexes[i] = i
		}
		return indexes
	}
	var indexes []int
	for i, track := range tracks {
		haystack := strings.ToLower(track.Title + " " + track.Artist + " " + track.Album)
		if strings.Contains(haystack, filter) {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

// repeatByName reads the name of a repeat mode.
func repeatByName(name string) player.Repeat {
	switch name {
	case "all":
		return player.RepeatAll
	case "one":
		return player.RepeatOne
	default:
		return player.RepeatOff
	}
}

// nameOfRepeat is the name of a repeat mode, as it is remembered.
func nameOfRepeat(repeat player.Repeat) string {
	switch repeat {
	case player.RepeatAll:
		return "all"
	case player.RepeatOne:
		return "one"
	default:
		return "off"
	}
}
