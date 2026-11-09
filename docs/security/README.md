# Security review artifacts

Commit only reviewed, sanitized Markdown summaries in this directory. Keep findings, severity, remediation status, validation scope, and limitations useful for engineering review. Remove actual account identities, Library data, record IDs, credentials, local home paths, and detailed exploit requests/code from publishable documents.

Raw evidence belongs in an ignored `private/` directory inside the review date directory. It may include original internal reports, scanner output, test logs, and detailed reproductions. Ignored local files are not a durable backup or an access-control boundary: protect them on the host and use approved private storage when sharing is necessary. Do not force-add them to Git or attach them to public issues/PRs.

The Git ignore rules allow only Markdown summaries under `docs/qa/` and `docs/security/` and always exclude `private/` directories. Screenshot and other non-Markdown evidence files are excluded. Docker and Vercel upload exclusions also omit these documentation trees so private local evidence is not packaged for deployment.

Regression tests under `internal/` and `scripts/` remain tracked. Keep their fixtures synthetic and their credentials disposable. Add tests asserting secure rejection or preservation behavior when fixing audit findings. Audit reproductions that intentionally demonstrate a vulnerability stay in private evidence until reviewed for disclosure.

QA/audit folders were removed from local and affected remote branch histories on 2026-11-09. Old clones, forks, and host-retained pull-request refs or caches can still retain earlier copies and require separate cleanup. See [the cleanup record](history-cleanup.md). Credential exposure, if discovered, requires rotation regardless of history removal.
