# QA artifact handling

Keep sanitized Markdown QA summaries in Git. Preserve reproducible UI observations, viewport dimensions, test scope, results, and untested flows. Use representative or synthetic examples instead of identifying a person's saved titles, notes, account names, record IDs, or account inventory.

Screenshots and raw reports remain local-only and are ignored by Git. Store internal Markdown reports in the date directory's ignored `private/` folder. Remove links to excluded screenshots from publishable summaries so a clean checkout has no broken evidence links. These local files may contain private data and must not be force-added or uploaded publicly.

The regression tests under `internal/` and `scripts/` remain tracked and are separate from private browser evidence. QA/security documentation is excluded from Docker and Vercel deployment uploads.

QA/audit folders were removed from local and affected remote branch histories on 2026-11-09. Current ignore rules prevent ordinary future additions. Older clones, forks, and host-retained refs or caches can still contain prior copies; see [the cleanup record](../security/history-cleanup.md).
