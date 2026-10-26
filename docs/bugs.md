# Bugs

Every bug this app had, written down so it can be gone back to: the symptom as
it was reported, the cause, the fix, and the test that now fails if it returns.

The point is not the history. It is that each one has a test, and this file says
which test, so a change that reintroduces a bug fails somewhere rather than
being noticed weeks later by someone listening to music.

---

## A track stopped mid-song and the music never came back

**Reported as:** "audio stopped before ended, it just stopped and not playing
next song", with a screenshot showing **2:38 of 3:15**. Then again, after a
first fix: "still stopped at certain times, 1-2 min before music end", showing
**4:21 of 5:37**.

**Cause, first version.** The engine had one rule for the end of a track: the
place in it reaches its length. A stream that dies early never gets there, so
the rule could never fire and nothing else advanced the queue.

**Cause, second version — and this is the interesting one.** The fix waited for
the decoder to report that it had finished. But a stream can also go *quiet*:
the connection stalls, ffmpeg sits waiting, and the decoder never reports an
end. So the rule still did not fire, which is why the bug came back looking
almost the same.

It was also aimed at the wrong outcome. Even when it fired, it **skipped the
track** — throwing away the rest of a song because a link had expired for a
moment.

**Fix.** The signal is the place in the track standing still, which covers both
kinds of stop: a decoder that finished moves it no further, and neither does one
running against a dead connection. The place only moves when samples reach the
mixer. When it has stood still:

