/* global document, fetch, FormData, window */

(function () {
  function clearErrors(form) {
    var box = form.previousElementSibling;
    if (box && box.getAttribute("data-js-auth-errors") === "1") {
      box.remove();
    }
  }

  function showErrors(form, errors) {
    clearErrors(form);
    if (!errors || !errors.length) return;
    var box = document.createElement("section");
    box.setAttribute("data-js-auth-errors", "1");
    var ul = document.createElement("ul");
    errors.forEach(function (msg) {
      var li = document.createElement("li");
      li.textContent = msg;
      ul.appendChild(li);
    });
    box.appendChild(ul);
    form.parentNode.insertBefore(box, form);
  }

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

    var body = new URLSearchParams(new FormData(form));

    fetch(form.action, {
      method: form.method || "POST",
      body: body.toString(),
      credentials: "same-origin",
      headers: {
        "X-Requested-With": "fetch",
        Accept: "application/json",
        "Content-Type": "application/x-www-form-urlencoded;charset=UTF-8",
      },
      redirect: "manual",
    })
      .then(function (response) {
        var contentType = response.headers.get("Content-Type") || "";
        if (contentType.indexOf("application/json") !== -1) {
          return response.json().then(function (data) {
            if (data && data.ok && data.redirect) {
              window.location.assign(data.redirect);
              return null;
            }
            showErrors(form, (data && data.errors) || []);
            return null;
          });
        }
        if (response.status >= 300 && response.status < 400) {
          var loc = response.headers.get("Location");
          if (loc) {
            window.location.assign(loc);
            return null;
          }
        }
        if (response.redirected) {
          window.location.assign(response.url);
          return null;
        }
        return response.text();
      })
      .then(function (html) {
        clearErrors(form);
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
