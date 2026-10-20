# playlist — development guidance

A native desktop player for YouTube playlists: paste a link, and it plays the
music of it. Built with [MyGo](https://mygo.egoist.dev) (native UI, no
webview) and [Onevra](https://oneo-me.github.io/onevra/) for the controls, in
Onevra's warm palette and Inter, laid out by Apple's rules for space, type and
motion.

## The three programs this app leans on

| Program | What it does | Where |
|---|---|---|
| InnerTube | reads the playlist: title, artist, album, artwork, length | `internal/innertube` |
| yt-dlp | finds the audio of one track, and reads the mixes YouTube generates | `internal/ytdlp` |
| ffmpeg | decodes the stream into PCM for the player | `internal/ffmpeg` |

All three are needed to play. The app says so plainly when one is missing.

## Layout of the code

The root holds only the entry point; every other Go file lives in `src/` (one
package) or `internal/` (one package each).

| Path | Responsibility |
|---|---|
| `main.go` | entry point only: calls `src.Run()` |
| `src/app.go` | the state, the window, and what the app does |
| `src/shell.go` | the column at the left, the page frame, and the player bar's room |
| `src/shellview.go` | the shell's layout without a window, so the tests draw what the window shows |
| `src/page_home.go` | the link, and the playlists played before |
| `src/page_playlist.go` | the cover, the head, the songs |
| `src/page_player.go` | the big player, the queue beside it, and the queue page |
| `src/page_stats.go` | the totals, the chart, the tracks played most |
| `src/page_appearance.go` | the theme, the accent, the artwork colour, the queue |
| `src/playerbar.go` | the transport, the place in the track, the volume |
| `src/widgets.go` | the type roles and the pieces every page is built from |
| `src/theme.go` | the look: Onevra's palette, the accent, the space, the type, the motion |
| `src/surfaces.go` | the few faces the app paints, named in the theme's own terms |
| `src/assets.go` | the typeface, embedded so the app looks the same everywhere |
| `src/artwork.go` | the pictures of the albums |
| `src/shortcuts.go` | the keys of the player |
| `src/keys.go` | the shortcuts of the system: media keys, and one to show the window |
| `src/tray.go` | the icon in the notification area, so the music outlives the window |
| `internal/music` | the domain model: `Track`, `Playlist` |
| `internal/innertube` | the InnerTube client and the reading of its answer |
| `internal/ytdlp` | yt-dlp: the audio of a track, and the playlists InnerTube cannot read |
| `internal/ffmpeg` | the decoder that feeds MyGo's audio package |
| `internal/player` | the engine: the queue, shuffle, repeat, autoplay |
| `internal/playlist` | the loader that picks the reader that knows the playlist |
| `internal/library` | what the app remembers between runs |

## Rules

### Check the docs first, never assume

- **MyGo**: https://mygo.egoist.dev/docs/ui — read the page for a component
  before using it, and check `go doc github.com/egoist/mygo/ui <Name>` for the
  exact signature of the pinned version.
- **Onevra**: `go doc github.com/oneo-me/onevra/source/components <Name>`.
- **yt-dlp**: `yt-dlp --help`, and `-v` to see what it does. The version on
  this machine must be recent: an old one hands out links that YouTube refuses
  to serve after the first megabyte.

### Use the framework's components

MyGo ships 95 UI constructors and Onevra a library on top of them. Before
writing a widget, check the list. Hand-rolled replacements lose focus
handling, keyboard navigation and accessibility.

| Need | Use |
|---|---|
| rows that scroll | `components.List` |
| text input | `components.TextBox` |
| buttons | `components.Button`, `components.IconButton` |
| the place in a track | `components.Slider` |
| pictures | `ui.Image` with a `*ui.Bitmap` |
| messages | `c.AddToast` |

### The view is a function of the state

The view is pure: it draws the state and nothing else. Change the state from
a goroutine with `win.Update(func(){...})`, which runs on the main thread and
redraws. Never store a `ui.Element` or a `*ui.Context` in the state: they
belong to one frame.

The engine reports what happens on a channel; `app.listen` takes it and
applies it on the thread that draws.

#### The lock is not reentrant

`a.mu` guards the state the engine writes. Go's mutex cannot be taken twice by
the same goroutine, so **a method that holds it may not call one that takes
it again**: the second `Lock` never returns, the thread that draws waits on
itself, and the window freezes — Windows shows "Not Responding" and the app
never recovers.

This froze the window the moment a track started: `apply` held the lock and
called `a.tray.refresh()`, which asked the app what was playing, which locked
again. The tray is now *told* the track and the state, and reads them from the
caller's own copy (`trackLocked`).

When you add something to `apply`, or to any other method that locks:

- read the state through a `*Locked` helper that assumes the lock is held
  (`trackLocked`), never through the locking reader (`currentTrack`);
- hand the callee what it needs as arguments instead of letting it ask;
- do work that reaches the desktop (a tray call, a window call) outside the
  lock, or on a goroutine, because it may wait for the desktop.

`TestApplyingATrackDoesNotDeadlock` runs that path on a deadline, and
`TestNoMethodLocksTwiceInOnePath` reads the source and fails on any method that
locks and then calls a method that locks again.

### Sizing

`Fill()` sets the width *and* the height, so it overrules a `Width` or a
`Height` set before it. Write the size last, or use `FillWidth()` and
`FillHeight()` for the one direction that should fill. `Grow(1)` inside a
column makes a child take the height that is left: that is how the list grows,
not how a button does. A `Box` lays its children out from the top left: call
`Center()` on it when it holds an icon that should sit in the middle.

### The look

The app is Onevra's, not its own. `theme.Onevra` supplies every colour, and
`src/theme.go` only lays the user's accent over that palette — hover, pressed,
text-on-accent, selection and focus are all derived from the accent, never
written by hand. `src/surfaces.go` names the few faces the app paints in the
theme's own terms (`card` is `Surface`, `raised` is `SurfaceHover`, `sunken` is
`Background`, `muted` is `TextMuted`, `chosen` is `Selection`); do not add a
colour that is not a role of the theme.

