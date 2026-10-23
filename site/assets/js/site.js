// VPS Pilot site: theme toggle, GitHub star count, copy buttons on code
// blocks and the mobile docs menu. The saved theme is applied before first
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

// GitHub star count in the header, from the GitHub API. Cached for an hour
// per tab so page views don't eat the API's rate limit. On any failure the
// button just shows no count.
(function () {
  var link = document.querySelector("[data-github-repo]");
  if (!link || !window.fetch) return;
  var stars = link.querySelector("[data-github-stars]");
  var key = "github-stars:" + link.getAttribute("data-github-repo");

  function show(count) {
    var text = count >= 1000 ? (count / 1000).toFixed(1).replace(/\.0$/, "") + "k" : String(count);
    stars.querySelector("[data-github-stars-count]").textContent = text;
    stars.hidden = false;
    link.setAttribute("aria-label", "VPS Pilot on GitHub, " + count + " stars");
  }

  try {
    var cached = JSON.parse(sessionStorage.getItem(key));
    if (cached && Date.now() - cached.at < 3600000) return show(cached.count);
  } catch (e) {}

  fetch("https://api.github.com/repos/" + link.getAttribute("data-github-repo"), {
    headers: { Accept: "application/vnd.github+json" },
  })
    .then(function (res) {
      return res.ok ? res.json() : Promise.reject(res.status);
    })
    .then(function (repo) {
      if (typeof repo.stargazers_count !== "number") return;
      show(repo.stargazers_count);
      try {
        sessionStorage.setItem(key, JSON.stringify({ count: repo.stargazers_count, at: Date.now() }));
      } catch (e) {}
    })
    .catch(function () {});
})();
