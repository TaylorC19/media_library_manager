# Migration from the legacy monorepo

This repository previously included a **Next.js** frontend (`apps/web`), a **NestJS** API (`apps/api`), and shared **TypeScript** packages used only by that stack.

That split architecture is **removed**. The shipping product is the **single Go application** at the repository root:

- HTTP, auth, and HTML: `cmd/web`, `internal/http`, `internal/views`
- Persistence: `internal/repository`
- Providers and domain logic: `internal/providers`, `internal/service`

## What went away

- `apps/web` — replaced by `html/template` and static assets under `internal/static`
- `apps/api` — replaced by Go handlers and services
- `packages/*` and the root Node/pnpm/Turbo toolchain — only relevant historically

## Comparing to old behavior

If you need to inspect the previous implementation, use **git history** from before those directories were deleted.

Current product and architecture expectations are documented in [`docs/spec.md`](spec.md) and [`docs/architecture.md`](architecture.md).
