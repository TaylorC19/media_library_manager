# Database backup, export, and import

Media Library Manager stores all application data in MongoDB. You can back up and restore the database in two ways:

1. **Ops scripts** — `mongodump` / `mongorestore` (standard MongoDB tools)
2. **App format** — gzip JSON archives from Settings, `cmd/db`, or the Go `internal/backup` package

Both approaches cover the **full database**: `users`, `sessions`, `library_entries`, `media_records`, `provider_cache`, and `scan_logs`.

## When to use which

| Approach | Best for |
|----------|----------|
| `make db-backup` / `mongodump` | Cron jobs, copying Docker volumes, Atlas migrations, very large databases |
| Settings download / `cmd/db export` | Single-binary deployments (production Docker image has no `mongodump`), quick portable archives |

App backups use the format `media_library_manager.backup` version `1` (`.mlm-backup.json.gz`).

## Security

Backup files include **password hashes** and **session tokens**. Treat them like secrets. Do not commit them to git (`backups/` is gitignored).

Personal API keys configured in Settings are stored outside MongoDB, in `PROVIDER_KEYS_DIR`. Database export/restore does not include or modify them. Back up that private directory separately; see [provider credentials](provider-notes.md#configure-api-keys-in-settings).

## Settings UI

1. Sign in as an **admin** and open **Settings**. General and demo accounts cannot access full-database export or restore, including direct requests.
   Promote the intended account using the server-side `cmd/users` command documented in the README.
2. **Download backup** — exports the full database as a gzip file.
3. **Restore from backup** — choose a `.mlm-backup.json.gz` file and type the database name (`MONGODB_DATABASE`, shown on the form) to confirm.

Import **replaces all data** in the configured database. Sign in again after restoring on a new instance if sessions do not match your cookie.

Disable UI backup with `BACKUP_ENABLED=false` (see [`.env.example`](../.env.example)).

## CLI (`cmd/db`)

```bash
go run ./cmd/db export -o ./backups/export.mlm-backup.json.gz
go run ./cmd/db import -i ./backups/export.mlm-backup.json.gz --confirm media_library
```

Uses `MONGODB_URI` and `MONGODB_DATABASE` from `.env` / environment (same as the web app).

## Ops scripts

```bash
make db-backup
# writes to backups/mongodump-<timestamp>/

make db-restore ARCHIVE=backups/mongodump-20260101T120000Z
```

Requires [MongoDB Database Tools](https://www.mongodb.com/docs/database-tools/) on the host, or run `mongodump` inside the Compose `mongo` container.

Scripts read `MONGODB_URI` and `MONGODB_DATABASE` from `.env` when present.

## Restore checklist

1. **Back up first** — export or `make db-backup` before any restore.
2. **Verify database name** — `MONGODB_DATABASE` must match the backup (app import enforces this).
3. **Stop or quiesce the app** (recommended) — avoid writes during `mongorestore` or app import.
4. Run restore (Settings, `cmd/db import`, or `make db-restore`).
5. **Re-login** if sessions were reset or you moved to a new host.

## Configuration

| Variable | Default | Purpose |
|----------|---------|---------|
| `BACKUP_ENABLED` | `true` | Show backup UI and allow export/import routes |
| `BACKUP_MAX_UPLOAD_BYTES` | `104857600` (100MB) | Max upload size for Settings import |

## Large databases

For very large catalogs, prefer `mongodump` and filesystem backups. App export loads collections into memory for JSON encoding; increase `BACKUP_MAX_UPLOAD_BYTES` only if uploads require it.
