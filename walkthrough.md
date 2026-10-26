# Home page redesign — what was built

The design came in as HTML/CSS. This is what it became in MyGo, and where it
departs from the mock.

## The decisions you made

| Question | Answer | What it means |
|---|---|---|
| Fake vs real sections | **Map to real data** | Nothing on screen does nothing |
| Sidebar items | **Keep real pages, restyled** | Five destinations, not the mock's seven |
| Font | **Embed Plus Jakarta Sans** | Four weights in `src/assets/fonts` |
| Colour | **Replace the palette system** | Monochrome paper and ink, no picker |

## Layout

```
┌─────────────┬──────────────────────────────────────────────┐
│  ◀  ○ ○ ○    │  Playlist                                     │  header
├─────────────┼──────────────────────────────────────────────┤
│ Home        │  ┌────────────────────────────────────────┐  │
│ Player      │  │ 🔗 link…                  [Extract]    │  │  import row
│             │  └────────────────────────────────────────┘  │
│ LIBRARY     │  ┌────────────────────────────────────────┐  │
│ Listen Now  │  │ [cover]  NOW PLAYING FROM •  ⋯         │  │
│ History     │  │          Playlist name                 │  │  hero
│ Appearance  │  │          n tracks · length · artist     │  │
│             │  │          (▶) Open queue  Start a mix    │  │
│ PLAYLISTS   │  └────────────────────────────────────────┘  │
│ Tokyo Rain  │  ┌────────────────────────────────────────┐  │
│ Calm        │  │ Title   Artist   Album/Tape   Duration │  │  table
│ Synthwave   │  │ ▮ Kingdom in Blue …                    │  │
│             │  │ 2 phantasm …                           │  │
│ [cover]     │  │                       All 40 tracks →  │  │
│ title/artist│  └────────────────────────────────────────┘  │
│             │  Recently played   (covers, sideways)         │
│             │  Your playlists    (3 wide cards)             │
└─────────────┴──────────────────────────────────────────────┘
      rail                          page
┌────────────────────────────────────────────────────────────┐
│ 1:04 ─────────────●─────────────────────── 4:13            │  footer
│ [cover] Title      ⤨ ⏮ (▶) ⏭ ⟳          🔊 ────           │
└────────────────────────────────────────────────────────────┘
```

## The Player page

```
  ‹ Playlist                                                    breadcrumb

  ┌───────────────────────┐   ┌───────────────────────────┐
  │                       │   │ Up next              1718 │
  │        cover          │   │ ┌───────────────────────┐ │
  │      (square,         │   │ │[c] Kingdom in Blue ▮  │ │
  │    warm shadow)       │   │ │    Kupla         16:25 │ │
  │                       │   │ └───────────────────────┘ │
  └───────────────────────┘   │ [c] In another life  3:51 │
    Kingdom in Blue (Live…)   │ [c] That Should Be Me 3:37│
    Kupla • Kingdom in Blue EP│ [c] When I Was Your Man   │
                              │ [c] See You Again    3:58 │
    2:15 ─────●─────── 16:25  │ [c] Night Changes    4:01 │
      ⤨ ⏮ (⏸) ⏭ ⟳           │ [c] To The Bone      5:54 │
                              │ [c] Die With A Smile 4:13 │
       (📡 Start radio)       │                           │
  └───────────────────────┘   └───────────────────────────┘
  ┌────────────────────────────────────────────────────────┐
  │ 2:15 ─────────●────────────────────────── 16:25        │  footer
  │ [cover] Kingdom in Blue    ⏮ (⏸) ⏭    ⤨ ⟳ 🔊 ────     │
  └────────────────────────────────────────────────────────┘
```

The left column is a poster on one axis: cover, name, timeline, transport, and one
offer under it. The right is the queue in its own panel, opening on the track that
plays and following it as the music moves. The footer is back on this page too —
see the bugs below for why it was wrong to remove it.

## The Appearance page

