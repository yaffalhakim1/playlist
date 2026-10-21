# Shipping the app

The app is built by `mygo build`, never by `go build`. That one difference is
what decides whether the download is an app or a loose executable.

## What `mygo build` does that `go build` does not

| | `go build` | `mygo build` |
|---|---|---|
| Windows subsystem | console | **GUI** — no terminal beside the window |
| Windows icon resource | none | the icon, 16 to 256 px, which the taskbar reads |
| Windows installer | none | `Playlist Setup <version> amd64.exe` (NSIS) |
| Windows version resource | none | name, version, copyright in the file's Properties |
| macOS | a bare executable | `Playlist.app` with its `.icns`, and a `.dmg` |
| Linux | a bare executable | a `.deb`, a tarball, `install.sh`, a `.desktop` file |

A `go build` of this app is a **console-subsystem** executable: Windows opens a
terminal beside the window, and the taskbar has no icon to show because the
executable carries no icon resource. That is what shipped in v0.4.0, and it is
why the release looked broken in two ways at once.

## The release

`.github/workflows/release.yml` runs `mygo build -platform <os>/<arch>` on each
platform's own runner, collects the installers, and attaches them to the
release. Push a tag to run it:

```sh
git tag -a v0.5.0 -m "…"
git push origin v0.5.0
```

The tag has to match `version` in `mygo.json`: the version in the file names,
in the installer, and in the executable's version resource all come from there.
The workflow fails the build if they disagree, so a release cannot be named
`v0.5.0` while its files say `0.4.0`.

## Why the runners are what they are

- **`macos-15-intel` for Intel Macs.** `macos-13` was retired, and a job on a
  retired label does not fail — it waits forever, which held two releases open
  for hours. The build job has a 25 minute timeout so that cannot happen again.
- **Windows needs no setup for NSIS.** `mygo build` downloads the compiler
  itself on Windows when it is not installed.
- **The macOS builds are ad-hoc signed, not notarized.** They run, but the first
  launch needs right-click, Open. Notarizing needs a paid Apple developer
  account; set `macos.signingIdentity` in `mygo.json` and add the credentials to
  the workflow to do it properly.

## The other window

A GUI app that starts a console program — `yt-dlp`, `ffmpeg` — makes Windows
open a console for the child, so a black rectangle flashes beside the player on
every track even when the app itself is built correctly. `internal/proc` sets
`CREATE_NO_WINDOW` on both, and `TestEveryChildIsStartedHidden` reads the source
and fails on any new call site that forgets it.

## The taskbar icon

`mygo build` embeds the icon in the executable, and Windows uses it when the
window claims none. The app does not rely on that: `src/app.go` calls
`win.SetIcon(appIcon)` with the same PNG at startup, so the taskbar is right even
for `go run` and the dev server, which have no resources at all.

## Checking a build before publishing

```powershell
go tool mygo build
$exe = "build\windows-amd64\Playlist.exe"
$b = [IO.File]::ReadAllBytes((Resolve-Path $exe))
$sub = [BitConverter]::ToUInt16($b, [BitConverter]::ToInt32($b, 0x3C) + 0x5C)
"subsystem = $sub (2 = GUI, 3 = console)"
```

Run the executable and look at the taskbar: the orange mark should be there, and
no terminal window should open.
