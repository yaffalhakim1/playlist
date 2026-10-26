# Plan — MyGo only, motion, states, playback fix

## Verdict

APPROVED, with one structural change: **drop Onevra completely**. The app keeps
its layout (rail · page · player bar) and its copy, and moves every widget to
`github.com/egoist/mygo/ui` v0.6.4. That removes 24 files' worth of an external
component library, one dependency, and one whole layer between the app and MyGo.

## 1. Dependencies

| Before | After |
|---|---|
| `github.com/egoist/mygo v0.5.5` | `v0.6.4` |
| `github.com/oneo-me/onevra/source v0.2026.13` | **removed** |

Onevra's `shells/sidebar` is replaced by `src/shellview.go` becoming the real
view — the same layout the tests already draw, now the only path.

## 2. What replaces Onevra

| Onevra | MyGo |
|---|---|
| `components.Text` | `ui.Text` |
| `components.Button` (3 variants) | `ui.ButtonBase` + our own paint (`src/ui.go`) |
| `components.Icon` | `ui.Icon` |
| `components.IconButton` | `ui.ButtonBase` + `ui.Icon` + `Tooltip` |
| `components.List` | `ui.List` |
| `components.ScrollView` | `ui.Scroll`, `ui.ScrollHorizontal` |
| `components.Sidebar` | `ui.Sidebar` + `SidebarSection` + `SidebarItem` |
| `components.Slider` | `ui.Slider` |
| `components.Switch` | `ui.Switch` |
| `components.TextBox` | `ui.TextInput` |
| `components.Spinner` | `ui.Spinner` |
| `components.Segmented` | `ui.Segmented` |
| `components.Toasts` | built in by MyGo (`Context.AddToast`) |
| `components.ScrollEdgeBarExtent` | our own reserved strip |
| `icons/lucide` | `src/icons.go`, ~22 inline SVG paths |
| `theme.Onevra` | `src/palettes.go` schemes over `ui.LightTheme`/`DarkTheme` |

## 3. Motion (60 fps, and 240 Hz)

Rule: **animate paint-only where the frame can.** MyGo's `AnimatePose` runs
transforms and opacity in frames that only paint — the view is not rebuilt. That
is the only way motion stays smooth at 240 Hz, because a rebuilt view costs a
layout pass per frame.

| Motion | Mechanism | Frames |
|---|---|---|
| page change | `ElementTransition{Animation:&Smooth, Enter/Exit}` | build |
| card hover lift | `AnimatePose{Y:-2, Scale:1.004}` | **paint** |
| row hover | `AnimateColor` on the row's face | build |
| transport press | `AnimatePose{Scale:0.94}` | **paint** |
| play ⇄ pause glyph | `AnimatePose{Scale, Rotate}` + crossfade | **paint** |
| artwork change | crossfade of two bitmaps | build (2 layers) |
| now-playing bars | `Loop` on four bar heights | build (4 elements) |
| sidebar collapse | `Animate` on the rail width | build |
| toast | MyGo's own | build |

All of it stills under `Preferences.ReduceMotion`; `Loop` keeps going because it
shows that something is happening.

## 4. States the pages were missing

- **Reading a playlist** — a skeleton of rows where the list will be, not just a
  spinner under the field.
- **Pasting** — the field shows a Paste button while empty, and the Play button
  enables only when the text parses as a playlist link.
- **Playing** — the player bar's play button becomes a pause mark, the now-playing
  row grows live bars, the bar's title gets the track's cover colour.
- **Buffering** — the bar says "Finding the track…" instead of looking idle.
- **Failed** — the bar keeps the track's name and offers Try again.
- **Empty** — unchanged copy, but drawn with the new empty-state widget.

## 5. Playback: why the music stopped early

Three causes, all in `internal/ffmpeg` and `internal/player`:

1. **`-xerror`.** ffmpeg aborts the whole process on the first recoverable HTTP
   error. googlevideo drops connections mid-stream routinely. Removed.
2. **No reconnect.** ffmpeg was told nothing about retrying a streamed HTTP
   source. Added `-reconnect 1 -reconnect_streamed 1 -reconnect_delay_max 5`.
3. **A clean EOF was treated as a broken stream.** The engine only knew
   "exhausted", so a track whose audio is shorter than its metadata (YouTube
   rounds `lengthSeconds`) was recovered three times — up to 3 × (resolve + 1.5s)
   of silence — before moving on. The decoder now reports a clean end apart from
   a failure, and a clean end ends the track.

## 6. Mixes

Already worked (`docs/mixes.md`), and verified again against the live service:
`yt-dlp --flat-playlist -J` on an `RDAMVM…` link returns the entries. One
improvement: yt-dlp reports `channel`/`uploader` per entry, so mix tracks can
carry an artist instead of a bare title. That makes them scrobble and read
properly in the queue.

## 7. Verification

- `go build ./...`, `go test ./...`, `go tool mygo vet .`
- new: `src/motion_test.go` — the poses are bounded and every moving element
  stills under Reduce Motion
- new: `internal/player/` — a clean EOF ends the track
- new: `internal/ffmpeg/` — the argument list carries the reconnect flags and no
  `-xerror`
- `src/shots_test.go` still writes every page for the eye
