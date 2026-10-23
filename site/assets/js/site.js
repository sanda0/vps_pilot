// VPS Pilot site: theme toggle, copy buttons on code blocks and the mobile
// docs menu. The saved theme is applied before first paint by the inline
// script in partials/head.html.
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

// Copy buttons on code blocks.
(function () {
  if (!navigator.clipboard) return;
  document.querySelectorAll(".highlight").forEach(function (block) {
    var code = block.querySelector("code");
    if (!code) return;
    var button = document.createElement("button");
    button.type = "button";
    button.className = "copy-button";
    button.textContent = "Copy";
    button.addEventListener("click", function () {
      navigator.clipboard.writeText(code.innerText.replace(/\n$/, "")).then(
        function () {
          button.textContent = "Copied";
          setTimeout(function () {
            button.textContent = "Copy";
          }, 1500);
        },
        function () {
          button.textContent = "Press Ctrl+C";
        }
      );
    });
    block.appendChild(button);
  });
})();

// Docs sidebar toggle on small screens.
(function () {
  var button = document.querySelector("[data-docs-menu]");
  if (!button) return;
  var sidebar = document.getElementById(button.getAttribute("aria-controls"));
  button.addEventListener("click", function () {
    var open = sidebar.classList.toggle("open");
    button.setAttribute("aria-expanded", open ? "true" : "false");
  });
})();
