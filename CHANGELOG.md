# Changelog

Every release, what it changed, and which commit it was. The versions are tags in
this repository; `git log v0.5.0..v0.6.0` shows the work behind each one.

For the bugs themselves — the symptom, the cause, the fix, and the test that
pins it — see [`docs/bugs.md`](docs/bugs.md).

---

## 0.6.0

Last.fm, and a mix from the track that is playing.

### Added

- **Last.fm scrobbling.** Connect an account in **Look → Last.fm**; the app
  sends what is playing now, and a scrobble once half the track or four minutes
  has been heard. Plays that cannot be sent wait on disk and go out later.
  `docs/lastfm.md`
- **Start a mix** on the player. Builds the mix YouTube generates from the
  playing track: no account needed, because a mix is built from a video id
  rather than from a listening history. `docs/mixes.md`
- A test that fails if any test file builds a Last.fm client without pointing it
  at a stub, after one such test called the real API in CI.

### Fixed

- **A stream that broke mid-track stopped the music for good.** It now finds the
  stream again and plays on from where it stopped, rather than skipping the
  track or hanging. `docs/playback.md`
- **Saving a Last.fm key looked like nothing happened.** The key is now checked
  with Last.fm before it is kept, the result is shown, and the secret leaves the
  fields once it is saved.

### Commits

`d18d871` Last.fm · `4a58a8e` stream recovery · `b808993` mixes · `59070f4`
offline tests · `3b835ae` the release

---

## 0.5.0

An app you can install, instead of an executable you have to know what to do
with.

### Added

- **Discord Rich Presence**: the track, the artist, the album as a second line,
  and a progress bar. `docs/discord.md`
- **Wrapped-style stats**: one statement about the period, three facts, and a
  chart as the evidence under them.
- **Material 3 palettes**: eight schemes, light and dark, with the accent laid
  over the chosen one.
- Installers instead of archives: NSIS on Windows, `.dmg` on macOS, `.deb` and a
  tarball on Linux. `docs/shipping.md`

### Fixed

- **A terminal opened beside the app, and the taskbar had no icon.** The release
  used `go build`, which produces a console-subsystem executable with no icon
  resources. It uses `mygo build` now.
- **A console flashed beside the player on every track**, because a GUI app that
  starts `yt-dlp` or `ffmpeg` makes Windows open one for the child.
- **A track ended a second early.**
- **Two releases sat queued for over two hours** and never published: `macos-13`
  is a retired runner label.
- **The Windows job failed with exit 127**: the workflow called `zip`, which is
  not on a Windows runner.
- **The Discord panel had no progress bar**, and no line for the album.
- **A deadlock froze the window** the moment a track started.

### Commits

`34d3068` palettes and tray · `b9f7b8a` Discord · `fbfc259` the wrap · `a08dab9`
progress bar · `615436b` the release · `e860aea` the app itself · `f04d238`
release file names

---

## 0.4.0

### Added

- Deleting a playlist from the sidebar, with a confirmation.
- Session restore: the playlist that was open, and the place in the track.

### Fixed

- The tray icon did not fill its bitmap, and the playing and paused states were
  indistinguishable.
- The most-played list flickered, because its sort was not a total order.
- Up next opened on the first track rather than the one playing.

### Commits

`d3ce746` delete · `1080b2e` session · `b2c7354` the mark

---

## 0.2.1 and 0.2.0

Tagged, and never published: both runs are stuck in the queue to this day. The
cause is in 0.5.0's list above. Nothing was lost; they had published nothing.

---

## 0.1.0

The first version: a native player for YouTube playlists, in Inter and Onevra's
palette. `eefa5d7`

---

## Not shipped, and why

- **Discord artwork does not render.** Discord fetches an image from a URL, and
  the thumbnails this app holds are YouTube's, which Discord will not display.
  It needs assets uploaded to the Discord application instead. `docs/discord.md`
- **`go test -race` cannot run on the machine this was built on**: there is no C
  toolchain for the race detector. The concurrency guards are ordinary tests
  instead, and `TestNoMethodLocksTwiceInOnePath` reads the source for the
  deadlock that started it.
- **Browsing the mixes on your own YouTube Music home page.** That is the
  personal feed, it needs a signed-in Google session, and it is a different
  feature from starting a mix from a track. `docs/mixes.md`
