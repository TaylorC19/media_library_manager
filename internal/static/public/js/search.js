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
    message(dialog.dataset.loading);
    const target = new URL(url, window.location.origin);
    target.searchParams.set("fragment", "1");
    try {
      const response = await fetch(target.pathname + target.search, { credentials: "same-origin", signal: request.signal });
      if (!response.ok || response.redirected) throw new Error("preview failed");
      const html = await response.text();
      if (request.signal.aborted || controller !== request || !dialog.open) return;
      content.innerHTML = html;
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
