/* Playlist — the two pieces of behaviour on the landing page.
 *
 * Both degrade: the page is complete and readable with this file missing. The
 * theme falls back to the desktop's own setting, and the reveals simply show.
 */

(function () {
  "use strict";

  /* --- the theme ------------------------------------------------------ */

  var root = document.documentElement;
  var toggle = document.getElementById("theme-toggle");
  var label = toggle && toggle.querySelector(".theme-label");

  function current() {
    return root.dataset.theme === "dark" ? "dark" : "light";
  }

  function paint(theme) {
    root.dataset.theme = theme;
    // The screenshots follow the page, not the desktop. A light shot on a dark
    // page reads as a bug, and <source media> only knows about the system
    // setting, not the choice this button makes.
    var shots = document.querySelectorAll("img[data-light][data-dark]");
    for (var i = 0; i < shots.length; i++) {
      var want = shots[i].getAttribute(theme === "dark" ? "data-dark" : "data-light");
      if (shots[i].getAttribute("src") !== want) shots[i].setAttribute("src", want);
    }
    if (!toggle) return;
    // The button says what pressing it will do, not what the page currently is:
    // a control that names the current state reads as a status, not an action.
    var next = theme === "dark" ? "light" : "dark";
    if (label) label.textContent = next === "dark" ? "Dark" : "Light";
    toggle.setAttribute("aria-label", "Switch to the " + next + " appearance");
    toggle.setAttribute("aria-pressed", theme === "dark" ? "true" : "false");
  }

  function remember(theme) {
    try { localStorage.setItem("theme", theme); } catch (e) {}
  }

  paint(current());

  if (toggle) {
    toggle.addEventListener("click", function () {
      var next = current() === "dark" ? "light" : "dark";
      paint(next);
      remember(next);
    });
  }

  // Until the reader chooses, follow the desktop as it changes. Once they have
  // chosen, their choice wins.
  var media = window.matchMedia("(prefers-color-scheme: dark)");
  var listener = function (event) {
    var chosen = null;
    try { chosen = localStorage.getItem("theme"); } catch (e) {}
    if (chosen) return;
    paint(event.matches ? "dark" : "light");
  };
  if (media.addEventListener) media.addEventListener("change", listener);
  else if (media.addListener) media.addListener(listener);

  /* --- the header's hairline ------------------------------------------ */

  // The rule under the header appears once the page has moved, so at the top
  // the header is only the page itself.
  var head = document.querySelector(".site-head");
  if (head) {
    var onScroll = function () {
      head.dataset.stuck = window.scrollY > 8 ? "true" : "false";
    };
    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });
  }

  /* --- the reveal ------------------------------------------------------ */

  // One IntersectionObserver, disconnected as each element is revealed: there
  // is nothing left to watch once it has come in.
  var items = document.querySelectorAll(".reveal");
  var reduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

  if (!items.length) return;

  if (reduced || !("IntersectionObserver" in window)) {
    for (var i = 0; i < items.length; i++) items[i].dataset.shown = "true";
    return;
  }

  var observer = new IntersectionObserver(function (entries) {
    entries.forEach(function (entry) {
      if (!entry.isIntersecting) return;
      entry.target.dataset.shown = "true";
      observer.unobserve(entry.target);
    });
  }, { rootMargin: "0px 0px -12% 0px", threshold: 0.08 });

  for (var j = 0; j < items.length; j++) observer.observe(items[j]);

  // A jump — an anchor link, a restored scroll position, a reload part way
  // down — can land past an element without ever intersecting it, which would
  // leave it invisible for good. Anything already above the fold is shown at
  // once, and the observer only handles what is still below.
  var showAbove = function () {
    for (var k = 0; k < items.length; k++) {
      if (items[k].dataset.shown === "true") continue;
      var box = items[k].getBoundingClientRect();
      if (box.top < window.innerHeight * 0.92) {
        items[k].dataset.shown = "true";
        observer.unobserve(items[k]);
      }
    }
  };
  showAbove();
  window.addEventListener("scroll", showAbove, { passive: true });
  window.addEventListener("resize", showAbove, { passive: true });
})();
