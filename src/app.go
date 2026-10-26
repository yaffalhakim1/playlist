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

	"youtube-playlist/internal/innertube"
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

// pendingDelete is a playlist the user asked to delete, waiting for the
// confirmation that cannot be taken back.
type pendingDelete struct {
	link  string
	title string
	open  bool
}

// app is the whole interface: the state it shows and the actions it takes.
type app struct {
	mu sync.Mutex

	page     string
	link     string
	// filter narrows the playlist's rows, as its own field types.
	filter   string
	playlist *music.Playlist
	queue    []music.Track
	visible  []int
	status   string
	loading  bool
	// loadingWhat says what is being read, so the page can say it too.
	loadingWhat string
	// openLink is the address of the playlist that is open, which is not the
	// one in the field: the field is cleared once a playlist opens.
	openLink string

	current   int
	position  time.Duration
	duration  time.Duration
	playing   bool
	buffering bool
	volume    float64
	shuffle   bool
	repeat    player.Repeat
	// seekValue and scrubbing are the slider's own state: while the pointer
	// holds the knob the engine's place must not move it back.
	seekValue float64
	scrubbing bool
	scrubAt   time.Time
	// sessionAt is when the place in the track was last written down.
	sessionAt time.Time

	// switchDynamic, switchDiscord and switchSolid are what the Appearance
	// page's switches are bound to. A switch flips the value it is given as it is
	// built, so the value has to outlive the frame: these are seeded from the real
	// state as the page is drawn, and read back after it.
	switchDynamic bool
	switchDiscord bool
	switchSolid   bool

	// deleting is the playlist waiting to be deleted, if any.
	deleting pendingDelete

	// followedTrack is the track the Up next list was last scrolled to, so it
	// follows the music without fighting the reader's own scrolling.
	followedTrack int

	// presence shows what plays in Discord, when it is turned on.
	presence *presence

	// scrobble sends what plays to Last.fm, when an account is connected.
	//
	// There is no longer a page to set it up from: the account and the
	// application's key are read from the settings at startup, and scrobbling
	// runs quietly in the background for an install that already has them.
	scrobble *scrobbler
	// chosenTrack says the next track to play was picked by the user, which
	// Last.fm records as chosenByUser.
	chosenTrack bool

	recents []library.Recent
	art     *artwork

	engine     *player.Engine
	win        *mygo.Window
	store      *library.Store
	notice     *notice
	generation int
	tray       *tray

	appearance appearance

	list          ui.ListState
	queueL        ui.ListState
	sidebarPixels float32
	// railClosed says the sidebar is hidden, which is a choice the user makes
	// with the header's button and the app keeps for the run.
	railClosed bool
	closed     bool
}

// appearance holds what the user chose about the look of the app.
type appearance struct {
	// mode is "system", "light" or "dark".
	mode string
	// accent is the colour of the accent, as a hex string, empty for the
	// paper's own ink.
	accent string
	// dynamic takes the accent from the artwork of the track that plays.
	dynamic bool
	// material is the window's backdrop: "", "tabbed", "mica" or "acrylic".
	material string
	// solid says the window draws its own opaque face instead of the material.
	solid bool
	// discord shows what plays in Discord.
	discord bool
}

