package src

import (
	"context"
	"errors"
	"log"
	"path/filepath"
	"sync"
	"time"

	"youtube-playlist/internal/lastfm"
	"youtube-playlist/internal/music"
)

// Last.fm scrobbling: what is playing goes to the profile, and a play is added
// once enough of it has been heard.
//
// The rules of the service live in internal/lastfm: half the track or four
// minutes, whichever comes first, measured from the positions the player
// reports so that a pause or a seek does not count. This file connects that to
// the app's state, and keeps the plays that could not be delivered.
const (
	// connectTimeout is how long the user has to approve the app on Last.fm.
	connectTimeout = 10 * time.Minute
	// approvalPoll is how often an unapproved token is exchanged again.
	approvalPoll = 3 * time.Second
	// callTimeout bounds one request to Last.fm.
	callTimeout = 30 * time.Second
	// flushAttempts is how many times in a row a delivery that fails for a
	// reason that may pass is tried, doubling the wait between tries.
	flushAttempts = 4
	// flushBackoff is the first wait between those tries.
	flushBackoff = 15 * time.Second
)

// scrobbler is the app's Last.fm state: the client, the account, the play being
// followed, and the plays waiting to be delivered.
//
// The client and the outbox lock themselves; the rest belongs to the thread
// that draws, like the player's state.
type scrobbler struct {
	mu sync.Mutex

	client *lastfm.Client
	// session is the account's key, empty until it is read from the store.
	session string
	// user is the account's name, from the settings.
	user string
	// paused says scrobbling is off without being disconnected.
	paused bool
	// reconnect says Last.fm refused the session, so nothing is sent until the
	// user connects again.
	reconnect bool
	// busy says a connection is in progress.
	busy bool
	// notice is why the last attempt to connect did not work.
	notice string

	// listen follows the play of the track that is playing, nil when there is
	// nothing to scrobble.
	listen *lastfm.Listen
	// shown is the track whose now-playing line was sent, so it is sent once.
	shown string

	// store keeps the session key, apart from the settings.
	store lastfm.SessionStore
	// outbox keeps the plays that qualified and have not been delivered.
	outbox *lastfm.Outbox

	// flushing guards against two deliveries at once.
	flushing bool
	// cancel stops the connection wait.
	cancel context.CancelFunc
	// onSave is told the account name and the pause choice, so the app can keep
	// them in the settings. It is set once, when the app starts.
	onSave func(user string, paused bool)
}

// newScrobbler returns a scrobbler that has not read anything from disk yet.
func newScrobbler() *scrobbler {
	return &scrobbler{outbox: lastfm.OpenOutbox("")}
}

// openScrobbler points the scrobbler at the files it keeps beside the settings.
func (s *scrobbler) open(dir string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store = lastfm.SessionStore{Path: filepath.Join(dir, "lastfm-session")}
	s.outbox = lastfm.OpenOutbox(filepath.Join(dir, "lastfm-pending.json"))
}

// configure sets the application's key and secret, which are entered in the
// Look page because a release build has none of its own.
//
// A client that is already pointed somewhere else, as a test points one at a
// stub, keeps pointing there: only a fresh client takes the real endpoint.
func (s *scrobbler) configure(key, secret string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	root := ""
	if s.client != nil {
		root = s.client.Root
	}
	s.client = lastfm.New(key, secret)
	s.client.Root = root
}

// configured reports whether an application key has been entered.
func (s *scrobbler) configured() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.client.Configured()
}

// status is what the Look page draws: whether an account is connected, who it
// is, and what is wrong if anything.
type lastfmStatus struct {
	Configured bool
	User       string
	Paused     bool
	Reconnect  bool
	Busy       bool
	Notice     string
	Pending    int
}

// statusNow returns the state the page shows.
func (s *scrobbler) statusNow() lastfmStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	pending := 0
	if s.outbox != nil {
		pending = s.outbox.Len()
	}
	return lastfmStatus{
		Configured: s.client.Configured(),
		User:       s.user,
		Paused:     s.paused,
		Reconnect:  s.reconnect,
		Busy:       s.busy,
		Notice:     s.notice,
		Pending:    pending,
	}
}

// scrobblingNow reports whether plays are being sent: an account is connected
// and scrobbling is not paused.
func (s *scrobbler) scrobblingNow() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.user != "" && !s.paused && s.session != ""
}

// begin starts following a track. It is called when a track starts playing,
// because entering a track says nothing about whether it plays.
func (s *scrobbler) begin(track music.Track, chosen bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.user == "" || s.paused {
		s.listen = nil
		return
	}
	s.listen = lastfm.NewListen(scrobbleTrack(track), chosen)
	s.shown = ""
}

// breakListen says the sound stopped flowing, as it does while the player is
// held or moving to another place. The time in between is not counted.
func (s *scrobbler) breakListen() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listen != nil {
		s.listen.Break()
	}
}

// stopListening forgets the play being followed, without sending anything.
func (s *scrobbler) stopListening() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listen = nil
	s.shown = ""
}

