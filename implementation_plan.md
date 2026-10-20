# YouTube Playlist Player — implementation plan

Native desktop music player. Paste a playlist link, it plays the playlist.

## Stack (verified, not assumed)

| Piece | Choice | Verified how |
|---|---|---|
| UI | MyGo native UI (`github.com/egoist/mygo/ui`), GPU-drawn, no webview | `mygo.egoist.dev/docs/ui`, `go doc` on v0.5.5 |
| Components | Onevra (`github.com/oneo-me/onevra/source` v0.2026.13) | `go doc` on the pinned version |
| Look | Material 3 (tonal surfaces, rounded, dynamic accent from album art) | design language, implemented in `src/theme.go` |
| Metadata | YouTube Music InnerTube `browse` endpoint, WEB_REMIX client, pure Go | POST to `music.youtube.com/youtubei/v1/browse` — returned real playlist data |
| Stream URL | `yt-dlp -f bestaudio -g` (handles YouTube's cipher / PO tokens) | ran it here: returned a working `itag=251` googlevideo URL |
| Decode | `ffmpeg` → s16le PCM → `mygo/audio` `Decoder` (WASAPI on Windows) | `go doc github.com/egoist/mygo/audio`; ffmpeg 7.1 present |

External binaries required at runtime: **yt-dlp** and **ffmpeg**. Both are on this
machine (`yt-dlp 2026.03.17`, `ffmpeg 7.1`). The app reports a clear error when
one is missing instead of failing silently.

## Architecture

```
main.go                  entry point only → src.Run()
src/                     native UI (one package, mirrors opencode-gui)
  app.go                 app state, window, root view
  theme.go               Material 3 colour roles → ui.Theme
  sidebar.go             library / recents / paste box
  nowplaying.go          big artwork + title/artist/album
  tracklist.go           virtualised track list
  playerbar.go           transport controls + seek + volume
  shortcuts.go           keyboard bindings
internal/
  music/                 domain model: Track, Playlist
  innertube/             InnerTube browse client + response parsing
  stream/                yt-dlp URL resolution + ffmpeg PCM decoder (audio.Decoder)
  player/                playback engine: queue, shuffle/repeat, autoplay, position
  library/               persistence: recents + settings (JSON in PathUserData)
```

Dependency direction: `src → player → stream → music`, `src → innertube → music`.
No cycles. `music` holds the shared domain types so nothing is duplicated.

## The three hard parts

1. **Metadata.** InnerTube `browse` with `browseId = "VL" + playlistID` returns
   `musicResponsiveListItemRenderer` items: `playlistItemData.videoId`, title and
   byline in `flexColumns`, duration in `fixedColumns`, artwork in `thumbnail`.
   Long playlists paginate with a `continuationItemRenderer` token. The parser
   walks the JSON for those renderers, so a layout change in one branch does not
   break the rest. yt-dlp is *not* used here: it returns no artist or album for
   music.youtube.com entries, which is exactly what a music player needs.

2. **Audio.** `ffmpeg -ss <t> -i <url> -vn -f s16le -ar 48000 -ac 2 -` feeds
   `stream.PCM`, which implements `audio.Decoder` (SampleRate/Channels/Read/
   SetPosition/Len) and hands it to `audio.NewPlayer`. Seeking restarts ffmpeg
   at the offset; `Len` comes from the track duration. Output is MyGo's own
   mixer → WASAPI on Windows, so the app owns the audio path, no external player.

3. **Threading.** The view is a pure function of state. The engine runs its own
   goroutine and publishes events on a channel; the UI reads that channel and
   applies changes with `win.Update`, which runs on the main thread. All shared
   state is mutex-guarded.

## Risks

| Risk | Mitigation |
|---|---|
| YouTube changes InnerTube/yt-dlp | yt-dlp is the maintained path for streams; metadata failure shows a toast, not a crash |
| yt-dlp / ffmpeg missing on the target machine | explicit startup check with the install command in the message |
| Long playlists (500+ items) | InnerTube pagination + virtualised `ui.List` |
| Stream URLs expire (they carry `expire=`) | re-resolve per track, never cache a URL longer than the track |
| Decode stall on a slow network | ffmpeg reads at its own pace; the UI shows a buffering state |

## Verification

- `go build ./...` — compiles.
- `go tool mygo vet .` — MyGo's native-UI lifetime checks (elements in state, goroutine captures).
- `go test ./...` — unit tests for the pure logic: InnerTube parsing against a
  saved response fixture, URL parsing, PCM conversion, engine queue/shuffle/repeat
  transitions with a fake stream.
- `ui.NewTester` — layout and interaction assertions on the views.
- Live check: paste a real playlist, play a track, hear audio.