// materialOf returns the backdrop a window is created with.
//
// A window has to be born with a material to be able to change it later, so one is
// always asked for unless the user has turned transparency off. The view paints
// its own face over it either way, so an unsupported desktop loses nothing.
func materialOf(a appearance) mygo.Vibrancy {
	if a.solid {
		return mygo.VibrancyNone
	}
	if a.material == "" {
		return mygo.VibrancyMica
	}
	return mygo.Vibrancy(a.material)
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
			mode, accent, dynamic, _ := store.Appearance()
			material, solid := store.Material()
			a.appearance = appearance{
				mode:     mode,
				accent:   accent,
				dynamic:  dynamic,
				material: material,
				solid:    solid,
			}
			a.appearance.discord = store.Discord()
		}
	}
	if a.appearance.mode == "" {
		a.appearance.mode = "system"
	}

	// Discord presence is off unless the user turned it on, and off entirely
	// when the build has no application id to register it under.
	a.presence = &presence{}
	if a.appearance.discord && discordClientID != "" {
		a.presence.enable(true)
	}

	// Last.fm: the application's key and the account are the user's, and are
	// read from the settings so a build carries neither. There is no page to
	// enter them on any more, so an install without them simply does not
	// scrobble.
	a.scrobble = newScrobbler()
	if a.store != nil {
		key, secret, user, paused := a.store.LastFM()
		if path, err := mygo.App.Path(mygo.PathUserData); err == nil {
			a.scrobble.open(path)
			a.scrobble.configure(key, secret)
			a.scrobble.restore(user, paused)
		}
	}
	a.scrobble.onSave = func(user string, paused bool) {
		if a.store != nil {
			a.store.SetLastFMAccount(user, paused)
		}
	}

	a.engine = player.New(ytdlp.NewResolver())
	a.engine.SetVolume(a.volume)
	a.engine.SetShuffle(a.shuffle)
	a.engine.SetRepeat(a.repeat)

	a.win = mygo.NewWindow(mygo.WindowOptions{
		Title:         "Playlist",
		Width:         1280,
		Height:        840,
		MinWidth:      960,
		MinHeight:     620,
		StateKey:      "playlist",
		TitleBarStyle: mygo.TitleBarHidden,
		// The window is created with its material, because on Windows a window
		// created without one cannot take one later: `SetVibrancy` changes the
		// material of a window that has one, and shows none in the others.
		Vibrancy: materialOf(a.appearance),
		Content:  ui.View(a.frame),
	})
	// The window owns the taskbar icon: set it here rather than trusting the
	// executable's resources, which a dev build does not have.
	if err := a.win.SetIcon(appIcon); err != nil {
		log.Printf("icon: the taskbar keeps the executable's: %v", err)
	}
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
		if a.presence != nil {
			a.presence.close()
		}
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
	a.openLink = session.Link
	a.playlist = &list
	a.queue = list.Tracks
	a.visible = allTracks(list.Tracks)
	a.page = pageQueue

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
	if closed {
		return
	}
	if a.win == nil {
		// There is no window to draw on, as in a test: the state is still the
		// app's to change, so it is changed here rather than dropped.
		fn()
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
			// Discord counts from the start of the track, so a new track resets
			// the clock before it is shown.
			if a.presence != nil {
				a.presence.resetTrack()
			}
			// Last.fm follows the play from its first sample. A track the queue
			// moved on to was not chosen; one the user asked for was.
			if a.scrobble != nil {
				a.scrobble.begin(event.Track, a.chosenTrack)
				a.chosenTrack = false
			}
		}
		a.setPlayState(event.State)
		if a.presence != nil {
			a.presence.show(a.trackLocked(), a.playing)
		}
	case player.KindState:
		a.setPlayState(event.State)
		a.tray.refresh(a.trackLocked(), a.playing)
		if a.presence != nil {
			a.presence.show(a.trackLocked(), a.playing)
		}
		// A held track is not being heard, so the time does not count towards
		// Last.fm's threshold.
		if a.scrobble != nil && !a.playing {
			a.scrobble.breakListen()
		}
	case player.KindPosition:
		a.position = event.Position
		if event.Duration > 0 {
			a.duration = event.Duration
		}
		if !a.scrubbing {
			a.seekValue = event.Position.Seconds()
		}
		// Last.fm counts what was heard from these positions, so a pause or a
		// seek forward cannot make a track qualify.
		if a.scrobble != nil {
			if a.scrubbing {
				a.scrobble.breakListen()
			} else {
				a.scrobble.observe(time.Now(), event.Position, a.duration)
			}
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

// setPlayState records what the playback is doing. The caller holds the lock.
//
// The two flags say different things and the page draws both: a track that is
// held is not a track that has not started, and a track whose stream is being
// found is neither.
func (a *app) setPlayState(state player.State) {
	a.playing = state == player.StatePlaying
	a.buffering = state == player.StateBuffering
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
	if a.store == nil || a.openLink == "" {
		return
	}
	track := a.trackLocked()
	a.store.SetSession(library.Session{
		Link:     a.openLink,
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
		case pageHome, pageQueue:
			if b := plainButton(c, "Paste link"); b.Clicked() {
				a.pasteLink()
			}
		case pageStats:
			if b := quietButton(c, "Clear history"); b.Clicked() {
				a.clearHistory()
			}
		}
	})
}