```
  PREFERENCES / APPEARANCE                                      breadcrumb
  Appearance
  The visual tone of the window, the colour it grows from, …    lede

  ┌──────────────────────────────────────────────────────────┐
  │ Interface theme                                          │
  │ Match the Windows theme, or anchor the window …           │
  │ ┌──────────┐ ┌──────────┐ ┌──────────┐                  │
  │ │ ▪▪▪ ───  │ │ ▪▪▪ ───  │ │ ▪▪▪ ───  │   miniature of   │
  │ │ ▭  ▭▭▭   │ │ ▭  ▭▭▭   │ │ ▭  ▭▭▭   │   each theme     │
  │ └──────────┘ └──────────┘ └──────────┘                  │
  │ Light      ✓  Dark         Auto                          │
  │ Daylight…     Charcoal…    Windows…                      │
  └──────────────────────────────────────────────────────────┘

  ┌──────────────────────────────────────────────────────────┐
  │ Accent colour                                            │
  │ ┌──────────────────────────────────────────────────────┐ │
  │ │ ● Charcoal ● Terra Cotta ● Sage ● Slate ● Ochre      │ │
  │ └──────────────────────────────────────────────────────┘ │
  │ ┌──────────────────────────────────────────────────────┐ │
  │ │ 🎨 Dynamic artwork radiance              [ toggle ]  │ │
  │ └──────────────────────────────────────────────────────┘ │
  └──────────────────────────────────────────────────────────┘

  ┌──────────────────────────────────────────────────────────┐
  │ Window & materials                                       │
  │ Sidebar acrylic & Mica      [Subtle][Standard][High]     │
  │ ──────────────────────────────────────────────────────── │
  │ Reduce transparency                              [ off ] │
  └──────────────────────────────────────────────────────────┘

  ┌──────────────────────────────────────────────────────────┐
  │ Sharing                                                  │
  │ Discord                                          [ off ] │
  └──────────────────────────────────────────────────────────┘
```

The three theme previews are **drawn, not pictured**: a miniature with the window
dots, a rail, a page with a cover and a half-played scrubber. Drawn in the palette,
they stay correct in either appearance — a screenshot of the dark theme would be a
light-coloured image on a light page.

### What the settings actually do

| Setting | What it changes |
|---|---|
| Light / Dark / Auto | `mygo.Theme.SetSource`, so the window chrome follows too |
| Accent tone | `Theme.Accent`, and everything derived from it: the scrubber fill, the play button, selection, focus |
| Dynamic artwork radiance | reads the cover's dominant hue, fits it to a target luminance, and uses that as the accent |
| Subtle / Standard / High | `mygo.VibrancyTabbed` / `VibrancyMica` / `VibrancyAcrylic` |
| Reduce transparency | `VibrancyNone` — the view paints its own face instead |

**The material only shows where the view paints nothing.** That is MyGo's rule, not
a choice: `c.Vibrancy()` reports whether a material is *actually* showing, and when
it is the page leaves its face transparent so the desktop's backdrop comes through,
while the rail keeps a 72% veil so its text still reads. On a desktop without
materials it reports false and both paint the paper.

**A window must be born with a material to change it later.** On Windows,
`SetVibrancy` changes the material of a window created with one and shows none in
others — so the window is created with `materialOf(...)` and never without.

## The Playlist page

```
  ┌────────┐   PLAYLIST • Synced from YouTube
  │        │   Tokyo Rain Sessions                    ← display type
  │ cover  │   RadioSync Studio • 18 tracks • 1 hr 12 min
  │  48    │   (▶ Play all)  (⟳ Re-sync)  (⋯)
  └────────┘

  (All tracks)                              [🔍 Filter playlist…]

  ┌──────────────────────────────────────────────────────────┐
  │ #    TITLE              ARTIST          ALBUM       🕐    │
  ├──────────────────────────────────────────────────────────┤
  │ ▮▮▮  [c] Midnight in Shibuya   Kazumi…  Tokyo Jazz  4:12 │  ← plays
  │ 2    [c] Raindrops on Shinjuku Uyama…   A Son of…   3:45 │
  │ 3    [c] Warm Tea at 2AM       Nujabes  Metaphor…   3:28 │
  └──────────────────────────────────────────────────────────┘
```

The head is a poster: the cover is the largest thing on the page and the name is
second, because a playlist is recognised before it is read. The table below is a
virtualised `ui.List` with a heading row that stays put while the tracks scroll
under it — a thousand tracks draw the rows in view and no others.

### The controls are real

| Control | What it does |
|---|---|
| **Play all** | starts the queue from the top |
| **Re-sync** | drops the cached tracks and reads the playlist from YouTube again |
| **Filter playlist…** | narrows the rows by title, artist or album, as you type |
| **All tracks** | the whole playlist, which is the only filter there is |

`Re-sync` is `Store.ForgetCache` plus `load()`: it is a re-read, not a forget, so
the playlist stays in the rail and the session is untouched.
`TestReSyncReadsThePlaylistAgain` fails if the cache survives it.

