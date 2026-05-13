# Migration archive (split app removal)

During the Go rewrite, this repository temporarily kept a **reference-only** Next.js frontend (`apps/web`) and NestJS API (`apps/api`), plus shared TypeScript packages (`packages/types`, `packages/provider-sdk`, `packages/config`).

That split stack is **removed** once the root-level Go application provides replacement product behavior. The Go app owns:

- routing, sessions, and HTML rendering (`cmd/web`, `internal/http`, `internal/views`)
- Mongo persistence (`internal/repository`)
- provider integrations (`internal/providers`, `internal/service/*`)

## What was removed

- `apps/web` — Next.js UI (replaced by `html/template` and static assets under `internal/`)
- `apps/api` — NestJS API (replaced by Go handlers and services)
- `packages/*` — TypeScript shared code used only by the old apps
- Root Node toolchain — `package.json`, pnpm workspace, Turbo, ESLint/TS configs tied to the old workspace

## Historical reference

If you need to compare behavior with the old implementation, use git history for commits prior to this migration. The canonical forward-looking spec is [`spec.md`](./spec.md).
