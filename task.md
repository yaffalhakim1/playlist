# Task list

Status: `[ ]` todo · `[~]` in progress · `[x]` done

## 1. MyGo only — done
- [x] `github.com/egoist/mygo` v0.5.5 → **v0.6.4**
- [x] **Onevra removed entirely**: one dependency now, not two
- [x] `src/icons.go`: the 28 icons the app uses, as inline SVG
- [x] `src/theme.go`: MyGo's own light and dark themes, in the app's palette
- [x] `src/widgets.go`: the app's buttons, cards and type roles on MyGo's bases
- [x] `src/shellview.go`: the shell layout is now the *only* view, not a copy
      of one a library owned
- [x] every page moved from `components.*` to `ui.*`
- [x] `go mod tidy`: `github.com/oneo-me/onevra` is gone from `go.mod`

## 2. Motion — done
- [x] `motionPress` / `motionLift` / `motionColor` / `motionFade`, all ≤300 ms
- [x] **paint-only where possible**: `AnimatePose` on cards, buttons, the
      transport and the shelf, so a moving frame costs a transform and not a
      layout pass
- [x] `AnimateColor` for every interactive face (a `Transition{Colors}` holds a
      stale face for a frame — found by drawing the dark theme)
- [x] a page arriving still uses `motionReveal`
- [x] artwork crossfades in
- [x] the playing row carries four bars that move with the music
- [x] the sidebar rail opens and closes by animating its own width
- [x] everything stills under Reduce Motion (`TestMotionStillsUnderReduceMotion`)

## 3. States — done
- [x] **pasting**: the field offers Paste while empty, Play once it holds a link
- [x] **reading**: a busy button that keeps its words readable, and a skeleton
      of the page where the list will be
- [x] **playing**: pause glyph, live bars, the track named in the bar
- [x] **buffering**: "Finding the track…" with a spinner
- [x] **failed**: "That track will not play" with **Try again**
- [x] **empty**: the same copy, drawn by the new empty-state widget

## 4. Playback — done
- [x] **`-xerror` removed**: one recoverable HTTP error no longer aborts ffmpeg
- [x] **reconnect flags**: ffmpeg reopens a streamed source that goes quiet
- [x] **a clean end is not a broken stream**: the decoder reports `Finished()`,
      and the engine ends the track instead of "recovering" it three times
- [x] the offset is an input option (`-ss` before `-i`)
- [x] `TestARealTrackPlaysToItsEnd` (e2e): a real track played to 3:06 of 3:07
- [x] verified live in the running app: a track played out and the queue moved on

## 5. Mixes — done
- [x] verified against the live service: `RDAMVM…` links still read
- [x] **mix tracks now carry an artist**, from the channel yt-dlp reports:
      528 of 528 on a real mix, where it was 0 before
- [x] `TestAMixFromATrackLoads` (e2e) checks ids, the seed track, and artists

## 6. Verify
- [x] `go build ./...`
- [x] `go vet ./...`
- [x] `go test ./...`
- [x] `go test -tags e2e ./...` on the paths that changed
- [x] the app runs: launched, loaded a real 182-track playlist, played it
- [x] every page drawn and looked at, light and dark, small and wide

## Left to do
- [ ] the OS media panel (Windows SMTC, macOS Now Playing, MPRIS)
- [ ] an audio cache on disk, so a track played twice costs one network trip
- [ ] gapless playback: start the next track before this one ends
