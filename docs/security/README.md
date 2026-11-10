# Security review artifacts

Commit only review-process documentation and reviewed summaries of completed remediations in this directory. Keep unresolved findings, severity registers, audit reports, and exploit details in ignored private storage. Removing credentials from a report does not make an unresolved risk register suitable for publication. Also remove actual account identities, Library data, record IDs, and local home paths from publishable documents.

Raw evidence belongs in an ignored `private/` directory inside the review date directory. It may include original internal reports, scanner output, test logs, and detailed reproductions. Ignored local files are not a durable backup or an access-control boundary: protect them on the host and use approved private storage when sharing is necessary. Do not force-add them to Git or attach them to public issues/PRs.

The Git ignore rules allow reviewed Markdown summaries under `docs/qa/`. Under `docs/security/`, only the named process guides are allowlisted; other files and all `private/` directories are ignored. Screenshot and other non-Markdown evidence files are excluded. Docker and Vercel upload exclusions also omit these documentation trees so private local evidence is not packaged for deployment.

Regression tests under `internal/` and `scripts/` remain tracked. Keep their fixtures synthetic and their credentials disposable. Add tests asserting secure rejection or preservation behavior when fixing audit findings. Audit reproductions that intentionally demonstrate a vulnerability stay in private evidence until reviewed for disclosure.

Keep incident-specific cleanup status and affected identifiers private. See the [history hygiene guide](history-cleanup.md) for the general process. Credential exposure, if discovered, requires rotation regardless of history removal.
