// Package player plays a queue of tracks: it resolves the audio of a track,
// decodes it and hands it to MyGo's audio package.
//
// The engine owns the queue and the state of the playback. It never touches
// the user interface: it reports what happens on a channel, and the interface
// reads that channel.
package player

import (
	"context"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/egoist/mygo/audio"

	"youtube-playlist/internal/ffmpeg"
	"youtube-playlist/internal/music"
	"youtube-playlist/internal/ytdlp"
)

// State is what the playback is doing.
type State int

const (
	// StateIdle is nothing playing.
	StateIdle State = iota
	// StateBuffering is the wait for the audio of a track.
	StateBuffering
	// StatePlaying is a track playing.
	StatePlaying
	// StatePaused is a track held.
	StatePaused
	// StateFailed is a track that could not play.
	StateFailed
)

// Repeat says what happens at the end of the queue.
type Repeat int

const (
	// RepeatOff stops at the end of the queue.
	RepeatOff Repeat = iota
	// RepeatAll starts the queue again.
	RepeatAll
	// RepeatOne plays the same track again.
	RepeatOne
)

// Kind is the sort of an event.
type Kind int

const (
	// KindTrack is a new track, ready or failed.
	KindTrack Kind = iota
	// KindState is a change of the state of the playback.
	KindState
	// KindPosition is the place in the track, as it advances.
	KindPosition
	// KindError is a failure the interface should show.
	KindError
)

// Event is something the engine reports.
type Event struct {
	Kind     Kind
	Index    int
	Track    music.Track
	State    State
	Position time.Duration
	Duration time.Duration
	Err      error
}

// tick is how often the engine reports the place in the track.
const tick = 200 * time.Millisecond

// minPlayTime is how long a track plays before the engine may take it for
// ended. Without it, a machine with no audio device would skip the queue.
const minPlayTime = 2 * time.Second

// exhaustible is a decoder that can say when its stream has stopped, whether
// it ended or failed.
//
// MyGo's audio player wraps the decoder and only reports that it stopped
// playing, which is also true while a stream waits on the network. The engine
// needs the difference, so it keeps the decoder itself as well.
type exhaustible interface {
	Exhausted() bool
}

// Engine plays a queue of tracks.
type Engine struct {
	resolver *ytdlp.Resolver

	mu         sync.Mutex
	queue      []music.Track
	order      []int
	cursor     int
	current    *audio.Player
	decoder    exhaustible
	state      State
	volume     float64
	shuffle    bool
	repeat     Repeat
	loading    bool
	generation int
	startedAt  time.Time
	failure    error
	// lastPlace and lastMoved track the place in the track between ticks, to
	// tell a stream that has stopped from one that is still playing.
	lastPlace time.Duration
	lastMoved time.Time
	// recoveries counts how many times in a row the stream of the track at
	// recoveryFor has been found again, and recoveredAt is where it was found.
	// A stream that plays on from there clears the count.
	recoveries  int
	recoveryFor int
	recoveredAt time.Duration

	events chan Event
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

// New returns an engine that resolves streams with the given resolver.
func New(resolver *ytdlp.Resolver) *Engine {
	ctx, cancel := context.WithCancel(context.Background())
	e := &Engine{
		resolver: resolver,
		order:    []int{},
		volume:   0.8,
		events:   make(chan Event, 64),
		ctx:      ctx,
		cancel:   cancel,
		done:     make(chan struct{}),
	}
	go e.run()
	return e
}

// Events returns the channel the engine reports on.
func (e *Engine) Events() <-chan Event { return e.events }

// Close stops the playback and the engine.
func (e *Engine) Close() {
	e.cancel()
	close(e.done)
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.current != nil {
		e.current.Close()
		e.current = nil
	}
	e.decoder = nil
}

// SetQueue replaces the queue of tracks and stops what plays.
func (e *Engine) SetQueue(tracks []music.Track) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stopLocked()
	e.queue = tracks
	e.order = e.order[:0]
	for i := range tracks {
		e.order = append(e.order, i)
	}
	e.cursor = 0
	if e.shuffle {
		e.shuffleLocked()
	}
	e.state = StateIdle
}

