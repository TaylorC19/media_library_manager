# QA artifact handling

Keep sanitized Markdown QA summaries in Git. Preserve reproducible UI observations, viewport dimensions, test scope, results, and untested flows. Use representative or synthetic examples instead of identifying a person's saved titles, notes, account names, record IDs, or account inventory.

Screenshots and raw reports remain local-only and are ignored by Git. Store internal Markdown reports in the date directory's ignored `private/` folder. Remove links to excluded screenshots from publishable summaries so a clean checkout has no broken evidence links. These local files may contain private data and must not be force-added or uploaded publicly.

The regression tests under `internal/` and `scripts/` remain tracked and are separate from private browser evidence. QA/security documentation is excluded from Docker and Vercel deployment uploads.

Current ignore rules prevent ordinary additions of private evidence. Follow the [history hygiene guide](../security/history-cleanup.md) when previously committed material requires cleanup; keep incident-specific status in private records.
