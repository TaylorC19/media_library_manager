/* Candidate keys are tested independently; only Save writes settings. */
(function () {
  "use strict";
  document.querySelectorAll("[data-test-provider]").forEach(function (button) {
    const provider = button.dataset.testProvider;
    const input = document.getElementById("provider-" + provider);
    const result = document.getElementById("provider-test-" + provider);
    const remove = button.form && button.form.querySelector('[name="remove_' + provider + '"]');
    let request = null;
    function reset() {
      if (request) request.abort();
      request = null;
      result.textContent = "";
      result.removeAttribute("aria-busy");
      button.disabled = false;
    }
    if (input) input.addEventListener("input", reset);
    if (remove) remove.addEventListener("change", reset);
    button.addEventListener("click", async function () {
      reset();
      const current = new AbortController();
      request = current;
      button.disabled = true;
      result.setAttribute("aria-busy", "true");
      result.textContent = result.dataset.testing;
      const body = new URLSearchParams({ provider });
      if (input) {
        body.set("key", input.value);
        body.set("remove", remove && remove.checked ? "on" : "");
      }
      try {
        const response = await fetch(button.dataset.testUrl, {
          method: "POST", credentials: "same-origin", body, signal: current.signal,
        });
        if (!response.ok || response.redirected) throw new Error("Test unavailable");
        const html = await response.text();
        if (request === current && !current.signal.aborted) result.innerHTML = html;
      } catch {
        if (request === current && !current.signal.aborted) result.textContent = result.dataset.error;
      } finally {
        if (request === current) {
          button.disabled = false;
          result.removeAttribute("aria-busy");
          request = null;
        }
      }
    });
    window.addEventListener("pagehide", reset);
  });
})();
