// Remove snapshots created by older releases before htmx can restore them.
try { localStorage.removeItem('htmx-history-cache'); } catch (_) {}
window.addEventListener('pageshow', function (event) {
  if (event.persisted) window.location.reload();
});
