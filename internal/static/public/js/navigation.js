/* HTMX changes the URL without rerendering the header. Keep language links current. */
(function () {
  "use strict";
  function updateLanguageLinks() {
    document.querySelectorAll("[data-locale-target]").forEach(function (link) {
      const locale = link.dataset.localeTarget;
      if (locale !== "en" && locale !== "ja") return;
      const url = new URL(window.location.href);
      const parts = url.pathname.split("/");
      if (parts[1] === "en" || parts[1] === "ja") parts[1] = locale;
      else parts.splice(1, 0, locale);
      url.pathname = parts.join("/");
      link.href = url.pathname + url.search + url.hash;
    });
  }
  document.addEventListener("htmx:afterSettle", updateLanguageLinks);
  document.addEventListener("htmx:historyRestore", updateLanguageLinks);
  window.addEventListener("popstate", updateLanguageLinks);
  window.addEventListener("pageshow", updateLanguageLinks);
  updateLanguageLinks();
})();
