// VPS Pilot site: theme toggle. The saved theme is applied before first
// paint by the inline script in partials/head.html.
(function () {
  var root = document.documentElement;

  function saved() {
    try {
      return localStorage.getItem("theme");
    } catch (e) {
      return null;
    }
  }

  var toggle = document.querySelector("[data-theme-toggle]");
  if (toggle) {
    toggle.addEventListener("click", function () {
      var dark = root.classList.toggle("dark");
      try {
        localStorage.setItem("theme", dark ? "dark" : "light");
      } catch (e) {}
    });
  }

  // Follow the system setting until the visitor picks a theme.
  var media = window.matchMedia("(prefers-color-scheme: dark)");
  if (media.addEventListener) {
    media.addEventListener("change", function (e) {
      if (!saved()) root.classList.toggle("dark", e.matches);
    });
  }
})();
