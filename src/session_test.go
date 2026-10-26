package src

import (
	"path/filepath"
	"testing"
	"time"

	"youtube-playlist/internal/library"
	"youtube-playlist/internal/music"
	"youtube-playlist/internal/player"
)

// The app remembers two things between runs: the playlists it has read, so
// opening one again needs no network, and where the player was, so a track
// carries on instead of starting the queue again. These tests are about that.

// openStore returns a store in a folder of the test's own.
func openStore(t *testing.T) *library.Store {
	t.Helper()
	store, err := library.Open(filepath.Join(t.TempDir(), "settings.json"))
	if err != nil {
		t.Fatalf("open the store: %v", err)
	}
	return store
}

// samplePlaylist is a playlist of three tracks.
func samplePlaylist() music.Playlist {
	return music.Playlist{
		ID:        "PL",
		Title:     "Mix - Bruno Mars",
		Thumbnail: "https://example.test/cover.jpg",
		Tracks: []music.Track{
			{ID: "one", Title: "It Will Rain", Artist: "Bruno Mars", Duration: 3 * time.Minute},
			{ID: "two", Title: "Roar", Artist: "Katy Perry", Duration: 3 * time.Minute},
			{ID: "three", Title: "Roll Up", Artist: "Wiz Khalifa", Duration: 3 * time.Minute},
		},
	}
}

// TestPlaylistIsCachedAndComesBackWhole checks that a playlist read once is
// kept with its tracks, and comes back the same: a cache that loses the artist
// or the length is worse than none, because the page then looks broken.
func TestPlaylistIsCachedAndComesBackWhole(t *testing.T) {
	store := openStore(t)
	want := samplePlaylist()
	store.CachePlaylist("https://example.test/list", want)

	got, ok := store.CachedPlaylist("https://example.test/list")
	if !ok {
		t.Fatal("the playlist was not cached")
	}
	if got.Title != want.Title || got.Thumbnail != want.Thumbnail {
		t.Errorf("the playlist came back as %q (%q), not %q (%q)",
			got.Title, got.Thumbnail, want.Title, want.Thumbnail)
	}
	if len(got.Tracks) != len(want.Tracks) {
		t.Fatalf("the cache kept %d tracks, not %d", len(got.Tracks), len(want.Tracks))
	}
	for i := range want.Tracks {
		if got.Tracks[i] != want.Tracks[i] {
			t.Errorf("track %d came back as %+v, not %+v", i, got.Tracks[i], want.Tracks[i])
		}
	}
}

// TestCacheSurvivesARestart checks that the cache is on disk, not in memory:
// the point of it is the next run.
func TestCacheSurvivesARestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	store, err := library.Open(path)
	if err != nil {
		t.Fatalf("open the store: %v", err)
	}
	store.CachePlaylist("https://example.test/list", samplePlaylist())

	reopened, err := library.Open(path)
	if err != nil {
		t.Fatalf("reopen the store: %v", err)
	}
	if _, ok := reopened.CachedPlaylist("https://example.test/list"); !ok {
		t.Error("the playlist was not there after a restart")
	}
}

// TestCacheIsKeyedByLink checks that two playlists do not overwrite each other:
// the link is what identifies one.
func TestCacheIsKeyedByLink(t *testing.T) {
	store := openStore(t)
	first := samplePlaylist()
	second := music.Playlist{Title: "Another one", Tracks: []music.Track{{ID: "x", Title: "X"}}}
	store.CachePlaylist("link-a", first)
	store.CachePlaylist("link-b", second)

	if got, ok := store.CachedPlaylist("link-a"); !ok || got.Title != first.Title {
		t.Errorf("link-a came back as %q, not %q", got.Title, first.Title)
	}
	if got, ok := store.CachedPlaylist("link-b"); !ok || got.Title != second.Title {
		t.Errorf("link-b came back as %q, not %q", got.Title, second.Title)
	}
	if _, ok := store.CachedPlaylist("link-c"); ok {
		t.Error("a link that was never read came back from the cache")
	}
}

