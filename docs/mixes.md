# Mixes

**Start a mix** on the player builds a mix around the track that is playing.

## Why it needs no account

A mix is not a playlist YouTube stores. It is built when it is asked for, around
one track. So the only thing a mix needs is a video id — no cookies, no Google
sign-in, no history.

That makes it the one part of "your mixes" the app can offer without
authentication. Browsing the mixes YouTube Music puts on *your* home page is a
different feature: that is the personal feed, it needs a signed-in session, and
it is not what this does.

## The id

YouTube Music names a mix `RDAMVM` followed by the video id:

```
RDAMVM + Il-an3K9pjg  →  RDAMVMIl-an3K9pjg
```

The link carries both parts:

```
https://www.youtube.com/watch?v=Il-an3K9pjg&list=RDAMVMIl-an3K9pjg
```

**Both are needed.** The watch id is not decoration: a link with only the list
id is refused with *"This playlist type is unviewable"*. The list is built
around the track being watched, so YouTube needs to know which track that is.
This was checked against the real service, not assumed.

## How it is read

Mixes are read by yt-dlp, not InnerTube. InnerTube's browse endpoint answers a
mix with an empty body, because there is no stored playlist to browse. That is
already how the loader worked — it tries InnerTube first and falls back to
yt-dlp — so a mix takes the same path as any other playlist.

A mix's tracks come back from yt-dlp with a title and a **channel**. The channel
is the artist when the title does not name one — it is stripped of YouTube's
`- Topic` suffix, and a name that is really a label (`Various Artists`) is left
out rather than guessed at. On a real mix that is 528 of 528 tracks naming an
artist, where before it was none:

- the queue shows what it should under the title;
- Last.fm can record a play, because it has an artist to record it against.

## The code

| Path | Responsibility |
|---|---|
| `internal/innertube/client.go` | `MixID`, `MixLink`, `MixTitle` |
| `src/app.go` | `playMix`, which opens the mix through the ordinary load path |
| `src/page_player.go` | the button, under the transport |

`playMix` builds the link and calls `load()`, so a mix is opened, cached,
remembered in the sidebar and played exactly like any other playlist. There is
no second path to keep working.

The button is left out of the smallest window, where the stage fills the height
and it would sit under the player bar. `TestPagesKeepClearOfTheHeaderAndTheBar`
is what enforces that.

## The tests

```powershell
go test ./internal/innertube/ -run 'TestMix|TestAWatchLink'
go test ./internal/ytdlp/ -run 'TestAMixTrackGetsTheArtistOfItsChannel'
```

And the real one, which reads a mix from YouTube through the whole loader:

```powershell
go test -tags e2e ./internal/playlist/ -run TestAMixFromATrackLoads
```

That one checks the mix holds several tracks, that it holds the track it was
built from, that every track has an id to play, and that the tracks name an
artist.