// Queue returns the tracks of the queue.
func (e *Engine) Queue() []music.Track {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.queue
}

// Current returns the index of the track that plays, or -1.
func (e *Engine) Current() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.state == StateIdle || len(e.order) == 0 {
		return -1
	}
	return e.order[e.cursor]
}

// State returns what the playback is doing.
func (e *Engine) State() State {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.state
}

// Volume returns the volume, from 0 to 1.
func (e *Engine) Volume() float64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.volume
}

// SetVolume sets the volume, from 0 to 1.
func (e *Engine) SetVolume(v float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.volume = clampVolume(v)
	if e.current != nil {
		e.current.SetVolume(e.volume)
	}
}

// Shuffle tells whether the queue plays in a random order.
func (e *Engine) Shuffle() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.shuffle
}

// SetShuffle turns the random order on or off, keeping the track that plays.
func (e *Engine) SetShuffle(on bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.shuffle = on
	if on {
		e.shuffleLocked()
		return
	}
	e.order = e.order[:0]
	for i := range e.queue {
		e.order = append(e.order, i)
	}
	e.cursor = e.indexOf(e.CurrentLocked())
}

// Repeat returns what happens at the end of the queue.
func (e *Engine) Repeat() Repeat {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.repeat
}

// SetRepeat says what happens at the end of the queue.
func (e *Engine) SetRepeat(repeat Repeat) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.repeat = repeat
}

// Position returns how far the track has played.
func (e *Engine) Position() time.Duration {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.current == nil {
		return 0
	}
	return e.current.Position()
}

// Duration returns the length of the track that plays.
func (e *Engine) Duration() time.Duration {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.current == nil {
		return 0
	}
	return e.current.Duration()
}

// Play starts a track of the queue.
func (e *Engine) Play(index int) {
	e.playAt(index, 0)
}

// PlayAt starts a track of the queue at a place in it, which is how a track
// that was playing when the app closed carries on rather than starting again.
func (e *Engine) PlayAt(index int, at time.Duration) {
	e.playAt(index, at)
}

// playAt starts a track at a place in it.
func (e *Engine) playAt(index int, at time.Duration) {
	e.mu.Lock()
	if index < 0 || index >= len(e.queue) {
		e.mu.Unlock()
		return
	}
	e.cursor = e.cursorOf(index)
	track := e.queue[index]
	e.stopLocked()
	e.loading = true
	e.generation++
	generation := e.generation
	e.state = StateBuffering
	e.mu.Unlock()

	e.emit(Event{Kind: KindState, State: StateBuffering})
	go e.start(track, index, generation, at)
}

// Toggle plays the track that plays, or holds it.
func (e *Engine) Toggle() {
	switch e.State() {
	case StatePlaying:
		e.Pause()
	case StatePaused:
		e.Resume()
	case StateIdle, StateFailed:
		e.mu.Lock()
		index := -1
		if len(e.order) > 0 {
			index = e.order[e.cursor]
		}
		e.mu.Unlock()
		if index >= 0 {
			e.Play(index)
		}
	}
}

// Pause holds the track.
func (e *Engine) Pause() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.current == nil {
		return
	}
	e.current.Pause()
	e.state = StatePaused
	e.emit(Event{Kind: KindState, State: StatePaused})
}

// Resume lets the track play on.
func (e *Engine) Resume() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.current == nil {
		return
	}
	e.current.Play()
	e.state = StatePlaying
	// The place stood still while the track was held. Without this, the stall
	// that pausing caused would read as a dead stream, and resuming would end
	// the track at once.
	e.lastMoved = time.Now()
	e.emit(Event{Kind: KindState, State: StatePlaying})
}
// Next plays the track after this one.
func (e *Engine) Next() {
	e.move(1)
}

// Previous plays the track before this one.
func (e *Engine) Previous() {
	e.move(-1)
}

// Seek moves to a place in the track.
func (e *Engine) Seek(at time.Duration) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.current == nil {
		return
	}
	e.current.Seek(at)
	// The place is about to jump, so it is not stalled; and a seek starts the
	// decoder again, which clears its end.
	e.lastPlace, e.lastMoved = at, time.Now()
}