// TestCacheForgetsTheOldest checks that the cache does not grow without end: a
// playlist past the limit is dropped, and the newest is kept.
func TestCacheForgetsTheOldest(t *testing.T) {
	store := openStore(t)
	for i := range library.MaxCached() + 3 {
		store.CachePlaylist(string(rune('a'+i)), music.Playlist{Title: string(rune('A' + i))})
	}
	if _, ok := store.CachedPlaylist("a"); ok {
		t.Error("the oldest playlist is still cached past the limit")
	}
	last := string(rune('a' + library.MaxCached() + 2))
	if _, ok := store.CachedPlaylist(last); !ok {
		t.Error("the newest playlist was not cached")
	}
}

// TestSessionSurvivesARestart checks that the track and the place in it are
// kept: this is what makes the app carry on rather than start again.
func TestSessionSurvivesARestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	store, err := library.Open(path)
	if err != nil {
		t.Fatalf("open the store: %v", err)
	}
	want := library.Session{Link: "link-a", TrackID: "two", Position: 97 * time.Second}
	store.SetSession(want)

	reopened, err := library.Open(path)
	if err != nil {
		t.Fatalf("reopen the store: %v", err)
	}
	if got := reopened.Session(); got != want {
		t.Errorf("the session came back as %+v, not %+v", got, want)
	}
}

// TestRestorePutsThePlayerWhereItStopped checks the whole point of the session:
// the playlist is on the page, and the track that plays is the one that was
// playing, not the first of the queue.
func TestRestorePutsThePlayerWhereItStopped(t *testing.T) {
	store := openStore(t)
	list := samplePlaylist()
	store.CachePlaylist("link-a", list)
	store.SetSession(library.Session{
		Link:     "link-a",
		TrackID:  "two",
		Position: 42 * time.Second,
	})

	a := &app{art: newArtwork(), current: -1, page: pageHome, store: store}
	a.restoreSession()

	if a.playlist == nil {
		t.Fatal("the playlist was not restored")
	}
	if a.playlist.Title != list.Title {
		t.Errorf("the page shows %q, not %q", a.playlist.Title, list.Title)
	}
	if a.page != pageQueue {
		t.Errorf("the app opened on %q, not the playlist", a.page)
	}
	if len(a.queue) != len(list.Tracks) {
		t.Fatalf("the queue has %d tracks, not %d", len(a.queue), len(list.Tracks))
	}
	if a.current != 1 {
		t.Errorf("the track that plays is %d, not 1 (the one that was playing)", a.current)
	}
	if a.link != "link-a" {
		t.Errorf("the link is %q, not the one that was open", a.link)
	}
}

// TestRestoreWithNoCacheLeavesTheLinkInTheField checks that a session whose
// playlist was not kept does not restore a half state: the link waits in the
// field, and the page is left alone.
func TestRestoreWithNoCacheLeavesTheLinkInTheField(t *testing.T) {
	store := openStore(t)
	store.SetSession(library.Session{Link: "link-gone", TrackID: "two"})

	a := &app{art: newArtwork(), current: -1, page: pageHome, store: store}
	a.restoreSession()

	if a.playlist != nil {
		t.Error("a playlist was restored that was never cached")
	}
	if a.page != pageHome {
		t.Errorf("the app opened on %q, not home", a.page)
	}
	if a.link != "link-gone" {
		t.Errorf("the link is %q: it should wait in the field to be opened again", a.link)
	}
}

// TestRestoreWithNoSessionLeavesTheAppAtHome checks the first run: nothing is
// kept, and the app opens where it should.
func TestRestoreWithNoSessionLeavesTheAppAtHome(t *testing.T) {
	a := &app{art: newArtwork(), current: -1, page: pageHome, store: openStore(t)}
	a.restoreSession()

	if a.playlist != nil || a.page != pageHome {
		t.Errorf("a fresh app restored something: page %q, playlist %v", a.page, a.playlist)
	}
}

