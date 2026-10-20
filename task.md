# Task list

Status: `[ ]` todo · `[~]` in progress · `[x]` done

## 1. Scaffold — done
- [x] Verified toolchain, deps (mygo v0.5.5, onevra v0.2026.13), external tools (yt-dlp, ffmpeg)
- [x] Verified InnerTube browse returns playlist metadata with artist, album, length and artwork
- [x] Verified yt-dlp resolves a direct audio URL that YouTube serves whole
- [x] `go mod init`, `mygo.json`, `main.go` → `src.Run()`

## 2. Metadata — done
- [x] `internal/music`: `Track`, `Playlist`
- [x] `internal/innertube`: client (browse + continuation), playlist-id parsing
- [x] `internal/innertube`: response parser (music + plain list items)
- [x] `internal/ytdlp`: flat playlist reading, for the mixes InnerTube cannot show
- [x] `internal/playlist`: the loader that picks the reader that knows the playlist
- [x] tests: fixture-based parsing tests, URL/ID parsing table tests

## 3. Audio — done
- [x] `internal/ytdlp`: stream URL resolution, with a cache
- [x] `internal/ffmpeg`: the decoder that feeds MyGo's audio package
- [x] tests: the decoder reads real samples (e2e)

## 4. Playback engine — done
- [x] `internal/player`: queue, shuffle, repeat, autoplay-next, volume
- [x] tests: the order of the queue, the shuffle, the repeat modes

## 5. Persistence — done
- [x] `internal/library`: the recents and the settings, in `PathUserData`

## 6. UI (native, Onevra's palette) — done
- [x] `src/theme.go`: the look — Onevra's palette, the accent, and one scale
      for space, type, corners and motion
- [x] `src/assets.go`: Inter, embedded, so the app looks the same everywhere
- [x] `src/app.go`: the state, the window, the root view
- [x] `src/shell.go`: the sidebar, the page frame, the header's room
- [x] `src/page_home.go`: the link, and the playlists played before
- [x] `src/page_playlist.go`: the cover, the head, the songs
- [x] `src/page_player.go`: the big player, the queue beside it, the queue page
- [x] `src/page_stats.go`: the totals, the chart, the tracks played most
- [x] `src/page_appearance.go`: the theme, the accent, the artwork colour, the queue
- [x] `src/playerbar.go`: the transport, the place in the track, the volume
- [x] `src/shortcuts.go`: space, the arrows, the keys with control
- [x] the pictures of the albums, fetched and kept, and their colour read
- [x] `ui.NewTester` tests of the views, and tests that draw every page in the
      light and the dark, on a small window, a wide one, and with nothing in it

## 7. Keeping the music going — done
- [x] `src/tray.go`: the icon in the notification area; closing the window
      hides it instead of ending the app
- [x] `src/keys.go`: the media keys, and a key to bring the window back

## 8. The design system, and its guard rails — done
- [x] `src/assets.go`: Inter, embedded, and a test that fails if the app
      silently falls back to the system's font
- [x] `src/theme.go`: one scale for space, type, tracking, leading, corners and
      motion, with the roles named in `src/widgets.go`
- [x] `src/design_test.go`: the scale is ordered, tracking follows size, radii
      are multiples of the theme, and no page writes a size or a colour by hand
- [x] `src/layout_test.go`: pages keep clear of the header and the bar, the
      column centers on a wide window, the header's action lines up with the
      page, list rows agree in height, and the window survives a larger text
      scale, high contrast and reduced motion
- [x] `src/behaviour_test.go`: the accent round trip, what an empty page says,
      and the shelf of playlists that starts a load
- [x] `src/deadlock_test.go`: the freeze that hit when a track started — the
      tray asked the app for its state while the caller held the lock, which
      is not reentrant. Fixed by telling the tray what plays; guarded by a
      deadline test and by a test that reads the source for a double lock

## 9. Verify
- [x] `go build ./...`, `go tool mygo vet .`, `go test ./...`
- [x] `go test -tags e2e ./...`: reads real playlists, decodes real audio
- [x] the app runs: `go tool mygo dev`
- [x] closing the window keeps the process and its music alive
- [x] the look: check the window and put it right — every page drawn and
      looked at, in the light, the dark, and on a small and a wide window
- [ ] `go tool mygo build`: package the app

## Left to do
- [ ] the OS media panel (Windows SMTC, macOS Now Playing, MPRIS on Linux),
      so the track shows in the system's own controls
- [ ] an audio cache on disk, so a track played twice does not go to the
      network twice
- [ ] gapless playback: start the next track before this one ends
