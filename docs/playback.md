# When a track ends, and when its stream does

A track can stop for two different reasons, and treating them the same was a
bug in both directions. It is worth writing down, because the second version of
this logic was still wrong.

## The three ways a track stops

| | What happened | What to do |
|---|---|---|
| **It played out** | the place reaches the track's length | play the next track |
| **Its stream broke** | the place stops moving, well before the length | **find the stream again and play on from there** |
| **It stopped in the last moment** | the place stops within 3s of the length | play the next track |

The middle one is the one that was wrong twice.

## What went wrong

**First bug: the queue stopped.** A stream died at 2:38 of 3:15. The only rule
was "the place reached the length", which could never fire, so nothing
advanced and the music simply stopped.

**Second bug: the fix only covered one kind of stop.** The replacement waited
for the decoder to report that it had finished. But a stream can also go *quiet*
without finishing: the connection stalls, ffmpeg sits waiting, and the decoder
never reports an end. The screenshot that showed this was **4:21 of 5:37** — the
place had stopped, the decoder had not finished, and the rule never fired.

**And the fix was aimed at the wrong outcome.** Even when it did fire, it
*skipped the track*. A broken connection is not a track that finished. Skipping
throws away the rest of a song because a link expired for a moment.

## What it does now

The signal is **the place in the track standing still**. That covers both kinds
of stop: a decoder that finished moves it no further, and neither does one still
running against a connection that went quiet. The place only moves when samples
reach the mixer, so a track playing normally keeps moving.

When the place has stood still:

1. **Within 3 seconds of the end?** The track is over. YouTube stops serving a
   track's data at its end, so this is the ordinary finish.
2. **Otherwise, find the stream again** — drop the cached URL (it is usually the
   reason the stream broke) and ask yt-dlp for a fresh one — then play on **from
   the position reached**.
3. **Three tries in a row**, with the count reset when a recovered stream plays
   10 seconds without breaking. After that, the track is given up on and the
   queue moves on.

A pause also stops the place, which is why the rule only runs while the engine
believes a track plays.

## The two waits

| | Wait | Why |
|---|---|---|
| decoder finished | 1.5s | its ring holds about a second of audio, so the place stops a second after the decoder is done |
| decoder still running | 3s | no such signal, so a longer wait before believing the place |

Both are longer than the one-second read-ahead. A wait shorter than that would
take the ordinary drain of the buffer for a stopped stream and interrupt every
track.

## The tests

Pure rules, no device:

```powershell
go test ./internal/player/ -run 'TestAStoppedStream|TestAStreamThatStopped|TestAStopInTheLast|TestTheRecovery|TestARecovered'
```

The real thing, which kills a live stream and checks the same track plays on:

```powershell
go test -tags e2e ./internal/player/ -run TestTheStreamIsFoundAgain
```

That one is the regression test for this bug: it plays a real track, kills the
decoder partway, and fails unless the track continues **on the same track** from
where it stopped.
