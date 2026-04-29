/**
 * Page-scoped barcode scan: camera + BarcodeDetector or ZXing, POST /barcode/lookup, render candidates.
 */
(function () {
  "use strict";

  const root = document.getElementById("scan-root");
  if (!root) return;

  const locale = (root.getAttribute("data-locale") || "en").trim();

  function str(id) {
    const el = document.getElementById(id);
    return el ? el.textContent.trim() : id;
  }

  function providerLabel(p) {
    if (!p) return "";
    const key = p.replace(/[^a-z0-9_]/gi, "_");
    const byId = document.getElementById("str-provider-" + p) || document.getElementById("str-provider-" + key);
    return byId ? byId.textContent.trim() : p;
  }

  function normalizeBarcode(value) {
    if (!value) return "";
    const s = String(value).trim().toUpperCase();
    let out = "";
    for (let i = 0; i < s.length; i++) {
      const c = s.charAt(i);
      if (c >= "0" && c <= "9") out += c;
      else if (c === "X") out += "X";
    }
    return out;
  }

  const video = document.getElementById("scan-video");
  const scanStatus = document.getElementById("scan-status");
  const resultsEl = document.getElementById("scan-results");
  const input = document.getElementById("scan-barcode-input");
  const preferred = document.getElementById("scan-preferred-type");
  const btnLookup = document.getElementById("scan-lookup-btn");
  const btnStart = document.getElementById("scan-start-cam");
  const btnStop = document.getElementById("scan-stop-cam");
  const btnAgain = document.getElementById("scan-again-btn");
  const cameraHint = document.getElementById("scan-camera-hint");
  const fallbackLink = document.getElementById("scan-fallback-search");

  let mediaStream = null;
  let zxingReader = null;
  let detectorLoop = 0;
  let decodeMode = "none"; // none | zxing | detector
  let lastResolved = "";
  let lastLookupNorm = "";
  let lookupInFlight = false;
  let blockedAfterRead = false;

  function setStatus(msg) {
    if (scanStatus) scanStatus.textContent = msg || "";
  }

  function stopCamera() {
    if (zxingReader) {
      try {
        zxingReader.stopContinuousDecode();
        zxingReader.reset();
      } catch {
        void 0;
      }
      zxingReader = null;
    }
    if (detectorLoop) {
      cancelAnimationFrame(detectorLoop);
      detectorLoop = 0;
    }
    if (mediaStream) {
      mediaStream.getTracks().forEach((t) => t.stop());
      mediaStream = null;
    }
    if (video) {
      try {
        video.srcObject = null;
      } catch {
        void 0;
      }
    }
    decodeMode = "none";
    if (btnStop) btnStop.disabled = true;
  }

  function onBarcodeLine(raw) {
    const n = normalizeBarcode(raw);
    if (n.length < 4) return;
    if (blockedAfterRead && n === lastResolved) return;
    lastResolved = n;
    if (input) input.value = raw;
    stopCamera();
    blockedAfterRead = true;
    if (btnAgain) btnAgain.disabled = false;
    runLookup(raw, { force: true });
  }

  function runDetectorLoop() {
    if (decodeMode !== "detector" || !video) return;
    if (!("BarcodeDetector" in window)) return;

    const detector = new window.BarcodeDetector({
      formats: ["ean_13", "ean_8", "upc_a", "upc_e"],
    });

    const tick = async () => {
      if (decodeMode !== "detector" || !video || video.readyState < 2) {
        detectorLoop = requestAnimationFrame(tick);
        return;
      }
      if (blockedAfterRead) {
        return;
      }
      try {
        const bmp = await createImageBitmap(video);
        const codes = await detector.detect(bmp);
        if (codes && codes[0] && codes[0].rawValue) {
          onBarcodeLine(codes[0].rawValue);
          return;
        }
      } catch {
        void 0;
      }
      detectorLoop = requestAnimationFrame(tick);
    };
    detectorLoop = requestAnimationFrame(tick);
  }

  function startZxing() {
    const ZB = window.ZXingBrowser;
    if (!ZB || !ZB.BrowserMultiFormatReader) {
      setHintText(str("str-no-zxing"));
      return;
    }
    const BF = ZB.BarcodeFormat;
    const hints = new Map();
    const fmts = [BF.EAN_13, BF.EAN_8, BF.UPC_A, BF.UPC_E].filter(Boolean);
    hints.set(2, fmts);
    zxingReader = new ZB.BrowserMultiFormatReader(hints, 300);
    zxingReader.decodeFromVideoElementContinuously(video, (result, err) => {
      if (blockedAfterRead) return;
      if (err) return;
      if (result && result.getText) {
        onBarcodeLine(result.getText());
      }
    });
  }

  function setHintText(t) {
    if (!cameraHint) return;
    if (t) {
      cameraHint.hidden = false;
      cameraHint.textContent = t;
    } else {
      cameraHint.hidden = true;
      cameraHint.textContent = "";
    }
  }

  async function startCamera() {
    setHintText("");
    if (!video || !navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
      setHintText(str("str-camera-unsupported"));
      return;
    }
    if (!("BarcodeDetector" in window) && !window.ZXingBrowser) {
      setHintText(str("str-no-decoder"));
      return;
    }
    stopCamera();
    blockedAfterRead = false;
    lastResolved = "";
    if (btnAgain) btnAgain.disabled = true;
    try {
      mediaStream = await navigator.mediaDevices.getUserMedia({
        video: { facingMode: { ideal: "environment" } },
        audio: false,
      });
      video.srcObject = mediaStream;
      await video.play();
    } catch (e) {
      if (e && e.name === "NotAllowedError") {
        setHintText(str("str-camera-denied"));
      } else {
        setHintText(String(e && e.message ? e.message : e));
      }
      return;
    }
    if (btnStop) btnStop.disabled = false;
    if ("BarcodeDetector" in window) {
      decodeMode = "detector";
      runDetectorLoop();
    } else {
      decodeMode = "zxing";
      startZxing();
    }
  }


  if (btnStart) {
    btnStart.addEventListener("click", function () {
      startCamera();
    });
  }
  if (btnStop) {
    btnStop.addEventListener("click", function () {
      stopCamera();
    });
  }
  if (btnAgain) {
    btnAgain.addEventListener("click", function () {
      blockedAfterRead = false;
      lastResolved = "";
      if (btnAgain) btnAgain.disabled = true;
      if (input) input.value = "";
      clearResults();
      setStatus("");
    });
  }

  function clearResults() {
    if (resultsEl) resultsEl.innerHTML = "";
  }

  function el(tag, cls, text) {
    const n = document.createElement(tag);
    if (cls) n.className = cls;
    if (text) n.textContent = text;
    return n;
  }

  function runLookup(raw, opts) {
    const force = opts && opts.force;
    const norm = normalizeBarcode(raw);
    if (norm.length < 4) {
      setStatus(str("str-lookup-failed"));
      return;
    }
    if (lookupInFlight) return;
    if (!force && norm === lastLookupNorm) {
      return;
    }
    lookupInFlight = true;
    setStatus("…");
    const pref = preferred && preferred.value ? preferred.value : "";
    fetch("/barcode/lookup", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "same-origin",
      body: JSON.stringify({ barcode: raw, preferredMediaType: pref }),
    })
      .then(function (res) {
        if (!res.ok) {
          if (res.status === 400) {
            setStatus(str("str-lookup-failed"));
            return null;
          }
          throw new Error(String(res.status));
        }
        return res.json();
      })
      .then(function (data) {
        if (!data) return;
        lastLookupNorm = norm;
        renderResponse(data, norm, raw);
        updateFallbackLink(data);
      })
      .catch(function () {
        setStatus(str("str-lookup-failed"));
      })
      .finally(function () {
        lookupInFlight = false;
      });
  }

  function updateFallbackLink(data) {
    if (!fallbackLink || !data) return;
    const u = new URL("/" + locale + "/search", window.location.origin);
    u.search = "";
    if (data.fallback) {
      const f = data.fallback;
      if (f.manualQuery) u.searchParams.set("q", f.manualQuery);
      if (f.mediaType) u.searchParams.set("media_type", f.mediaType);
    }
    fallbackLink.href = u.pathname + (u.search ? u.search : "");
  }

  function hasBucket(links, bucket) {
    if (!links) return false;
    return links.some(function (L) {
      return L.bucket === bucket;
    });
  }

  function renderResponse(data, norm, raw) {
    clearResults();
    setStatus("");

    const h2 = el("h3", "scan-results-heading", str("str-results-heading"));
    resultsEl.appendChild(h2);

    const bar = el("p", "muted scan-scanned", "");
    const label = str("str-barcode-label");
    const shown = (data && data.barcode) || norm || (raw && String(raw)) || "";
    bar.appendChild(document.createTextNode(label + " "));
    const codeStrong = el("strong", null, shown);
    bar.appendChild(codeStrong);
    resultsEl.appendChild(bar);

    if (data.candidates && data.candidates.length) {
      data.candidates.forEach(function (c) {
        resultsEl.appendChild(renderCandidate(c, data.barcode || norm || String(raw)));
      });
    } else {
      resultsEl.appendChild(el("p", "empty-state", str("str-noresults")));
    }

    if (data.failures && data.failures.length) {
      const fh = el("h4", "scan-failures-heading", str("str-failures-heading"));
      const ul = el("ul", "scan-failure-list", "");
      data.failures.forEach(function (f) {
        const li = el("li", "muted", (f.provider || "?") + ": " + (f.code || ""));
        ul.appendChild(li);
      });
      resultsEl.appendChild(fh);
      resultsEl.appendChild(ul);
    }
  }

  function renderCandidate(c, scanned) {
    const card = el("article", "search-hit card scan-candidate", "");
    const media = el("div", "search-hit__media", "");
    if (c.imageUrl) {
      const img = document.createElement("img");
      img.className = "search-hit__img";
      img.src = c.imageUrl;
      img.alt = "";
      img.width = 96;
      img.height = 144;
      img.loading = "lazy";
      media.appendChild(img);
    } else {
      media.appendChild(el("div", "search-hit__placeholder", ""));
    }
    const body = el("div", "search-hit__body", "");
    const title = el("h3", "search-hit__title", c.title || "");
    body.appendChild(title);
    if (c.year) {
      body.appendChild(el("p", "search-hit__year muted", String(c.year)));
    }
    if (c.creatorLine) {
      body.appendChild(el("p", "search-hit__subtitle muted", c.creatorLine));
    }
    const src = el("p", "search-hit__provider", "");
    if (c.source === "local") {
      src.appendChild(el("span", "provider-pill", str("str-source-local")));
    } else {
      const pill = el("span", "provider-pill", providerLabel(c.provider) + " — " + str("str-source-provider"));
      src.appendChild(pill);
    }
    body.appendChild(src);

    const actions = el("div", "search-hit__actions", "");

    if (c.source === "provider" && c.provider && c.providerId) {
      actions.appendChild(buildImportForm("catalog", c));
      actions.appendChild(buildImportForm("wishlist", c));
    } else if (c.source === "local" && c.mediaRecordId) {
      (c.linkedLibraryEntries || []).forEach(function (L) {
        const a = el("a", "button-link", str("str-view-entry-short") + " (" + L.bucket + ")");
        a.href = "/" + locale + "/library/" + L.entryId;
        actions.appendChild(a);
      });
      if (!hasBucket(c.linkedLibraryEntries, "catalog")) {
        actions.appendChild(buildAttachForm("catalog", c.mediaRecordId, scanned));
      }
      if (!hasBucket(c.linkedLibraryEntries, "wishlist")) {
        actions.appendChild(buildAttachForm("wishlist", c.mediaRecordId, scanned));
      }
    }

    body.appendChild(actions);
    card.appendChild(media);
    card.appendChild(body);
    return card;
  }

  function buildImportForm(bucket, c) {
    const form = el("form", "inline-form", "");
    form.method = "post";
    form.action = "/media/import";
    form.appendChild(hidden("locale", locale));
    form.appendChild(hidden("bucket", bucket));
    form.appendChild(hidden("media_type", c.mediaType || ""));
    form.appendChild(hidden("provider", c.provider || ""));
    form.appendChild(hidden("external_id", c.providerId || ""));
    form.appendChild(hidden("tmdb_kind", ""));
    const b = el("button", null, bucket === "catalog" ? str("str-add-catalog") : str("str-add-wishlist"));
    b.type = "submit";
    form.appendChild(b);
    return form;
  }

  function buildAttachForm(bucket, mediaRecordId, scanned) {
    const form = el("form", "inline-form", "");
    form.method = "post";
    form.action = "/library/attach";
    form.appendChild(hidden("locale", locale));
    form.appendChild(hidden("media_record_id", mediaRecordId));
    form.appendChild(hidden("bucket", bucket));
    form.appendChild(hidden("barcode", scanned));
    const b = el("button", null, bucket === "catalog" ? str("str-add-catalog") : str("str-add-wishlist"));
    b.type = "submit";
    form.appendChild(b);
    return form;
  }

  function hidden(name, value) {
    const i = document.createElement("input");
    i.type = "hidden";
    i.name = name;
    i.value = value;
    return i;
  }

  if (btnLookup) {
    btnLookup.addEventListener("click", function () {
      const v = input && input.value;
      runLookup(v, { force: true });
    });
  }
  if (input) {
    input.addEventListener("keydown", function (ev) {
      if (ev.key === "Enter") {
        ev.preventDefault();
        runLookup(input.value, { force: true });
      }
    });
  }

  window.addEventListener("pagehide", stopCamera);
  window.addEventListener("beforeunload", stopCamera);
  document.addEventListener("visibilitychange", function () {
    if (document.visibilityState === "hidden") stopCamera();
  });
})();
