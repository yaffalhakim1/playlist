# Playlist — the mark

## The idea

A queue is a sequence, so the mark is **one band that turns twice**. A playlist
plays, so the band **arrives at a play mark**. One idea: *the queue arrives*.

The band folds rather than listing: three runs, two 180° turns, no closed
shape. The play mark stands clear of it by a full stroke, because a triangle
that grows out of a band is an arrowhead, and a triangle set beside one is a
play button.

## Files

| File | What it is |
|---|---|
| `b-master.svg` | The mark, black. The master: everything else is made from it. |
| `b-small.svg` | The small cut: thicker stroke, tighter gap, for 16–24 px. |
| `lockup-horizontal.svg` | Mark then name, for headers and wide spaces. |
| `lockup-stacked.svg` | Mark over name, for square spaces and app tiles. |
| `b-master-{black,white,mono}.svg` | One-colour and reversed, exported. |
| `b-master-app-icon.svg`, `icon-*.png` | The app icon on terracotta. |
| `favicon.svg`, `favicon.ico`, `favicon-*.png` | The favicon, from the small cut. |
| `site.webmanifest`, `head-snippet.html` | Web icon set and the tags to paste. |

## Colour

One colour, used flat. The mark never carries a gradient.

| Role | Value |
|---|---|
| Ink | `#000000` on light, `#FFFFFF` on dark |
| Brand | `#BD481B` — the app's own terracotta accent |

The mark works in one colour, reversed, and on the brand terracotta. It does
not work on a mid-tone without a backing: the gap between the band and the
play mark closes when they are the same value as what is behind them.

## Clear space and minimum size

**Clear space** is one stroke — the band's thickness — on every side. Nothing
enters it: no text, no rule, no edge of a card.

**Minimum sizes.** The full mark is used from 24 px up. Below that, use the
small cut, and below 16 px the mark is not used at all: a 16 px tray icon is
the floor.

| Where | Which file |
|---|---|
| 24 px and up | `b-master.svg` |
| 16–24 px, favicon, tray | `b-small.svg` |
| App icon, 512 px | `b-master-app-icon.svg` |

## Type

The name is set in **Plus Jakarta Sans SemiBold**, tracked −0.024 em at display sizes and
−0.018 em in the stacked lockup. The same face and tracking the app uses, so
the mark and the interface are one family.

The wordmark is live text in the lockup files, not outlines. Before any use
that must not depend on a font being installed — print, a press kit, a
trademark filing — convert the text to outlines, or ask for the outlined
version. Plus Jakarta Sans is under the SIL Open Font License, which allows this.

## Misuse

- Do not close the band into a loop, or drop one of its folds.
- Do not touch the play mark to the band: the gap is the idea.
- Do not rotate, skew or outline the mark.
- Do not put the mark on a photograph without a solid backing.
- Do not recolour the band and the play mark differently.
- Do not set the name in another face, or add a tagline to the lockup.

## Honest notes

- **At 16 px the full mark's two folds collapse**; the small cut exists for
  that reason and is not interchangeable with the master.
- **Mirrored, the mark reads as a "2"**, and rotated 180° it is not
  meaningful. It is designed to sit the right way up.
- The mark is not trademarked. A clearance search is the owner's step, and
  nothing here substitutes for one.
