# The landing page

`site/` is the page at <https://yaffalhakim1.github.io/playlist/>. It is static:
no build step, so what is in the folder is what is served.

```powershell
cd site
python -m http.server 8765     # then open http://localhost:8765
```

## What is there

| Path | |
|---|---|
| `index.html` | the page |
| `styles.css` | the look, with the app's own palette and scale |
| `main.js` | the theme switch and the reveal |
| `assets/` | the screenshots, the mark, the icons, the social card |
| `sitemap.xml`, `robots.txt`, `site.webmanifest` | what a crawler and a phone look for |

## The look is the app's

The colours are `src/palettes.go`'s default palette, not a new set invented for
a web page: `#fcfbf9` page, `#33332f` ink, `#bd481b` accent, and the dark pair
`#211f1c` / `#eeeae2` / `#ed986e`. The scale is the app's too — an 8px base, a
24px inset, one rounding for cards — because a page about a product should look
like the product.

Every pair was computed rather than eyeballed:

| Pair | Ratio |
|---|---|
| ink on page | 12.27:1 |
| muted on page | 4.69:1 |
| accent on page | 4.94:1 |
| white on accent | 5.11:1 |
| dark ink on dark page | 13.70:1 |
| dark muted on dark page | 8.17:1 |

## Two appearances, and the screenshots follow

The switch is in the header, and the choice is remembered in `localStorage`.
Until the reader chooses, the page follows the desktop as it changes; once they
have chosen, their choice wins.

The screenshots swap with the theme. A light shot on a dark page reads as a bug,
so each `<img>` carries `data-light` and `data-dark` and `main.js` sets `src` on
every paint. `<picture>` with `media="(prefers-color-scheme: dark)"` was the
first attempt and it is wrong here: it follows the **system**, not the button,
so a reader who chose dark on a light desktop would get light shots.

## Motion

Two things move, and both are optional.

- The theme icon's fill, 200ms.
- A section comes in from 14px below, once, 600ms, `cubic-bezier(0.23, 1, 0.32, 1)`.

The hidden state lives behind `html.js`, added by an inline script in the head.
Without JavaScript there is no rule at all, so nothing can be left invisible —
which is the failure this pattern exists to avoid. The reveal also shows
anything already above the fold at load, because an anchor jump or a restored
scroll position can land *past* an element without it ever intersecting the
observer.

`prefers-reduced-motion` keeps the fade and drops the movement, at a
specificity that beats the hidden rule.

## What was checked

Lighthouse, on the local server: **100 accessibility, 100 best practices, 100
SEO**, no failures. Getting there found two real bugs:

- **A `<code>` chip failed contrast at 4.21:1.** The tinted background lifts the
  surface enough that muted text falls under 4.5:1. The chip now uses the ink,
  which is 11:1 on the same tint.
- **The header overflowed at 360px.** The sections and the switch could not sit
  beside the name, and the switch was pushed off the edge. The row wraps now,
  and under 560px the sections take a line of their own.

Also checked: every asset the HTML references exists; the page has no horizontal
overflow at 360px; the no-JS path leaves nothing hidden; both themes swap the
screenshots.

## SEO

- `title`, a 228-character `description`, `canonical`
- Open Graph and Twitter card, with a 1280×640 `assets/social.png` rendered by
  `scripts/render-social.ps1` from the real player screenshot
- `application/ld+json` as a `SoftwareApplication`, with the version, the
  platforms, the price and the feature list
- `sitemap.xml`, `robots.txt`, `site.webmanifest`
- One `h1`, and every image with a real `alt`
- `theme-color` per appearance, so the phone's chrome matches the page

## Publishing

`.github/workflows/pages.yml` deploys `site/` on a push to `main` that touches
it. It checks first that every file and every asset reference exists, because a
missing screenshot is a page that 404s on its own picture.

Enable it once: **Settings → Pages → Build and deployment → Source: GitHub
Actions**.