// observe takes a sample of the track playing and acts on what it makes true.
//
// It runs on every position report, which the engine sends five times a second.
// Nothing here waits on the network: a call is started on its own goroutine.
func (s *scrobbler) observe(at time.Time, position, total time.Duration) {
	s.mu.Lock()
	listen := s.listen
	if listen == nil {
		s.mu.Unlock()
		return
	}
	event := listen.Observe(at, position, total)
	// The first sample confirms playback, which is when the now-playing line
	// goes out, once.
	sendNowPlaying := false
	if event == lastfm.Started {
		if key := listen.Track.Title + "|" + listen.Track.Artist; key != s.shown {
			s.shown = key
			sendNowPlaying = true
		}
	}
	var play lastfm.Scrobble
	if event == lastfm.Qualified {
		play = listen.Scrobble(total)
	}
	track := listen.Track
	outbox := s.outbox
	session := s.session
	client := s.client
	canSend := s.user != "" && !s.paused && !s.reconnect && s.session != ""
	s.mu.Unlock()

	if sendNowPlaying && canSend {
		go s.sendNowPlaying(client, session, track)
	}
	if event == lastfm.Qualified {
		// The play is kept first, then sent: a crash between the two loses
		// nothing, because the outbox is on disk.
		outbox.Add(play)
		if canSend {
			go s.flush()
		}
	}
}

// sendNowPlaying tells Last.fm what is playing. A failure costs only the line on
// the profile, so only a refused session is acted on.
func (s *scrobbler) sendNowPlaying(client *lastfm.Client, session string, track lastfm.Track) {
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()
	if err := client.UpdateNowPlaying(ctx, session, track); err != nil {
		log.Printf("last.fm: now playing: %v", err)
		s.failed(err)
	}
}

// flush delivers the plays waiting, oldest first, in batches.
func (s *scrobbler) flush() {
	s.mu.Lock()
	if s.flushing || s.outbox == nil {
		s.mu.Unlock()
		return
	}
	s.flushing = true
	client, session, outbox := s.client, s.session, s.outbox
	// Without a session key there is nothing to sign a scrobble with, and the
	// plays wait on disk until one is connected.
	if session == "" {
		s.flushing = false
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.flushing = false
		s.mu.Unlock()
	}()

	for {
		batch := outbox.Batch(time.Now())
		if len(batch) == 0 {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
		err := client.Scrobble(ctx, session, batch)
		cancel()
		switch {
		case err == nil:
			outbox.Done(len(batch))
		case lastfm.InvalidSession(err):
			s.failed(err)
			return
		case lastfm.Transient(err):
			// It stays in the outbox for the next try.
			log.Printf("last.fm: scrobbling will be tried again: %v", err)
			return
		case len(batch) > 1:
			// One play in the batch is bad. Send them one at a time, so it is
			// dropped rather than holding the rest back.
			if err := s.oneByOne(client, session, outbox, batch); err != nil {
				return
			}
		default:
			// A single play Last.fm will never take: drop it, rather than
			// retry it forever.
			log.Printf("last.fm: dropping %q by %q: %v",
				batch[0].Track.Title, batch[0].Track.Artist, err)
			outbox.Done(1)
		}
	}
}

// oneByOne sends a batch play by play, dropping the ones that can never be
// taken. It stops on a failure worth retrying.
func (s *scrobbler) oneByOne(client *lastfm.Client, session string, outbox *lastfm.Outbox, batch []lastfm.Scrobble) error {
	for _, play := range batch {
		ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
		err := client.Scrobble(ctx, session, []lastfm.Scrobble{play})
		cancel()
		switch {
		case err == nil:
			outbox.Done(1)
		case lastfm.Transient(err):
			return err
		case lastfm.InvalidSession(err):
			s.failed(err)
			return err
		default:
			log.Printf("last.fm: dropping %q by %q: %v", play.Track.Title, play.Track.Artist, err)
			outbox.Done(1)
		}
	}
	return nil
}

// failed acts on a failure that says the session is no good.
func (s *scrobbler) failed(err error) {
	if !lastfm.InvalidSession(err) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.reconnect || s.user == "" {
		return
	}
	s.reconnect = true
	s.notice = "Last.fm needs you to connect again."
}

// verify checks the application's key and secret against Last.fm, which it does
// by asking for a token: the one call that needs the two and nothing else.
func (s *scrobbler) verify() error {
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if !client.Configured() {
		return errors.New("no key was entered")
	}
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()
	_, err := client.Token(ctx)
	return err
}

// connect starts the authorization: a token is asked for, Last.fm's page for it
// opens in the browser, and the token becomes a session once the user approves
// it there. The user's password is never asked for.
func (s *scrobbler) connect(openURL func(string) error) {
	s.mu.Lock()
	if s.busy || !s.client.Configured() {
		s.mu.Unlock()
		return
	}
	s.busy, s.notice = true, ""
	client := s.client
	s.mu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
		token, err := client.Token(ctx)
		cancel()
		if err != nil {
			s.finishConnect(err, lastfm.Session{})
			return
		}
		if err := openURL(client.AuthURL(token)); err != nil {
			s.finishConnect(err, lastfm.Session{})
			return
		}
		s.awaitApproval(client, token)
	}()
}

