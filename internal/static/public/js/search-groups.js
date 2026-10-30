/* Each media type loads independently; only connected, current slots may update. */
(function () {
  "use strict";
  const active = new Map();

  function summarize(root) {
    const slots = Array.from(root.querySelectorAll("[data-search-group-url]"));
    const pending = slots.some(slot => ["pending", "loading"].includes(slot.dataset.groupState));
    const failed = slots.some(slot => slot.dataset.groupState === "error");
    root.querySelector("[data-group-summary]").textContent = pending ? root.dataset.loading : failed ? root.dataset.partial : root.dataset.complete;
    root.querySelector("[data-groups-empty]").hidden = pending || failed || !!root.querySelector('[data-has-results="true"]');
  }

  async function load(slot) {
    if (active.has(slot)) return;
    const root = slot.closest("[data-search-groups]");
    const placeholder = slot.querySelector("[data-group-placeholder]");
    const message = slot.querySelector("[data-group-message]");
    const content = slot.querySelector("[data-group-content]");
    const request = new AbortController();
    active.set(slot, request);
    slot.dataset.groupState = "loading";
    slot.setAttribute("aria-busy", "true");
    message.textContent = root.dataset.loading;
    const previousRetry = slot.querySelector("[data-search-group-retry]");
    if (previousRetry) previousRetry.remove();
    summarize(root);
    const current = () => !request.signal.aborted && slot.isConnected && active.get(slot) === request;
    try {
      const response = await fetch(slot.dataset.searchGroupUrl, { credentials: "same-origin", signal: request.signal });
      if (!response.ok || response.redirected) throw new Error("group failed");
      const html = await response.text();
      if (!current()) return;
      content.innerHTML = html;
      if (!content.querySelector("[data-group-result]")) throw new Error("invalid group response");
      placeholder.hidden = true;
      slot.dataset.groupState = "done";
    } catch (error) {
      if (!current()) return;
      content.replaceChildren();
      message.textContent = slot.dataset.error;
      const retry = document.createElement("button");
      retry.type = "button";
      retry.setAttribute("data-search-group-retry", "");
      retry.textContent = slot.dataset.retry;
      placeholder.appendChild(retry);
      slot.dataset.groupState = "error";
    } finally {
      if (active.get(slot) === request) {
        active.delete(slot);
        slot.setAttribute("aria-busy", "false");
        if (slot.isConnected) summarize(root);
      }
    }
  }

  function initialize() {
    for (const [slot, request] of active) {
      if (!slot.isConnected) { request.abort(); active.delete(slot); }
    }
    document.querySelectorAll('[data-search-group-url][data-group-state="pending"]').forEach(load);
  }
  document.addEventListener("click", function (event) {
    const retry = event.target.closest("[data-search-group-retry]");
    if (retry) load(retry.closest("[data-search-group-url]"));
  });
  document.addEventListener("htmx:afterSwap", initialize);
  window.addEventListener("pagehide", function () {
    for (const [slot, request] of active) {
      request.abort();
      slot.dataset.groupState = "pending";
      slot.setAttribute("aria-busy", "false");
    }
    active.clear();
  });
  window.addEventListener("pageshow", initialize);
  initialize();
})();
