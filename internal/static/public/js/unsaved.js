/* Values stay in the form: this guard never persists drafts or credentials. */
(function () {
  "use strict";
  const selector = "form[data-unsaved]";
  const baselines = new WeakMap();
  let leaving = false;
  function allowNavigation() {
    leaving = true;
    // Downloads or canceled navigation can leave this document in place.
    setTimeout(() => { leaving = false; }, 1000);
  }
  function snapshot(form) {
    return JSON.stringify(Array.from(form.elements).filter(field => field.name && !["submit", "button", "reset", "hidden"].includes(field.type)).map(field => {
      if (field.type === "checkbox" || field.type === "radio") return [field.name, field.checked];
      if (field.type === "file") return [field.name, Array.from(field.files || []).map(file => [file.name, file.size, file.lastModified])];
      if (field.multiple) return [field.name, Array.from(field.selectedOptions).map(option => option.value)];
      return [field.name, field.value];
    }));
  }
  function initialize() {
    document.querySelectorAll(selector).forEach(form => {
      if (!baselines.has(form)) baselines.set(form, form.dataset.unsavedInvalid === "true" ? null : snapshot(form));
    });
  }
  function dirty(form) { return baselines.has(form) && baselines.get(form) !== snapshot(form); }
  function changed(root, except) {
    initialize();
    return Array.from(root.querySelectorAll(selector)).some(form => form !== except && dirty(form));
  }
  function confirmDiscard(root = document, except) {
    return !changed(root, except) || window.confirm(document.body.dataset.unsavedMessage);
  }
  window.UnsavedChanges = {
    snapshot,
    saved(form, values) { baselines.set(form, values === undefined ? snapshot(form) : values); },
    confirmDiscard,
    initialize
  };
  document.addEventListener("click", function (event) {
    const link = event.target.closest("a[href]");
    if (!link || event.defaultPrevented || event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey || link.download || (link.target && link.target !== "_self")) return;
    const url = new URL(link.href, window.location.href);
    if (!["http:", "https:"].includes(url.protocol)) return;
    if (url.pathname === window.location.pathname && url.search === window.location.search && url.hash) return;
    if (!confirmDiscard()) { event.preventDefault(); event.stopImmediatePropagation(); return; }
    // Dialog previews and HTMX links prevent their native navigation later.
    queueMicrotask(() => { if (!event.defaultPrevented) allowNavigation(); });
  }, true);
  document.addEventListener("submit", function (event) {
    const form = event.target;
    if (event.defaultPrevented) return;
    const ownSave = form.matches(selector) && form.method.toLowerCase() === "post";
    if (!confirmDiscard(document, ownSave ? form : null)) {
      event.preventDefault(); event.stopImmediatePropagation(); return;
    }
    queueMicrotask(() => { if (!event.defaultPrevented && (!form.target || form.target === "_self")) allowNavigation(); });
  }, true);
  // A background HTMX refresh must not replace edited content silently.
  document.addEventListener("htmx:beforeSwap", function (event) {
    const target = event.detail.target;
    if (target && !confirmDiscard(target)) event.preventDefault();
  });
  window.addEventListener("beforeunload", function (event) {
    if (!leaving && changed(document)) { event.preventDefault(); event.returnValue = ""; }
  });
  window.addEventListener("pageshow", function () { leaving = false; initialize(); });
  initialize();
  new MutationObserver(initialize).observe(document.body, { childList: true, subtree: true });
})();