1. within 3 seconds of the end → the track is over (YouTube stops serving a
   track's data at its end, so this is the ordinary finish);
2. otherwise → drop the cached URL, resolve the track again, and **play on from
   the position reached**;
3. three tries in a row, the count reset when a recovered stream plays 10
   seconds cleanly. After that the track is given up on.

Two waits, because the two cases differ: **1.5s** once the decoder has finished
(its ring holds about a second of audio, so the place stops a second later), and
**3s** while it is still running, where there is no such signal. Both are longer
than the one-second read-ahead — a shorter wait would take the ordinary drain of
the buffer for a stopped stream and interrupt every track.

**Tests.** `TestAStoppedStreamIsNotTheEndOfTheTrack`,
`TestAStopInTheLastMomentIsTheEnd`, `TestTheRecoveryWaitsLongerThanTheReadAhead`,
`TestTheRecoveryTriesAreBounded` (pure rules), and
`TestTheStreamIsFoundAgain` (e2e: kills a real stream and fails unless the same
track continues). See `docs/playback.md`.

**Verified against the real thing:** a stream killed at 6s of 3:33; the same
track played on from 7s and the queue did not move.

---

## A track still stopped a second or two before its end

**Reported as:** "fix why music stopped before the music end" — again, after the
recovery above was in place.

**Cause.** Three, and only the last one is subtle.

1. **`-xerror`.** It makes ffmpeg abort the whole process on the first
   *recoverable* error. googlevideo drops connections mid-stream routinely, so a
   single dropped read ended the song.
2. **No reconnect.** ffmpeg was told nothing about reopening a streamed HTTP
   source that went quiet, so it sat on a dead socket while the place in the
   track stood still — which the engine then read as a stall.
3. **A clean end was taken for a break.** YouTube **rounds** `lengthSeconds`, so
   a track advertised at 3:42 can hold 3:40 of audio. ffmpeg wrote its last
   sample and exited *cleanly*, leaving the place standing still two seconds
   short of the length. The stall rule fired, the stream was dropped and
   resolved again, and the track played on from where it stopped — **three
   times**, with a wait each time. What you heard was the last seconds of the
   song three times over, then the next track.

**Fix.** The decoder now reports `Finished()` apart from `Exhausted()`, and a
clean finish ends the track instead of recovering it. `-xerror` is gone, the
reconnect flags are in, and the offset is an input option.

**Tests.** `TestAFinishedStreamEndsTheTrack`,
`TestAFinishedStreamIsNeverRecovered`, `TestABrokenStreamIsStillRecovered`,
`TestFinishedIsFalseForADecoderThatCannotSay`,
`TestTheStreamIsAllowedToRecover`, `TestALocalFileIsNotGivenNetworkFlags`,
`TestTheOffsetComesBeforeTheInput` (pure), and `TestARealTrackPlaysToItsEnd`
(e2e: plays a real track to its last second).

**Verified:** the e2e check reported **3:06 of 3:07**; and in the running app a
182-track playlist played a track to its end and moved to the next one at 0:41
with no silence and no repeat.

---

## The window froze the moment a track started

**Reported as:** Windows showed "Not Responding" and the app never recovered.

**Cause.** `apply` held `a.mu` and called `a.tray.refresh()`, which asked the app
what was playing — locking the same mutex. **Go's mutex is not reentrant**, so
the thread that draws waited on itself forever.

**Fix.** The tray is *told* the track and the state, and reads them from the
caller's own copy (`trackLocked`). Anything that reaches the desktop — a tray
call, a window call — happens outside the lock or on a goroutine.

**Tests.** `TestApplyingATrackDoesNotDeadlock` runs that exact path on a
deadline, because a deadlocked test would hang the suite instead of failing it.
`TestNoMethodLocksTwiceInOnePath` reads the source and fails on any method that
locks and then calls a method that locks again — a structural guard, so the next
method cannot repeat it.

---

## A terminal opened beside the app, and the taskbar had no icon

**Reported as:** "when installed, there is a terminal running, remove it" and
"logo on taskbar still not seen".

**Cause.** The release ran plain `go build`, which produces a
**console-subsystem** executable with no icon resources. Two symptoms, one
cause.

**Fix.** The release uses `mygo build`, which links `-H=windowsgui`, embeds the
icon at every size Windows asks for, writes the version resource, and makes the
NSIS installer.

**A second, separate cause for the terminal**, found while fixing the first: a
GUI app that starts a console program makes Windows open a console *for the
child*, so `yt-dlp` and `ffmpeg` each flashed a black rectangle on every track.
`internal/proc` sets `CREATE_NO_WINDOW` on both.

**A second cause for the icon:** the window now sets its own with
`win.SetIcon(appIcon)` at startup, so the taskbar is right even for `go run` and
the dev server, which have no resources at all.

**Tests.** `TestEveryChildIsStartedHidden` reads the source and fails on any
`exec.Command` without `proc.Hide` — the helper being correct is not enough, the
bug was that nothing called it. I confirmed it fails by deleting a call.

**Verified:** the shipped executable reports subsystem 2, carries its icon, opens
one window and no console, and the taskbar draws the mark.

---

## A release queued for two hours and published nothing

**Cause.** `macos-13` is a **retired runner label**. A job on a retired label
does not fail — it waits for a machine that will never exist, and because the
publish step waits on every platform, the whole release hung. Two releases were
lost this way.

**Fix.** `macos-15-intel`, the oldest Intel image still served, and a
**25-minute timeout** on the build job, so a runner that never picks up the job
fails instead of blocking a release indefinitely.

**The lesson:** the failure was invisible rather than loud. That is why the
timeout matters more than the label change.

**Also in the same workflow:** the Windows job failed with `exit 127` because it
called `zip`, which is not on a Windows runner. It uses `Compress-Archive`.

---

## Saving a Last.fm key looked like nothing happened

**Reported as:** "after I add lastfm key and click save, nothing happen, should
be a loading state and the key gone or masked with ****".

**Cause.** Two things. The button configured the client and saved to disk, but
the page went on drawing the same fields, because the frame had already read
`status.Configured` **before** the click was handled. And a wrong key was
indistinguishable from a right one until the first scrobble failed — by which
time the mistake was a day old.

**Fix.** The key is **checked with Last.fm before it is kept** (a token request
needs the key and secret and nothing else, and changes nothing about the
account). While it is being checked the page shows a spinner; when it passes the
fields are replaced with "Key saved" and a **Replace** button, so the secret is
not left on screen. A refused key is not configured, the reason is shown, and
the fields are left for correction.

**Tests.** `TestAShortKeyIsRefusedBeforeACall` (six cases, including a key of the
wrong length and spaces), `TestAKeyOfTheRightShapeIsCheckedAndThenKept`,
`TestAKeyLastFmRefusesIsNotKept`.

---

## A test scrobbled to the real Last.fm

**Found by:** CI, not locally. The first release run of the Last.fm feature
failed with `403 Invalid API key` in the log — a test had built a client at the
**real endpoint** and scrobbled with a made-up key.

**Cause.** The helper that makes a "connected" scrobbler configured a real
client. Tests that never reached delivery passed locally; CI's ordering exposed
it.

**Fix.** Counting tests use a scrobbler with **no session key**, so nothing can
be delivered at all. Delivery tests point the client at a **closed local port**,
so a stray call fails instantly instead of reaching the internet.
`configure` now preserves a client's `Root`, so a stub survives reconfiguring.

**Test.** `TestTheScrobblerTestsStayOffline` reads the sources and fails if any
test file builds a client without pointing it at a stub. I confirmed it fails by
removing the redirect.

**Why it matters:** a suite that reaches the internet is slow, flaky, and reports
failures about someone else's service rather than about this code.

---

## A track was cut off a second before its end

**Cause.** The engine went by the player's `Playing()` flag. That flag drops as
the mixer reaches the end of what it has **decoded** — up to a second before the
sound has left the speakers — and it also drops while the decoder waits on the
network.

**Fix.** Compare the place in the track against its length, with 100ms of slack
for the place to run past as the buffer drains. A track of unknown length is
never ended by its place, because there is nothing to compare against.

**Tests.** `TestTrackEndsAtItsLength` (eight cases), `TestTrackEndedAllowsForTheDrain`,
`TestATrackOfUnknownLengthIsNotCutShort`.

---

## The two switches on the Look page did nothing

**Reported as:** the knob moved, the setting never changed.

**Cause.** Two things. MyGo flips a switch's bound value **as the control is
built**, so a fresh local each frame is always equal to what was passed in. And
`tester.Click(name)` clicks the centre of the label, while the switch sits at the
right of its row.

**Fix.** The value lives on the app (`a.switchShuffle`, `a.switchDynamic`,
`a.switchDiscord`), the switch is handed a pointer to it, and the caller reads
the field back once the frame has drawn. A click moves the field; nothing else
does.

**Tests.** `src/switches_test.go` drives the switch through the same binding the
page uses, so it checks the wiring rather than the hit area.

---

## The Discord panel had no progress bar and no album line

**Cause.** Discord draws the elapsed-against-total bar only when it knows **both**
ends of the timestamp; the app sent only a start, so it showed a counter and no
bar. And the protocol has two text fields, not three, so the album had nowhere to
go.

**Fix.** Send `start` **and** `end`. Put the album on a second line **inside**
`state`, which Discord renders as a line of its own.

**A bug in that fix, caught by its own test:** joining artist and album directly
produced `"A\nB\nC\nD"` — **four lines** — when either name contained a newline.
Line breaks are flattened before joining.

**Tests.** `TestTheStateLineCarriesTheAlbum`,
`TestTheStateLineHasAtMostTwoLines`.

**Still open:** artwork does not render, because Discord fetches images from a URL
and will not display YouTube's thumbnails. It needs assets uploaded to the
Discord application. `docs/discord.md`

---

## The most-played list flickered

**Cause.** Its sort was not a **total order**: tracks with the same count could
swap places on every redraw.

**Fix.** Sort by count, then by a stable tie-breaker, so the same data always
produces the same order.

---

## Smaller ones

- **Up next opened on the first track** rather than the one playing. It now
  scrolls to the current track, and stops following once the reader scrolls.
- **The tray icon did not fill its bitmap**, and playing and paused looked the
  same. Both are drawn to the canvas edge with distinct shapes.
- **A playlist could not be deleted.** Right-click, a confirmation, and the cache
  entry goes too.
- **The loading spinner stayed** after a load finished. The form resets on
  success and the spinner clears only when loading is false.
- **A key of the wrong length reached Last.fm.** It is now refused before the
  call, because the message that comes back is less useful than one that says
  what is wrong.

---

## The rules these bugs produced

Worth reading before changing this code.

1. **A method that holds `a.mu` may not call one that takes it.** Use a
   `*Locked` helper that assumes the lock is held, and hand the callee what it
   needs as arguments.
2. **Do not decide playback from `Playing()`.** It drops early and it drops
   while buffering. Use the place in the track.
3. **A stream that breaks is not a track that finished.** Recover it; do not skip.
4. **A stream that finished is not a stream that broke.** YouTube rounds its
   lengths, so a clean end short of the metadata is the end. Never recover it.
5. **Every spawn of a console program needs `proc.Hide`.**
6. **Every `go build` for a release is wrong.** It is `mygo build`.
7. **A test may not reach the internet.** Point the client at a stub.
8. **A fix needs a test that fails without it.** Every bug above has one, and
   for the structural ones I checked the test fails by reverting the fix.
