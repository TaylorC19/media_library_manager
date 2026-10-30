/* Search previews are read-only; the dialog's POST actions save explicitly. */
(function () {
  "use strict";
  const dialog = document.getElementById("search-detail-dialog");
  const content = document.getElementById("search-detail-content");
  if (!dialog || !content) return;
  const close = document.getElementById("search-detail-close");
  let controller = null;
  let trigger = null;
  let currentURL = "";
  let pendingSave = null;
  let saveRevision = 0;
  const acknowledgedSaves = new Map();

  function identity(url) {
    const params = new URL(url, window.location.origin).searchParams;
    if (params.get("media_record_id")) return "local:" + params.get("media_record_id");
    const provider = params.get("provider") === "openlibrary" ? "open_library" : params.get("provider");
    return JSON.stringify([provider, params.get("external_id"), params.get("media_type"), provider === "tmdb" ? params.get("tmdb_kind") || params.get("media_type") : ""]);
  }

  function savedAction(result) {
    const slot = content.querySelector('[data-save-bucket="' + result.bucket + '"]');
    if (!slot) return null;
    const link = document.createElement("a");
    link.href = result.url;
    link.className = "button-link";
    link.textContent = dialog.dataset.openSaved + " · " + (result.bucket === "catalog" ? dialog.dataset.collectionLabel : dialog.dataset.wishlistLabel);
    slot.replaceChildren(link);
    return link;
  }

  function saveStatus(text, failed) {
    const status = content.querySelector("[data-save-status]");
    if (!status) return;
    status.className = failed ? "form-error" : "";
    status.setAttribute("role", failed ? "alert" : "status");
    status.textContent = text;
  }

  function updateBadges(key, bucket) {
    document.querySelectorAll("a[data-search-detail]").forEach(link => {
      if (identity(link.href) !== key) return;
      const body = link.querySelector(".search-hit__body");
      if (!body) return;
      let badges = body.querySelector(".saved-badges");
      if (!badges) {
        badges = document.createElement("span");
        badges.className = "saved-badges";
        body.querySelector(".search-hit__title").after(badges);
      }
      if (badges.querySelector('[data-saved-bucket="' + bucket + '"]')) return;
      const badge = document.createElement("span");
      badge.dataset.savedBucket = bucket;
      badge.className = "saved-badge saved-badge--" + (bucket === "catalog" ? "collection" : "wishlist");
      badge.textContent = bucket === "catalog" ? dialog.dataset.badgeCollection : dialog.dataset.badgeWishlist;
      badges.appendChild(badge);
    });
  }

  async function save(form) {
    if (pendingSave) return;
    const key = identity(currentURL);
    const body = new URLSearchParams(new FormData(form));
    const bucket = body.get("bucket");
    const buttons = Array.from(content.querySelectorAll('form[data-dialog-save] button[type="submit"]'));
    const submit = form.querySelector('button[type="submit"]');
    const label = submit.textContent;
    pendingSave = { key };
    buttons.forEach(button => { button.disabled = true; });
    submit.textContent = dialog.dataset.saving;
    form.setAttribute("aria-busy", "true");
    saveStatus(dialog.dataset.saving, false);
    try {
      // Do not abort a POST on dialog close: it may already have committed.
      // Its acknowledgement can still update the matching visible result card.
      const response = await fetch(form.action, { method: "POST", credentials: "same-origin", headers: { "X-Search-Dialog": "1", "Accept": "application/json", "Content-Type": "application/x-www-form-urlencoded;charset=UTF-8" }, body });
      if (!response.ok || response.redirected) throw new Error("save failed");
      const result = await response.json();
      if (!/^[a-f0-9]{24}$/.test(result.entryId) || result.bucket !== bucket || !["catalog", "wishlist"].includes(bucket)) throw new Error("invalid save response");
      const target = new URL(result.url, window.location.origin);
      if (target.origin !== window.location.origin || !/^\/(en|ja)\/library\/[a-f0-9]{24}$/.test(target.pathname)) throw new Error("invalid saved-item link");
      if (!acknowledgedSaves.has(key)) acknowledgedSaves.set(key, new Map());
      acknowledgedSaves.get(key).set(bucket, { revision: ++saveRevision, result });
      updateBadges(key, bucket);
      if (dialog.open && identity(currentURL) === key) {
        const restoreFocus = form.contains(document.activeElement);
        const link = savedAction(result);
        saveStatus(result.existing ? dialog.dataset.existing : dialog.dataset.saved, false);
        if (restoreFocus && link) link.focus();
      }
    } catch (error) {
      if (dialog.open && identity(currentURL) === key) saveStatus(dialog.dataset.saveError, true);
    } finally {
      pendingSave = null;
      submit.textContent = label;
      form.setAttribute("aria-busy", "false");
      buttons.forEach(button => { button.disabled = false; });
      content.querySelectorAll('form[data-dialog-save] button[type="submit"]').forEach(button => { button.disabled = false; });
    }
  }

  document.addEventListener("submit", function (event) {
    const form = event.target.closest("form[data-dialog-save]");
    if (!form || !dialog.contains(form)) return;
    event.preventDefault();
    if (dialog.dataset.canWrite === "false") return;
    save(form);
  });

  function message(text) {
    const heading = document.createElement("h2");
    heading.id = "search-detail-title";
    heading.textContent = text;
    content.replaceChildren(heading);
  }

  async function load(url) {
    if (controller) controller.abort();
    const request = new AbortController();
    controller = request;
    currentURL = url;
    const revisionAtStart = saveRevision;
    message(dialog.dataset.loading);
    const target = new URL(url, window.location.origin);
    target.searchParams.set("fragment", "1");
    try {
      const response = await fetch(target.pathname + target.search, { credentials: "same-origin", signal: request.signal });
      if (!response.ok || response.redirected) throw new Error("preview failed");
      const html = await response.text();
      if (request.signal.aborted || controller !== request || !dialog.open) return;
      content.innerHTML = html;
      // A preview that began before a save completed may contain older status.
      for (const ack of acknowledgedSaves.get(identity(url))?.values() || []) {
        if (ack.revision > revisionAtStart) savedAction(ack.result);
      }
      if (pendingSave) content.querySelectorAll('form[data-dialog-save] button[type="submit"]').forEach(button => { button.disabled = true; });
      content.querySelector("#search-detail-title").focus();
    } catch (error) {
      if (request.signal.aborted || controller !== request || !dialog.open) return;
      message(dialog.dataset.error);
      const retry = document.createElement("button");
      retry.type = "button";
      retry.textContent = dialog.dataset.retry;
      retry.addEventListener("click", () => load(currentURL));
      content.appendChild(retry);
    }
  }

  // Barcode-only providers without an import adapter can still be catalogued manually.
  function manualPreview(link) {
    if (controller) controller.abort();
    message(link.dataset.previewTitle);
    if (link.dataset.previewImage) {
      const image = document.createElement("img");
      image.src = link.dataset.previewImage;
      image.alt = "";
      image.className = "media-preview__cover";
      content.appendChild(image);
    }
    const description = document.createElement("p");
    description.textContent = link.dataset.previewCreator || "";
    content.appendChild(description);
    if (dialog.dataset.canWrite === "false") return;
    for (const bucket of ["catalog", "wishlist"]) {
      const action = document.createElement("a");
      const url = new URL(link.href, window.location.origin);
      url.searchParams.set("bucket", bucket);
      action.href = url.pathname + url.search;
      action.className = "button-link";
      action.textContent = bucket === "catalog" ? dialog.dataset.manualCatalog : dialog.dataset.manualWishlist;
      content.appendChild(action);
    }
  }

  document.addEventListener("click", function (event) {
    const link = event.target.closest("a[data-search-detail]");
    if (!link || event.defaultPrevented || event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;
    if (typeof dialog.showModal !== "function") return; // Ordinary detail-page navigation remains available.
    event.preventDefault();
    trigger = link;
    if (!dialog.open) dialog.showModal();
    if (link.hasAttribute("data-manual-preview")) manualPreview(link);
    else load(link.href);
  });
  close.addEventListener("click", () => dialog.close());
  dialog.addEventListener("click", function (event) {
    if (event.target !== dialog) return;
    const bounds = dialog.getBoundingClientRect();
    if (event.clientX < bounds.left || event.clientX > bounds.right || event.clientY < bounds.top || event.clientY > bounds.bottom) dialog.close();
  });
  dialog.addEventListener("close", function () {
    if (controller) controller.abort();
    content.replaceChildren();
    if (trigger && trigger.isConnected) trigger.focus();
  });

  const manual = document.getElementById("create-manually");
  const mediaType = document.getElementById("media_type");
  const query = document.getElementById("q");
  function updateManualLink() {
    if (!manual) return;
    const url = new URL(manual.href);
    url.searchParams.set("media_type", mediaType.value);
    url.searchParams.set("q", query.value);
    manual.href = url.pathname + url.search;
  }
  mediaType.addEventListener("change", updateManualLink);
  query.addEventListener("input", updateManualLink);
  updateManualLink();
})();