// TestRestoreFallsBackToTheFirstTrack checks that a session whose track has
// left the playlist still opens the playlist, from its start, rather than
// playing nothing.
func TestRestoreFallsBackToTheFirstTrack(t *testing.T) {
	store := openStore(t)
	store.CachePlaylist("link-a", samplePlaylist())
	store.SetSession(library.Session{Link: "link-a", TrackID: "gone"})

	a := &app{art: newArtwork(), current: -1, page: pageHome, store: store}
	a.restoreSession()

	if a.playlist == nil {
		t.Fatal("the playlist was not restored")
	}
	if a.current != 0 {
		t.Errorf("the track that plays is %d: a track that left the playlist should fall back to the first",
			a.current)
	}
}

// TestSessionIsWrittenWhenTheTrackChanges checks that the session is kept as
// the app plays, not only at quit: an app that is killed does not get to quit.
func TestSessionIsWrittenWhenTheTrackChanges(t *testing.T) {
	store := openStore(t)
	a := &app{art: newArtwork(), current: -1, link: "link-a", openLink: "link-a", store: store}
	a.queue = samplePlaylist().Tracks
	a.tray = &tray{app: a}

	a.apply(playerTrackEvent(1, a.queue[1]))

	session := store.Session()
	if session.Link != "link-a" {
		t.Errorf("the session kept %q as the link, not link-a", session.Link)
	}
	if session.TrackID != "two" {
		t.Errorf("the session kept %q as the track, not two", session.TrackID)
	}
}

// TestCachedLoadStartsTheQueue checks that opening a cached playlist starts
// playing without going to the network: the queue is set, and the first track
// is asked for.
func TestCachedLoadStartsTheQueue(t *testing.T) {
	store := openStore(t)
	list := samplePlaylist()
	store.CachePlaylist("link-a", list)

	a := &app{art: newArtwork(), current: -1, page: pageHome, store: store, link: "link-a"}
	a.load()

	if a.playlist == nil {
		t.Fatal("the cached playlist was not shown")
	}
	if a.page != pageQueue {
		t.Errorf("the app is on %q, not the playlist", a.page)
	}
	if len(a.queue) != len(list.Tracks) {
		t.Errorf("the queue has %d tracks, not %d", len(a.queue), len(list.Tracks))
	}
	if a.loading {
		t.Error("the app is still loading a playlist it had in the cache")
	}
	// Opening a cached playlist still counts as playing it.
	if len(store.Recents()) == 0 {
		t.Error("the playlist was not remembered as played")
	}
}

// TestThePlaceInTheTrackIsWrittenDown checks that the position is kept as the
// track plays, not only when it starts: an app that remembered the track but
// not the place would always start it from the beginning, which is the bug this
// is here to stop.
func TestThePlaceInTheTrackIsWrittenDown(t *testing.T) {
	store := openStore(t)
	a := &app{art: newArtwork(), current: 0, link: "link-a", openLink: "link-a", store: store}
	a.queue = samplePlaylist().Tracks
	a.tray = &tray{app: a}

	// A report from the engine, a minute into the track.
	a.apply(player.Event{Kind: player.KindPosition, Position: 61 * time.Second})

	session := store.Session()
	if session.Position != 61*time.Second {
		t.Errorf("the place in the track was kept as %v, not 1m1s", session.Position)
	}
	if session.TrackID != "one" {
		t.Errorf("the track was kept as %q, not one", session.TrackID)
	}
}

// TestThePlaceIsNotWrittenFiveTimesASecond checks the throttle: the engine
// reports the place every 200ms, and a file write that often costs more than
// the second of accuracy it buys.
func TestThePlaceIsNotWrittenFiveTimesASecond(t *testing.T) {
	store := openStore(t)
	a := &app{art: newArtwork(), current: 0, link: "link-a", openLink: "link-a", store: store}
	a.queue = samplePlaylist().Tracks
	a.tray = &tray{app: a}

	a.apply(player.Event{Kind: player.KindPosition, Position: 10 * time.Second})
	first := store.Session().Position
	if first != 10*time.Second {
		t.Fatalf("the first report was not written: %v", first)
	}

	// The next few reports come well inside the interval, and must not be
	// written: the session still holds the first place.
	for i := range 4 {
		a.apply(player.Event{Kind: player.KindPosition, Position: time.Duration(11+i) * time.Second})
	}
	if got := store.Session().Position; got != first {
		t.Errorf("the place was rewritten inside the interval: %v, then %v", first, got)
	}
}

