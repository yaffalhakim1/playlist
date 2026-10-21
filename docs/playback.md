# When a track ends

A track can end in two ways, and the queue has to notice both. It only noticed
one, which is why a track could stop at 2:38 of 3:15 and nothing played after
it.

## The two ways

| | What happens | How the engine knows |
|---|---|---|
| **It plays out** | the place in the track reaches its length | `trackEnded`: the place is within 100ms of the length |
| **Its stream dies** | the stream stops before the length the metadata promised | `streamEnded`: the decoder is finished *and* the place has stopped moving |

The second one is the bug. A dropped connection, an expired link, or a video
whose audio runs short of its metadata leaves the place at 2:38 while the track
claims 3:15. The length comparison can never fire, because the place never gets
there. Nothing else advanced the queue, so the music simply stopped.

## Why both halves are needed

The obvious fix — end the track when the decoder finishes — would cut the last
second off every track.

MyGo's audio player reads about a second ahead into a ring buffer. The decoder
therefore reaches its end while a second of audio is still waiting to be heard.
Ending on "the decoder finished" alone would start the next track over that
second, which is the bug the length comparison was introduced to fix in the
first place.

So the engine waits for both:

- **the decoder is finished**, which `ffmpeg.PCM.Exhausted()` reports; and
- **the place in the track has stopped moving** for `stallTime` (three
  seconds).

While the ring drains, the place still advances. Once it stops advancing with
the decoder finished, nothing more is coming.

Three seconds is comfortably past the one-second read ahead, and short enough
that a dead stream does not leave the player silent for long.

## What must not look like a stall

Two things stop the place moving without the stream being dead, and both reset
the stall clock:

- **Holding the track.** `Resume` sets the clock to now. Without it, a track
  held for more than three seconds would end the moment it played again.
- **Seeking.** `Seek` sets the clock to now, and the seek restarts ffmpeg,
  which clears `Exhausted`.

A decoder that has merely not produced samples yet is not finished, so a stream
waiting on the network is not the end either.

## Repeat

`RepeatOne` replays the track by loading it again rather than seeking the old
player. A stream that died early died with the link it was given, so the same
link would die in the same place.

## The tests

The rules are pure functions, so they are tested without a device:

```powershell
go test ./internal/player/ -run 'TestAStream|TestTheOrdinary|TestADecoder|TestAStall|TestTheStall|TestHolding|TestSeeking|TestRepeat'
```

The decoder's flag is tested against real ffmpeg and a local file, so it needs
no network:

```powershell
go test ./internal/ffmpeg/ -run TestExhausted
```

And the whole thing is checked on a real stream, which is slow:

```powershell
go test -tags e2e ./internal/player/ -run TestEngineAdvancesWhenTheStreamDies
```

That one plays a real track, kills the decoder partway, and fails unless the
queue moves on by itself.
