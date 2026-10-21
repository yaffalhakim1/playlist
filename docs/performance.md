# Performance

What this app does to stay fast, why, and what to watch if it stops being fast.

Most of it is one idea: **keep work off the thread that draws.** A frame that
takes too long is a stutter the user feels; a network call on that thread is a
frozen window.

---

## The thread that draws

The view is a pure function of the state: it draws the state and nothing else.
Anything slow happens elsewhere and comes back through `win.Update`, which runs
on the main thread.

| Work | Where it runs | Why |
|---|---|---|
| Resolving a stream (yt-dlp) | goroutine | it takes seconds and reaches the network |
| Decoding (ffmpeg) | its own process, read by MyGo's audio thread | never on the frame |
| Discord presence | goroutine per call | a socket write must not hold up a frame |
| Last.fm now-playing and scrobbles | goroutine per call | same |
| Writing the outbox | the outbox's own goroutine | a disk write must not hold up a frame |
| Saving the session | debounced, 5s | see below |
| Artwork | its own goroutine, `onReady` when done | decoding an image is not a frame's work |

The rule that follows: **never store a `ui.Element` or a `*ui.Context` in the
state.** They belong to one frame.

---

## The writes that could cost a frame

The engine reports the place in the track **five times a second**. Writing the
session at that rate is 300 disk writes a minute, which costs more than the
second of accuracy it buys.

- The session is written when a track changes, and every `sessionInterval`
  (5 seconds) while it plays. A crash loses at most 5 seconds of position.
- Settings are written only when a setting changes.
- The Last.fm outbox is written **in memory at once** and flushed by its own
  goroutine, so a caller never waits on the disk.

---

## The buffers

| Buffer | Size | Consequence |
|---|---|---|
| MyGo's audio ring | about 1 second | the decoder finishes ~1s before the audio does |
| `ffmpeg` read chunk | 32 KB | one read is ~170ms of 48kHz stereo |
| Last.fm outbox | 2000 plays | a machine offline for months does not grow a file without end |
| Playlist cache | 10 playlists | the file is read and written as the music plays |

The one-second ring is behind the subtlest bug in this codebase: it is why a
"decoder finished" signal alone cuts the last second off every track. See
`docs/playback.md`.

---

## The network

- Every call has a timeout: 30s for Last.fm, 3 minutes for a playlist load, 3
  minutes for the e2e checks. Nothing waits forever.
- The resolver caches stream URLs for **30 minutes**, keyed by video, capped in
  size. A recovery **forgets** the cached URL first, because the link is usually
  why the stream broke.
- The next track's stream is resolved while the current one plays, so a track
  change does not wait on yt-dlp.
- A playlist is read once and kept; opening it again costs **no network at all**.
  Two tests pin that: `TestOpeningACachedPlaylistNeverLoads` and
  `TestRestoringNeverLoads` fail if a cached open bumps the load generation.

---

## The frame

- **One inset, one gap** per page (`pageInset`), so spacing is a decision made
  once rather than per widget.
- **Type roles** (`pageName`, `sectionName`, `itemName`, `body`, `caption`,
  `eyebrow`) carry size, weight, tracking and leading as one decision, so a page
  cannot accidentally ask the font for a new combination.
- **Lists are `components.List`**, which MyGo virtualizes: only the rows in view
  are drawn. A playlist of 340 tracks does not draw 340 rows.
- **Only two things animate**: `motionReveal` brings a page in, `motionPress`
  settles a control. Both are springs, and MyGo stills them when the desktop asks
  for less motion.

---

## What is checked, and what is not

**Checked** by the suite:

- layout: pages stay clear of the header and the player bar at four window sizes,
  and do not overlap their neighbours (`layout_test.go`)
- design: pages use the type scale and theme colours, and write no size or colour
  by hand (`design_test.go`)
- the font is really in use, rather than silently falling back
  (`TestFontInUse` — it draws the same words in the app's family and in one that
  does not exist, and fails if the pictures match)

**Not checked**, and worth measuring if it ever feels slow:

- **frame time under load.** There is no benchmark of a redraw. The app is small
  enough that this has not been a problem, but nothing would catch a regression.
- **memory over a long session.** The caches are bounded, but nothing measures
  them.
- **`go test -race`.** It cannot run here: there is no C toolchain for the race
  detector. Concurrency is guarded by ordinary tests instead —
  `TestNoMethodLocksTwiceInOnePath` reads the source for the deadlock that
  started it, and `TestApplyingATrackDoesNotDeadlock` runs that path on a
  deadline.

If you want a starting point for profiling: the redraw starts at `a.frame` in
`src/app.go`, and the engine's tick is `internal/player/engine.go` (`tick`,
200ms). Those are the two places where work on the wrong thread would show up
first.