// TestThePlaceIsWrittenAgainAfterTheInterval checks that the throttle is a
// delay and not a stop: once the interval has passed, the new place is kept.
func TestThePlaceIsWrittenAgainAfterTheInterval(t *testing.T) {
	store := openStore(t)
	a := &app{art: newArtwork(), current: 0, link: "link-a", openLink: "link-a", store: store}
	a.queue = samplePlaylist().Tracks
	a.tray = &tray{app: a}

	a.apply(player.Event{Kind: player.KindPosition, Position: 10 * time.Second})

	// Pretend the interval has passed, as it would a few seconds later.
	a.mu.Lock()
	a.sessionAt = time.Now().Add(-sessionInterval - time.Second)
	a.mu.Unlock()

	a.apply(player.Event{Kind: player.KindPosition, Position: 40 * time.Second})
	if got := store.Session().Position; got != 40*time.Second {
		t.Errorf("the place was not written after the interval: %v", got)
	}
}

// TestOpeningACachedPlaylistNeverLoads checks the promise of the cache: a
// playlist that was read before is opened from disk, and the network is never
// asked for it.
//
// The guard is that load() returns before it reaches playlist.Load, which is
// the only place that goes out. The test proves it by pointing the app at a
// link that could not resolve, and checking that nothing went out at all.
func TestOpeningACachedPlaylistNeverLoads(t *testing.T) {
	store := openStore(t)
	list := samplePlaylist()
	store.CachePlaylist("link-a", list)

	a := &app{art: newArtwork(), current: -1, page: pageHome, store: store, link: "link-a"}
	a.load()

	// A cached open is complete on the spot: no generation was bumped, which
	// is what load() does only when it is about to go to the network.
	if a.generation != 0 {
		t.Errorf("opening a cached playlist started a load (generation %d): it went out to the network",
			a.generation)
	}
	if a.status != "" {
		t.Errorf("a cached open left a status of %q", a.status)
	}
	if a.playlist == nil || a.playlist.Title != list.Title {
		t.Error("the cached playlist was not shown")
	}
}

// TestRestoringNeverLoads checks the same promise for the session: the app
// opens on the playlist it had, without reading it again.
func TestRestoringNeverLoads(t *testing.T) {
	store := openStore(t)
	store.CachePlaylist("link-a", samplePlaylist())
	store.SetSession(library.Session{Link: "link-a", TrackID: "one"})

	a := &app{art: newArtwork(), current: -1, page: pageHome, store: store}
	a.restoreSession()

	if a.generation != 0 {
		t.Errorf("restoring started a load (generation %d): it went out to the network", a.generation)
	}
	if a.playlist == nil {
		t.Error("restoring did not open the playlist")
	}
}

// TestAPlaylistTooLargeIsNotCached checks the size limit: a mix of a thousand
// tracks is not kept, because the file is read and written on every position
// report, and a playlist that big would make that slow.
func TestAPlaylistTooLargeIsNotCached(t *testing.T) {
	store := openStore(t)
	var tracks []music.Track
	for i := range 40000 {
		tracks = append(tracks, music.Track{
			ID:    "id" + itoa(i),
			Title: "A track with a name long enough to add up",
		})
	}
	store.CachePlaylist("huge", music.Playlist{Title: "Huge", Tracks: tracks})
	if _, ok := store.CachedPlaylist("huge"); ok {
		t.Error("a playlist past the size limit was cached")
	}
}

// TestASmallPlaylistIsStillCached checks that the size limit does not stop the
// ordinary case: a normal playlist is well under it.
func TestASmallPlaylistIsStillCached(t *testing.T) {
	store := openStore(t)
	store.CachePlaylist("small", samplePlaylist())
	if _, ok := store.CachedPlaylist("small"); !ok {
		t.Error("an ordinary playlist was not cached")
	}
}

// itoa writes a number.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// playerTrackEvent is what the engine reports when a track starts.
func playerTrackEvent(index int, track music.Track) player.Event {
	return player.Event{Kind: player.KindTrack, Index: index, Track: track, State: player.StatePlaying}
}
