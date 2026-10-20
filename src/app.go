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
		a.engine.Close()
	})
	a.art.onReady = func() {
		a.update(func() {})
	}
	a.newTray()
	a.registerKeys()
	go a.listen()
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
} // load reads the playlist of the link and starts it.
func (a *app) load() {
	link := strings.TrimSpace(a.link)
	if link == "" {
		return
	}
	a.loading = true
	a.status = "Reading the playlist…"
	a.generation++
	generation := a.generation

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), loadTimeout)
		defer cancel()
		result, err := playlist.Load(ctx, link)
		a.update(func() {
			a.mu.Lock()
			defer a.mu.Unlock()
			if generation != a.generation {
				return // a newer link won
			}
			a.loading = false
			if err != nil {
				a.status = err.Error()
				a.notice = &notice{title: "That playlist would not open", body: err.Error(), kind: "error"}
				return
			}
			a.status = ""
			a.playlist = result
			a.queue = result.Tracks
			a.visible = visibleTracks(result.Tracks, "")
			a.filter = ""
			a.engine.SetQueue(result.Tracks)
			if a.store != nil {
				a.store.AddRecent(link, result.Title, result.Thumbnail)
				a.recents = a.store.Recents()
			}
			a.page = pagePlaylist
			if len(result.Tracks) > 0 {
				a.engine.Play(0)
			}
		})
	}()
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
