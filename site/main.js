/* Playlist, the landing page.
 *
 * Four pieces of behaviour, and the page is complete without any of them: the
 * theme switch, the menu the burger opens, the words of the statement lighting
 * one at a time, and the accent swatches that repaint the page.
 *
 * The download links carry a working href in the HTML already. This file only
 * makes sure a link that was written without one still goes somewhere real,
 * because a button pointing at "#" is a button that does nothing.
 */

(function () {
  "use strict";

  var root = document.documentElement;
  var reduce = window.matchMedia("(prefers-reduced-motion: reduce)");

  /* --- the theme ------------------------------------------------------- */

  var toggle = document.getElementById("theme-toggle");

  function theme() {
    return root.getAttribute("data-theme") === "dark" ? "dark" : "light";
  }

  function paint(next) {
    if (next === "dark") {
      root.setAttribute("data-theme", "dark");
    } else {
      root.removeAttribute("data-theme");
    }
    try {
      localStorage.setItem("playlist-theme", next);
    } catch (e) {}
    if (toggle) {
      toggle.setAttribute(
        "aria-label",
        next === "dark" ? "Switch to the light theme" : "Switch to the dark theme"
      );
    }
    /* An accent chosen in one scheme has to be refitted for the other, or it
     * disappears into the page it was not chosen for. */
    if (typeof wearAccent === "function") wearAccent();
  }

  if (toggle) {
    toggle.setAttribute(
      "aria-label",
      theme() === "dark" ? "Switch to the light theme" : "Switch to the dark theme"
    );
    toggle.addEventListener("click", function () {
      paint(theme() === "dark" ? "light" : "dark");
    });
  }

  /* --- the menu -------------------------------------------------------- */

  var burger = document.getElementById("burger");
  var menu = document.getElementById("menu");

  function setMenu(open) {
    if (!burger || !menu) return;
    menu.setAttribute("data-open", open ? "true" : "false");
    burger.setAttribute("aria-expanded", open ? "true" : "false");
    burger.setAttribute("aria-label", open ? "Close the menu" : "Open the menu");
    document.body.style.overflow = open ? "hidden" : "";
    if (open) {
      var first = menu.querySelector("a");
      if (first) first.focus();
    }
  }

  if (burger && menu) {
    burger.addEventListener("click", function () {
      setMenu(menu.getAttribute("data-open") !== "true");
    });

    menu.addEventListener("click", function (event) {
      if (event.target.closest("a")) setMenu(false);
    });

    document.addEventListener("keydown", function (event) {
      if (event.key === "Escape" && menu.getAttribute("data-open") === "true") {
        setMenu(false);
        burger.focus();
      }
    });
  }

  /* --- the nav's shadow, once the page has moved ------------------------ */

  var nav = document.getElementById("nav");

  if (nav) {
    var ticking = false;
    var mark = function () {
      nav.setAttribute("data-stuck", window.scrollY > 8 ? "true" : "false");
      ticking = false;
    };
    window.addEventListener(
      "scroll",
      function () {
        if (!ticking) {
          ticking = true;
          window.requestAnimationFrame(mark);
        }
      },
      { passive: true }
    );
    mark();
  }

  /* --- the statement, a word at a time ---------------------------------- */

  /* Each word is wrapped in its own span, and lights when it crosses a line a
   * third of the way up the viewport. The observer watches the words, not the
   * scroll position: a scroll listener would fire on every frame the page
   * moves, and this has to stay cheap on a phone.
   *
   * The text nodes are collected first and wrapped afterwards. Wrapping while
   * walking the tree is what a stack overflow is made of: the spans you create
   * become children of the node you are still iterating, so the walk descends
   * into its own output and never returns. */
  var statement = document.querySelector("[data-tagline]");

  if (statement && !reduce.matches) {
    var texts = [];
    var collect = function (node) {
      for (var i = 0; i < node.childNodes.length; i++) {
        var child = node.childNodes[i];
        if (child.nodeType === 3 && child.textContent.trim()) {
          texts.push(child);
        } else if (child.nodeType === 1) {
          collect(child);
        }
      }
    };
    collect(statement);

    var words = [];
    texts.forEach(function (textNode) {
      var frag = document.createDocumentFragment();
      textNode.textContent.split(/(\s+)/).forEach(function (part) {
        if (!part) return;
        if (/^\s+$/.test(part)) {
          frag.appendChild(document.createTextNode(part));
          return;
        }
        var span = document.createElement("span");
        span.className = "word";
        span.textContent = part;
        words.push(span);
        frag.appendChild(span);
      });
      textNode.parentNode.replaceChild(frag, textNode);
    });

    if ("IntersectionObserver" in window) {
      var lit = new IntersectionObserver(
        function (entries) {
          entries.forEach(function (entry) {
            if (entry.isIntersecting) {
              entry.target.setAttribute("data-lit", "true");
              lit.unobserve(entry.target);
            }
          });
        },
        { rootMargin: "0px 0px -34% 0px", threshold: 0 }
      );
      words.forEach(function (word) {
        lit.observe(word);
      });
    } else {
      words.forEach(function (word) {
        word.setAttribute("data-lit", "true");
      });
    }
  }

  /* --- the reveals ------------------------------------------------------ */

  var reveals = document.querySelectorAll(".reveal");

  if (reveals.length && "IntersectionObserver" in window) {
    var show = new IntersectionObserver(
      function (entries) {
        entries.forEach(function (entry) {
          if (!entry.isIntersecting) return;
          entry.target.classList.add("is-in");
          show.unobserve(entry.target);
        });
      },
      { rootMargin: "0px 0px -8% 0px", threshold: 0.08 }
    );
    reveals.forEach(function (item) {
      show.observe(item);
    });
  } else {
    reveals.forEach(function (item) {
      item.classList.add("is-in");
    });
  }

  /* --- the accents ------------------------------------------------------ */

  /* The app derives hover, pressed, text on the accent and selection from the
   * accent itself, so a tone here has to do the same. Two things follow, both
   * copied from src/color.go: the tone is fitted to the scheme's luminance
   * before it is used, and the ink written on it is chosen from the fill.
   *
   * The fit matters in dark mode. A near black accent on a near black page is
   * an accent nobody can see, so a dark theme lifts every tone to 0.72 and the
   * light theme settles it at 0.38. */
  var swatches = document.querySelectorAll("[data-accent]");

  function channels(hex) {
    var n = parseInt(hex.replace("#", ""), 16);
    return [(n >> 16) & 255, (n >> 8) & 255, n & 255];
  }

  function fit(hex, dark) {
    var rgb = channels(hex);
    var target = dark ? 0.72 : 0.38;
    var lum = (0.2126 * rgb[0] + 0.7152 * rgb[1] + 0.0722 * rgb[2]) / 255;
    if (lum <= 0.01) return hex;
    var factor = target / lum;
    return (
      "#" +
      rgb
        .map(function (v) {
          return Math.max(0, Math.min(255, Math.round(v * factor)))
            .toString(16)
            .padStart(2, "0");
        })
        .join("")
    );
  }

  function readableOn(hex) {
    var rgb = channels(hex);
    var channel = function (v) {
      v /= 255;
      return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4);
    };
    var l =
      0.2126 * channel(rgb[0]) + 0.7152 * channel(rgb[1]) + 0.0722 * channel(rgb[2]);
    return l > 0.45 ? "#1b1c1a" : "#faf9f5";
  }

  var chosen = null;

  function wearAccent() {
    if (!chosen) return;
    var tone = fit(chosen, theme() === "dark");
    root.style.setProperty("--accent", tone);
    root.style.setProperty("--accent-ink", readableOn(tone));
  }

  swatches.forEach(function (swatch) {
    swatch.addEventListener("click", function () {
      chosen = swatch.getAttribute("data-accent");
      swatches.forEach(function (other) {
        other.setAttribute("aria-pressed", other === swatch ? "true" : "false");
      });
      wearAccent();
    });
  });

  /* --- the downloads ---------------------------------------------------- */

  var releases = "https://github.com/yaffalhakim1/playlist/releases/latest";

  document.querySelectorAll("[data-download]").forEach(function (link) {
    var href = link.getAttribute("href");
    if (!href || href === "#") link.setAttribute("href", releases);
    link.setAttribute("rel", "noopener");
  });
})();