// move steps through the queue.
func (e *Engine) move(step int) {
	e.mu.Lock()
	if len(e.order) == 0 {
		e.mu.Unlock()
		return
	}
	next := e.cursor + step
	if next < 0 {
		next = len(e.order) - 1
	}
	if next >= len(e.order) {
		if e.repeat != RepeatAll {
			e.stopLocked()
			e.state = StateIdle
			e.mu.Unlock()
			e.emit(Event{Kind: KindState, State: StateIdle})
			return
		}
		next = 0
	}
	e.cursor = next
	index := e.order[next]
	e.mu.Unlock()
	e.Play(index)
}

// start resolves the audio of a track and plays it, from at.
func (e *Engine) start(track music.Track, index, generation int, at time.Duration) {
	url, err := e.resolver.Resolve(e.ctx, track.ID)
	if err != nil {
		e.fail(track, index, generation, err)
		return
	}
	duration := track.Duration
	if duration == 0 {
		duration = ytdlp.DurationFromURL(url)
	}
	decoder, err := ffmpeg.NewPCM(e.ctx, url, duration)
	if err != nil {
		e.fail(track, index, generation, err)
		return
	}
	sound := audio.NewPlayer(decoder)

	e.mu.Lock()
	if e.generation != generation {
		// A newer track was asked for while this one was loading.
		e.mu.Unlock()
		sound.Close()
		return
	}
	e.loading = false
	e.current = sound
	e.decoder = decoder
	e.state = StatePlaying
	e.startedAt = time.Now()
	e.lastPlace, e.lastMoved = 0, time.Now()
	volume := e.volume
	e.mu.Unlock()

	// Carry on where the last run stopped, before the first sample plays, so
	// the track does not start again from the beginning.
	if at > 0 && duration > 0 && at < duration {
		sound.Seek(at)
	}

	sound.SetVolume(volume)
	sound.Play()
	e.emit(Event{Kind: KindTrack, Index: index, Track: track, State: StatePlaying})
	e.emit(Event{Kind: KindState, State: StatePlaying})
	e.prefetch()
}

// prefetch resolves the audio of the next track, so the queue keeps moving.
func (e *Engine) prefetch() {
	e.mu.Lock()
	if len(e.order) == 0 {
		e.mu.Unlock()
		return
	}
	next := e.order[(e.cursor+1)%len(e.order)]
	track := e.queue[next]
	e.mu.Unlock()
	go func() {
		_, _ = e.resolver.Resolve(e.ctx, track.ID)
	}()
}

// fail reports a track that could not play.
func (e *Engine) fail(track music.Track, index, generation int, err error) {
	e.mu.Lock()
	if e.generation != generation {
		e.mu.Unlock()
		return
	}
	e.loading = false
	e.state = StateFailed
	e.failure = err
	e.mu.Unlock()
	e.emit(Event{Kind: KindTrack, Index: index, Track: track, State: StateFailed, Err: err})
	e.emit(Event{Kind: KindError, Err: err})
}

// run reports the place in the track and plays the next one when a track
// ends.
func (e *Engine) run() {
	ticker := time.NewTicker(tick)
	defer ticker.Stop()
	for {
		select {
		case <-e.done:
			return
		case <-ticker.C:
			e.tick()
		}
	}
}