// load opens the playlist of the link and starts it.
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
	a.loadingWhat = "playlist"
	// The spinner says a load is running, so the status line is cleared: it is
	// for what went wrong, and the two would otherwise both speak.
	a.status = ""
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

// playMix opens the mix YouTube generates from the track that is playing.
//
// This is the one mix the app can offer without an account: a mix is built
// around a track, so a video id is all it needs. The list YouTube builds for it
// is read by yt-dlp, which is the only reader that can see it.
func (a *app) playMix() {
	a.mu.Lock()
	track := a.trackLocked()
	a.mu.Unlock()
	if track.ID == "" {
		a.mu.Lock()
		a.notice = &notice{
			title: "Nothing is playing",
			body:  "Play a track, then start a mix from it.",
			kind:  "info",
		}
		a.mu.Unlock()
		return
	}

	link := innertube.MixLink(track.ID)
	a.mu.Lock()
	a.link = link
	a.mu.Unlock()
	a.load()
}

// showPlaylist puts a playlist on the page and starts it, and keeps it for the
// next time it is opened.
func (a *app) showPlaylist(link string, list music.Playlist) {
	a.mu.Lock()
	a.loading = false
	a.loadingWhat = ""
	a.status = ""
	a.playlist = &list
	a.queue = list.Tracks
	a.visible = allTracks(list.Tracks)
	a.filter = ""
	a.current = 0
	a.position = 0
	a.page = pageQueue
	// The form did its job, so it is cleared: the link is in the sidebar now,
	// and leaving it in the field invites pasting it again.
	a.link = ""
	a.openLink = link
	a.mu.Unlock()

	if a.engine != nil {
		a.engine.SetQueue(list.Tracks)
	}
	if a.store != nil {
		a.store.AddRecent(link, list.Title, list.Thumbnail)
		a.store.CachePlaylist(link, list)
		a.recents = a.store.Recents()
	}
	// The session is written with the link the playlist was read from, which
	// the field no longer holds.
	a.rememberSessionFor(link)
	if len(list.Tracks) > 0 {
		a.play(0)
	}
}

// rememberSessionFor keeps the session for a link, which is not necessarily the
// one in the field: the field is cleared once a playlist opens.
func (a *app) rememberSessionFor(link string) {
	if a.store == nil || link == "" {
		return
	}
	a.mu.Lock()
	track := a.trackLocked()
	position := a.position
	a.mu.Unlock()
	a.store.SetSession(library.Session{Link: link, TrackID: track.ID, Position: position})
}


// visibleTracks returns the places of the tracks whose title or artist holds
// the text.
// refilter recomputes which rows the filter lets through.
//
// It runs when the field changes rather than on every frame: the queue is a slice
// of titles already in memory, but a thousand rows filtered sixty times a second
// is work nobody asked for.
func (a *app) refilter() {
	a.mu.Lock()
	queue := a.queue
	filter := a.filter
	a.visible = visibleTracks(queue, filter)
	a.mu.Unlock()
}

// visibleTracks returns the places of the tracks the filter lets through.
//
// A track matches on its title, its artist or its album, because a reader who
// types "nujabes" means the artist and one who types "metaphorical" means the
// album, and the field cannot tell them apart.
func visibleTracks(tracks []music.Track, filter string) []int {
	filter = strings.ToLower(strings.TrimSpace(filter))
	if filter == "" {
		return allTracks(tracks)
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

// resync reads the open playlist from YouTube again, ignoring the cache.
//
// It is the one action that spends the network on a playlist the app already has,
// so it says what it is doing: the page keeps its tracks while the read runs, and
// only replaces them when the answer arrives.
func (a *app) resync() {
	a.mu.Lock()
	link := a.openLink
	a.mu.Unlock()
	if link == "" {
		return
	}
	if a.store != nil {
		a.store.ForgetCache(link)
	}
	a.mu.Lock()
	a.link = link
	a.mu.Unlock()
	a.load()
}

// allTracks returns the places of every track of a queue, in order.
//
// It is the one shape the pages draw: a row is a place in the queue, so the list
// and the table agree about what row 3 is.
func allTracks(tracks []music.Track) []int {
	indexes := make([]int, len(tracks))
	for i := range tracks {
		indexes[i] = i
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
