# Unsaved-edit protection QA — 2026-11-02

- Browser: typed disposable notes into Create manually, clicked Settings, and observed the discard confirmation. Cancel kept the page and notes intact. Clearing the notes back to their original value then allowed Settings navigation without a warning. No item or API key was saved.
- JavaScript regression coverage: unchanged/reverted values, reload protection, canceled links/logout, native versus intercepted submissions, saved snapshots with newer edits, checkbox/dynamic forms, HTMX replacement, and Search dialog close/Escape.
- All Go tests, including disposable MongoDB integration fixtures, passed. Browser-level tab-close/reload and actual saves were not submitted against user records; event and save behavior are covered by the JavaScript suite.
- Local drafts are outside this change. Browser beforeunload warnings depend on browser support and a prior user interaction.