// tick reports the place in the track, and moves on when the track ends.
func (e *Engine) tick() {
	e.mu.Lock()
	current := e.current
	decoder := e.decoder
	state := e.state
	started := e.startedAt
	lastPlace := e.lastPlace
	lastMoved := e.lastMoved
	index := -1
	if len(e.order) > 0 && state != StateIdle {
		index = e.order[e.cursor]
	}
	e.mu.Unlock()
	if current == nil {
		return
	}
	position := current.Position()
	duration := current.Duration()

	// Where the place in the track has moved, note when. A place that stands
	// still is the sign that audio has stopped arriving.
	e.mu.Lock()
	if position != lastPlace {
		e.lastPlace, e.lastMoved = position, time.Now()
		lastMoved = e.lastMoved
	}
	e.mu.Unlock()
	still := time.Duration(0)
	if !lastMoved.IsZero() {
		still = time.Since(lastMoved)
	}

	// A track has ended when the place in it reaches its length, not when the
	// player stops reporting that it plays. That flag drops as the mixer
	// reaches the end of what it has decoded — up to a second before the
	// sound has left the speakers — and it also drops while the decoder waits
	// on the network. Going by the flag cut the last second off every track
	// and started the next one over it.
	if state == StatePlaying && time.Since(started) > minPlayTime {
		if duration > 0 && trackEnded(position, duration) {
			e.ended()
			return
		}
		decoderDone := decoder != nil && decoder.Exhausted()
		if e.stopped(position, duration, still, decoderDone) {
			return
		}
		// A stream that has played on since it was found again gets its tries
		// back, so the next break is not the one that uses them up.
		if e.recoveries > 0 && position > e.recoveredAt+recoveryHealthy {
			e.mu.Lock()
			e.recoveries = 0
			e.mu.Unlock()
		}
	}
	e.emit(Event{Kind: KindPosition, Index: index, Position: position, Duration: duration})
}

// stopped handles a stream that stopped before the end of its track. It reports
// whether it dealt with it, which it does as soon as the place has stood still
// long enough to mean something.
//
// The place standing still is the signal, and the only one that covers both
// ways a stream stops: a decoder that finished moves it no further, and neither
// does one still running against a connection that went quiet. What the decoder
// adds is how long to wait before believing it: once it has finished, only the
// ring's last second has to play out, so a shorter wait will do.
func (e *Engine) stopped(position, duration, still time.Duration, decoderDone bool) bool {
	if position <= 0 {
		return false
	}
	limit := stallTime
	if decoderDone {
		limit = drainTime
	}
	if still < limit {
		return false
	}
	if nearTheEnd(position, duration) {
		// YouTube stops serving a track's data at its end, so a stream that
		// stopped here is the track finishing, not a stream to find again.
		e.ended()
		return true
	}
	if e.recover(position) {
		return true
	}
	// The tries are used up: the track is over and the queue moves on.
	e.ended()
	return true
}

// recover finds the stream of the track that is playing again and starts it
// from where it stopped. It reports false when the tries are used up.
func (e *Engine) recover(position time.Duration) bool {
	e.mu.Lock()
	if len(e.order) == 0 {
		e.mu.Unlock()
		return false
	}
	index := e.order[e.cursor]
	if index < 0 || index >= len(e.queue) {
		e.mu.Unlock()
		return false
	}
	// A new track gets its own tries: the count belongs to the stream that is
	// failing, not to the queue.
	if e.recoveryFor != index {
		e.recoveryFor, e.recoveries = index, 0
	}
	if e.recoveries >= maxRecoveries {
		e.mu.Unlock()
		return false
	}
	e.recoveries++
	e.recoveredAt = position
	attempt := e.recoveries
	track := e.queue[index]
	duration := track.Duration
	e.mu.Unlock()

	log.Printf("playback: the stream stopped at %v of %v, finding it again (%d/%d)",
		position.Round(time.Second), duration.Round(time.Second), attempt, maxRecoveries)

	// The link it was given is the usual reason it broke, so the cached one is
	// dropped and yt-dlp is asked again.
	if e.resolver != nil {
		e.resolver.Forget(track.ID)
	}
	e.playAt(index, position)
	return true
}