The **Downloaded** chip is gone. The app streams — nothing is ever downloaded — so
a filter for it could only ever be empty. `TestThereIsNoDownloadedFilter` pins that.

## Mapped to real data

| The mock's section | What it is here |
|---|---|
| Import bar + "Extract Playlist" | The link field; the button reads the clipboard when empty, loads when it holds a link |
| Hero ("Lofi Girl Radio") | **The playlist that is open** — its cover, name, track count, length, and what plays |
| Tracklist table | **The queue's first six rows**, with `All n tracks →` for the rest |
| "Popular YouTube Radios" | **Dropped.** The app has no catalogue of radios, so cards for them would be fiction |
| "Recently Extracted Playlists" | **Recently played** — the covers shelf, then three wide cards with a play button |
| Sidebar playlists | The real recents, as names |
| Sidebar mini player | What plays, with its cover |
| Search field | **Removed**, as you asked |
| Round "W" avatar | **Removed**, as you asked |

## The theme

`src/theme.go` is now one scheme in two appearances, not eight palettes:

| Role | Light | Dark |
|---|---|---|
| page | `#FAF9F5` | `#131312` |
| card | `#FFFFFF` | `#1C1C1B` |
| rail | `#F5F4F0` | `#171716` |
| sunken | `#EFEEEA` | `#242322` |
| line | `#EAE8E3` | `#2E2D2B` |
| ink | `#1B1C1A` | `#EDEBE6` |
| muted | `#787774` | `#9A9791` |
| filled button | `#1C1C1A` | `#EDEBE6` |

The accent is the near-black of a filled button. There is no hue anywhere, so the
artwork is the only colour in the window. `TestTheThemeIsPaperAndInk` fails if a
hue ever creeps back in.

Gone with the palettes: `src/palettes.go`, `src/color.go`, the accent swatches,
the palette swatches, the "colour from artwork" switch, and the dominant-colour
reader in `src/artwork.go`.

## Bugs found and fixed while building this

**The rail toggle vanished with the rail.** It was drawn inside the rail's own
row, so closing the rail closed the only way to open it — the window was stranded.
It is now drawn at the window's edge whether the rail is open or closed.
`src/rail_test.go` pins it: three tests, including one that clicks it while the
rail is closed.

**`setMode` hung the whole test suite.** `mygo.Theme.SetSource` waits on the
desktop's main thread; with no app running it never returns, and the suite timed
out after ten minutes. The state change and the desktop call are now separate, and
the desktop call only happens when there is a window.

**The Discord switch never showed its state.** Its field was never seeded before
the page drew, so the knob always read off. Fixed, with a test.

**The footer was removed from the Player page, and that was wrong.** The argument
was that the page already had a transport, so a bar under it would repeat it. Two
things were wrong with that: the page's transport cannot say what the footer says
— "Finding the track…", "That track will not play", the retry — and a page that
hides the only status line looks frozen when a track takes a moment. The bar is
now on every page, and `TestEveryPageCarriesTheFooter` fails if one hides it.

**Cards blurred their own text on hover.** A pose with a `Scale` is a real
transform, so MyGo resampled the whole card — words included. Cards no longer move
at all; a tile that is *itself* a button rises by translation only, which MyGo
folds into untransformed content so text stays sharp.

## What was removed

- the Playlist page (a playlist is the Queue page)
- the Shuffle setting (the transport has the button)
- the Last.fm settings page and its tests (scrobbling still runs from stored settings)
- the search field and the avatar
- the "Windows 11 Edition v2.4" badge
- `src/lastfm_page.go`, `src/page_playlist.go` (its parts moved to
  `src/playlist.go`), `src/lastfm_stub_test.go`, `src/palettes_test.go`,
  `src/paletteshot_test.go`

## What came back

The Appearance page brought three things back that the monochrome pass had
deleted:

- **`src/color.go`** — reading a cover's dominant hue, and fitting it to a
  luminance the accent needs.
- **the hue in `src/artwork.go`** — read from the same bytes as the bitmap, so a
  cover is still decoded once.
- **the accent in `src/theme.go`** — laid over the paper, with its hover, pressed,
  ink, selection and focus all derived from it.

## Verification

```
go build ./...        ok
go vet ./...          ok
go tool mygo vet .    ok
go test ./...         ok
```

`go tool mygo dev` builds and runs the window.
