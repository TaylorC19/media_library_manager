/* global document, fetch, FormData, window */

(function () {
  function handleSubmit(event) {
    var form = event.target;
    if (!form || form.tagName !== "FORM") return;
    event.preventDefault();

    var submit = form.querySelector("[data-js-auth-submit]");
    var previousLabel = submit ? submit.textContent : "";
    if (submit) {
      submit.disabled = true;
      submit.textContent = "...";
    }

    fetch(form.action, {
      method: form.method || "POST",
      body: new FormData(form),
      credentials: "same-origin",
      headers: {
        "X-Requested-With": "fetch",
      },
      redirect: "follow",
    })
      .then(function (response) {
        if (response.redirected) {
          window.location.assign(response.url);
          return null;
        }
        return response.text();
      })
      .then(function (html) {
        if (typeof html === "string" && html.length > 0) {
          document.open();
          document.write(html);
          document.close();
        }
      })
      .catch(function () {
        form.submit();
      })
      .finally(function () {
        if (submit) {
          submit.disabled = false;
          submit.textContent = previousLabel;
        }
      });
  }

  document.querySelectorAll("form[data-js-auth-form]").forEach(function (form) {
    form.addEventListener("submit", handleSubmit);
  });
})();