// How a stream that stops is handled.
//
// A stream can stop before the track's length for two different reasons, and
// they are not the same problem:
//
//   - the decoder finished, which says itself that no more audio will come;
//   - the decoder is still running but no audio is arriving, which is a
//     connection that went quiet.
//
// Both leave the place in the track standing still. Neither is the end of the
// track, so the answer is not to skip it: the link is found again and the
// track plays on from where it stopped. Only when that fails, or when the
// stream stopped in the last moment of the track, is it over.
const (
	// stallTime is how long the place in a track may stand still before the
	// stream is taken for stopped. MyGo's player reads about a second ahead,
	// so the ordinary drain of that buffer keeps the place moving for a
	// moment after the decoder finishes; three seconds is comfortably past it.
	stallTime = 3 * time.Second
	// drainTime is the wait once the decoder has finished. Its ring holds
	// about a second of audio, so the place stops moving a second after the
	// decoder says it is done: a wait shorter than the one above, and still
	// well past that second.
	drainTime = 1500 * time.Millisecond
	// endOfTrackLead is how much of a track may be left when its stream stops
	// and the track still counts as over. YouTube stops serving a track's data
	// at its end, which cuts the stream off in the last moment; finding it
	// again there would play that moment a second time.
	endOfTrackLead = 3 * time.Second
	// maxRecoveries is how many times in a row a stream is found again before
	// the track is given up on.
	maxRecoveries = 3
	// recoveryHealthy is how far a recovered stream plays before it counts as
	// working again, which gives the next break a fresh set of tries.
	recoveryHealthy = 10 * time.Second
)

// trackEnded tells whether a track at position has reached the end of its
// duration.
//
// The tolerance is small on purpose: a tenth of a second of slack for the place
// to run past the length as the buffer drains. Anything wider starts cutting
// the last moments off a track, which is the bug this replaced.
func trackEnded(position, duration time.Duration) bool {
	if duration <= 0 {
		return false
	}
	return position >= duration-100*time.Millisecond
}

// nearTheEnd reports whether so little of the track is left that a stream which
// stopped there is over rather than broken.
func nearTheEnd(position, duration time.Duration) bool {
	return duration > 0 && duration-position <= endOfTrackLead
}

// ended plays the next track when one ends.
func (e *Engine) ended() {
	e.mu.Lock()
	repeat := e.repeat
	index := -1
	if len(e.order) > 0 {
		index = e.order[e.cursor]
	}
	e.mu.Unlock()
	if repeat == RepeatOne && index >= 0 {
		// The track starts again from the beginning, and is resolved again:
		// a stream that died early died with the link it was given, and the
		// same link would die in the same place.
		e.playAt(index, 0)
		return
	}
	e.Next()
}

// emit sends an event, without waiting for the interface to take it.
func (e *Engine) emit(event Event) {
	select {
	case e.events <- event:
	default:
	}
}

// stopLocked stops what plays. The caller holds the lock.
func (e *Engine) stopLocked() {
	if e.current != nil {
		e.current.Close()
		e.current = nil
	}
	e.decoder = nil
	e.loading = false
}

// cursorOf returns the place of a track in the order.
func (e *Engine) cursorOf(index int) int {
	for i, item := range e.order {
		if item == index {
			return i
		}
	}
	return e.cursor
}

// CurrentLocked returns the track that plays. The caller holds the lock.
func (e *Engine) CurrentLocked() int {
	if len(e.order) == 0 {
		return -1
	}
	return e.order[e.cursor]
}

// indexOf returns the place of a track in the order, or -1.
func (e *Engine) indexOf(index int) int {
	if index < 0 {
		return 0
	}
	for i, item := range e.order {
		if item == index {
			return i
		}
	}
	return 0
}

// shuffleLocked puts the order at random, keeping the track that plays first.
func (e *Engine) shuffleLocked() {
	e.order = shuffledOrder(len(e.queue), e.CurrentLocked())
	e.cursor = 0
}

// shuffledOrder returns the order of a queue at random, with the track that
// plays first and every other track after it, once.
func shuffledOrder(count, current int) []int {
	order := make([]int, 0, count)
	rest := make([]int, 0, count)
	for i := range count {
		if i == current {
			continue
		}
		rest = append(rest, i)
	}
	rand.Shuffle(len(rest), func(i, j int) { rest[i], rest[j] = rest[j], rest[i] })
	if current >= 0 && current < count {
		order = append(order, current)
	}
	return append(order, rest...)
}

// clampVolume keeps a volume between 0 and 1.
func clampVolume(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v > 1:
		return 1
	default:
		return v
	}
}