The layout follows Apple's rules for space, type and motion, which is what
makes the window read as a product rather than as a pile of widgets:

- **Space.** Every distance is `t.Space(n)` and every size `t.Rem(n)`, so the
  whole app follows the theme's spacing and the desktop's text scale. A page
  keeps one inset (`pageInset`) and one gap between its groups, and the header
  uses the same inset so an action lines up with the page it acts on.
- **Type.** Four sizes and three weights, nothing else: `remDisplay`,
  `remHeadline`, `remTitle`, `remBody`, `remCaption`, `remEyebrow`, with 600 for
  names, 500 for labels, 400 for text. Large text is tracked tight and small
  text open (`trackDisplay`, `trackHeadline`, `trackEyebrow`), and leading
  follows size the other way (`leadDisplay` … `leadBody`). Never set a size
  alone: use the role helpers in `src/widgets.go` (`pageName`, `sectionName`,
  `itemName`, `body`, `caption`, `eyebrow`), which carry size, weight, tracking
  and leading as one decision.
- **Motion.** `motionReveal` brings a page in from a little below; `motionPress`
  settles a control into the face it takes under the pointer. Nothing else
  moves. Both are springs, and MyGo stills them when the desktop asks for less
  motion.

A card around every section of every page is not the way: a page keeps its
lists on the page itself, and only a group that holds settings, a chart or a
summary gets a card. A card is `card()` in `src/widgets.go`: one rounding, room
inside, and a shadow too faint to notice instead of a border.

### The typeface

The app draws in Inter, embedded in `src/assets.go` so a machine that never
installed it looks the same. Three weights are registered under one family; the
theme asks for `familySans`, and falls back to the system's own font if the
desktop refuses the file. `TestFontInUse` draws the same words in the app's
family and in one that does not exist, and fails if the two pictures match —
that is what catches a font that silently did not load.

### Use Onevra's components, not a copy of them

The window is `shells/sidebar`, the navigation is `components.Sidebar` with
`SidebarSection`, `SidebarItem` and `SidebarImageItem`, the page scrolls in
`components.ScrollView`, a long list is `components.List`, and the player bar
keeps its own strip at the bottom of the window through `ScrollEdgeBarExtent`
so nothing the page holds ever passes under it.

`shells/sidebar` draws its header over the full width of the window and
reserves nothing above the sidebar, so the sidebar passes `HeaderHeight(c)`
as its own top padding — otherwise its first heading hides under the window
controls.

The Look page is built from the same cards, settings rows and controls as the
other pages rather than from `components.Settings`: the app already has a
sidebar, and a second navigation beside it would be one too many.

### The tests of the look

The look is checked, not just drawn. `src/design_test.go` reads the source and
fails on a page that writes a size or a colour by hand, so the scale stays one
scale; `src/layout_test.go` draws the pages and fails on anything that runs
under the header or the bar, off the column, or out of step with its neighbours;
`src/shots_test.go` writes the pictures of every page, light and dark, small and
wide, loaded and empty, for a person to look at.

```powershell
go test ./src/ -run 'TestDrawPages|TestDrawEmptyPages|TestDrawStats'   # the pictures
go test ./src/ -run 'TestPagesUseTheTypeScale|TestPagesUseTheThemeColours'  # the rules
```

### Keeping the music going

Closing the window hides it: the app lives in the notification area, where
the menu plays, holds and skips, and a click brings the window back. Quitting
from that menu is the only way to end the app. The media keys of the keyboard
are registered with `mygo.GlobalShortcut` and work from any window.

### What the app remembers

`internal/library` keeps three things beside the settings, and each one exists
to save the user a wait:

- **The playlists it has read** (`Cached`), so opening one again needs no
  network at all. The link is the key; the tracks, their artists and their
  lengths come back whole. Ten are kept, and one too large to keep is left out
  rather than crowding the file, because the file is read and written as the
  music plays.
- **Where the player was** (`Session`): the playlist, the track, and the place
  in it. On the next run the app opens that playlist from the cache and seeks
  to that place, so a track carries on instead of starting the queue again.
- **What played** (`Plays`), for the stats page.

The session is written when a track changes, and every `sessionInterval` as the
track plays: the engine reports the place five times a second, and a file write
that often costs more than the second of accuracy it buys.

`restoreSession` and `showPlaylist` read the cache before they read the network,
and `playlist.Load` is reachable only from `load()` after a cache miss. Two
tests pin that: `TestOpeningACachedPlaylistNeverLoads` and
`TestRestoringNeverLoads` fail if a cached open bumps the load generation, which
is what happens only when something is about to go out.

### Threads

Keep I/O off the thread that draws. Guard the state the engine writes with the
mutex; the fields only the interface touches need no lock.

## Verification

```powershell
go build ./...            # compiles
go tool mygo vet .        # Go vet plus MyGo's checks on the life of the UI
go test ./...             # the tests of the pure logic and of the views
go test -tags e2e ./...   # the checks that reach YouTube: slow, and real
go tool mygo dev          # runs the app and rebuilds it on every change
go tool mygo build        # packages the app in build/
```

`go test -tags e2e` reads real playlists and decodes real audio. It is the
check that says the app really plays.
