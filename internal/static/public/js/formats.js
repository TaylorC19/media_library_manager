/* Keep format choices in sync with each form's selected media type. */
(function () {
  "use strict";
  function update(form) {
    const select = form.querySelector("select[data-media-format]");
    const type = form.querySelector('[name="media_type"]');
    const template = form.querySelector("template[data-format-options]");
    if (!select || !type || !template) return;
    const selected = select.value;
    const blank = select.options[0].cloneNode(true);
    const choices = Array.from(template.content.querySelectorAll("option")).filter(option => {
      return !type.value || option.dataset.mediaTypes.split(" ").includes(type.value) || (type.value === select.dataset.legacyMediaType && option.value === select.dataset.legacyFormat);
    }).map(option => option.cloneNode(true));
    select.replaceChildren(blank, ...choices);
    select.value = choices.some(option => option.value === selected) ? selected : "";
  }
  document.addEventListener("change", function (event) {
    if (event.target.name === "media_type") {
      const form = event.target.closest("form");
      if (form) update(form);
    }
  });
  function initialize() { document.querySelectorAll("form").forEach(update); }
  document.addEventListener("htmx:historyRestore", initialize);
  window.addEventListener("pageshow", initialize);
  initialize();
})();