// awaitApproval exchanges the token for a session until the user has approved
// it, or the time is up.
func (s *scrobbler) awaitApproval(client *lastfm.Client, token string) {
	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	s.mu.Lock()
	s.cancel = cancel
	s.mu.Unlock()
	defer cancel()

	for {
		session, err := client.Session(ctx, token)
		if err == nil {
			s.finishConnect(nil, session)
			return
		}
		if !lastfm.NotApproved(err) && !lastfm.Transient(err) {
			s.finishConnect(err, lastfm.Session{})
			return
		}
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.Canceled) {
				s.finishConnect(ctx.Err(), lastfm.Session{})
				return
			}
			s.finishConnect(errors.New("Last.fm was not approved in time. Try connecting again."), lastfm.Session{})
			return
		case <-time.After(approvalPoll):
		}
	}
}

// finishConnect keeps the session the user approved.
func (s *scrobbler) finishConnect(err error, session lastfm.Session) {
	s.mu.Lock()
	// A cancelled wait was the user's choice, not a failure worth reporting.
	cancelled := errors.Is(err, context.Canceled)
	s.busy = false
	s.cancel = nil
	if err != nil {
		if !cancelled {
			s.notice = "Last.fm did not connect: " + err.Error()
		}
		s.mu.Unlock()
		return
	}
	store, outbox := s.store, s.outbox
	changed := s.user != "" && s.user != session.Name
	s.session, s.user, s.reconnect, s.notice = session.Key, session.Name, false, ""
	s.mu.Unlock()

	if changed {
		// Plays waiting for another account must not be sent as this one.
		outbox.Clear()
	}
	if err := store.Save(session.Key); err != nil {
		log.Printf("last.fm: saving the session: %v", err)
		s.mu.Lock()
		s.notice = "Connected, but the session could not be kept, so it will be lost on the next run."
		s.mu.Unlock()
	}
	s.rememberAccount()
	go s.flush()
}

// disconnect forgets the account: the key leaves the disk, and the plays that
// were waiting for it are dropped.
func (s *scrobbler) disconnect() {
	s.mu.Lock()
	cancel := s.cancel
	outbox := s.outbox
	s.session, s.user, s.reconnect, s.busy, s.notice = "", "", false, false, ""
	s.listen, s.shown = nil, ""
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	outbox.Clear()
	_ = s.store.Delete()
	s.rememberAccount()
}

// cancelConnect gives up waiting for the user's approval.
func (s *scrobbler) cancelConnect() {
	s.mu.Lock()
	cancel := s.cancel
	s.busy, s.cancel = false, nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// setPaused stops or resumes scrobbling without disconnecting.
func (s *scrobbler) setPaused(paused bool) {
	s.mu.Lock()
	s.paused = paused
	// The play in progress was not followed from its start, so it is left out.
	s.listen, s.shown = nil, ""
	s.mu.Unlock()
	s.rememberPaused()
}

// rememberPaused keeps the pause choice in the settings.
func (s *scrobbler) rememberPaused() {
	s.mu.Lock()
	user, paused, save := s.user, s.paused, s.onSave
	s.mu.Unlock()
	if save != nil {
		save(user, paused)
	}
}

// rememberAccount keeps the account name in the settings.
func (s *scrobbler) rememberAccount() {
	s.mu.Lock()
	user, paused, save := s.user, s.paused, s.onSave
	s.mu.Unlock()
	if save != nil {
		save(user, paused)
	}
}

// restore reads the session key from disk and the account from the settings.
func (s *scrobbler) restore(user string, paused bool) {
	s.mu.Lock()
	s.user, s.paused = user, paused
	store := s.store
	s.mu.Unlock()
	if user == "" {
		return
	}
	key, err := store.Load()
	if err != nil {
		log.Printf("last.fm: reading the saved session: %v", err)
		s.mu.Lock()
		s.reconnect = true
		s.mu.Unlock()
		return
	}
	if key == "" {
		// The settings say connected, but the key is gone.
		s.mu.Lock()
		s.reconnect = true
		s.mu.Unlock()
		return
	}
	s.mu.Lock()
	s.session = key
	s.mu.Unlock()
	go s.flush()
}

// close stops waiting for an approval and lets the outbox finish writing.
func (s *scrobbler) close() {
	s.mu.Lock()
	cancel := s.cancel
	outbox := s.outbox
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if outbox != nil {
		outbox.Wait()
	}
}

// scrobbleTrack is what a track says to Last.fm.
func scrobbleTrack(track music.Track) lastfm.Track {
	return lastfm.Track{
		Artist:   track.Artist,
		Title:    track.Title,
		Album:    track.Album,
		Duration: track.Duration,
	}
}
