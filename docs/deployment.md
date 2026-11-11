# Installation and deployment

Docker Compose is the primary self-hosting path. Docker builds the Go binaries
inside the image; installers do not need Go, Make, GitHub Actions, or a GitHub
fork. Download a source archive or clone the repository and run commands from
its root. The default `docker-compose.yml` remains a database-only development
stack; installation uses `docker-compose.prod.yml` explicitly.

## Install the published Docker bundle

Download an architecture-matched Docker bundle and `SHA256SUMS` from
[the alpha release](https://github.com/TaylorC19/media_library_manager/releases/tag/v0.1.0-alpha.1). The bundle includes prebuilt app/setup images,
Compose files and a README. Verify the checksum, extract the bundle, then follow
its guide: `docker load --input images.tar`, run the versioned setup image's
`init` command, and `docker compose --env-file .env.prod up -d`. No source checkout
or build is required. Docker Desktop also uses the Linux bundles; standalone
setup executables have separate Linux/macOS/Windows archives.

These Docker images are release downloads rather than container-registry URLs.
Use `docker load`; there is no registry sign-in step. For external databases or
Vercel, load the bundle's setup image or download the standalone setup executable.

## Build from source (optional)

1. Build the installer:

   ```sh
   docker build --target setup -t media-library-manager-setup:local .
   ```

2. Generate private configuration. Replace the example origin with your real
   HTTPS address. These commands use a macOS/Linux shell:

   ```sh
   docker run --rm --user "$(id -u):$(id -g)" \
     --mount "type=bind,source=$PWD,target=/config" \
     media-library-manager-setup:local init \
     -dir /config -origin https://library.example.com
   ```

   This creates owner-only `.env.prod`, `.env.setup`, and
   `.bootstrap-admin-password` files with random credentials and an encryption
   key. It refuses to overwrite existing files. `.env.prod` contains only the
   runtime database credentials; the maintenance credentials stay in `.env.setup`.
   Keep these files private and backed up. Add optional provider credentials to
   `.env.prod`; API keys can also be configured later in Settings.

3. Start the installation:

   ```sh
   docker compose --env-file .env.prod -f docker-compose.prod.yml up --build -d
   ```

   Compose starts an authenticated single-node MongoDB replica set on a private
   internal network. The one-shot setup service initializes the replica set,
   creates all collections and indexes, provisions a CRUD-only database user,
   and creates the initial application administrator. The app starts only after
   setup exits successfully. Setup runs as root only to read the owner-only bind
   secret; the web application runs as a non-root user with runtime credentials.
   MongoDB has no host port, and the web container has no maintenance credentials.

4. Connect your HTTPS reverse proxy to `127.0.0.1:8080`, then open the exact URL
   supplied to `init`. The initial application username is `admin`; read its
   password locally from `.bootstrap-admin-password` and store it in your
   password manager.
   Do not publish the password file or paste it into logs.

TLS termination is provided by your reverse proxy, not this Compose file.
Plain HTTP access will not support the production Secure session cookie.
`PORT` controls the host port. `BIND_ADDRESS` defaults to loopback; change it
only for an intentional proxy/network configuration. The app uses port 8080
inside the container and requires no source or template mounts.

## Restarting, updating and stopping

```sh
docker compose --env-file .env.prod -f docker-compose.prod.yml logs --tail 50 setup app
docker compose --env-file .env.prod -f docker-compose.prod.yml up --build -d
docker compose --env-file .env.prod -f docker-compose.prod.yml down
```

Repeat setup is safe: it preserves records and existing administrator/database
passwords. Back up first when updating. Do not run `init` again for an existing
installation, replace the generated credentials, or delete volumes to solve a
startup failure. A failed setup blocks app creation on a fresh start; stop the
app explicitly before running maintenance during an upgrade. `down` preserves
the `mongo-prod-data` and `mongo-config` volumes; `down -v` deletes them.
A single-node replica set enables transactions but is not a high-availability
cluster. Existing manual/legacy deployments need a planned migration; the
installer is for fresh installations and does not adopt unauthenticated volumes.

## Existing external MongoDB with Compose

Create `.env.prod` from `.env.prod.example` with runtime database credentials,
HTTPS origin, encryption key, secure cookies and `ENSURE_INDEXES=false`. Create
an owner-only `.env.setup` from `.env.setup.example` using separate maintenance
credentials for the same database. Use an existing replica set or Atlas cluster.

```sh
docker compose --env-file .env.prod -f docker-compose.external.yml run --build --rm setup
docker compose --env-file .env.prod -f docker-compose.external.yml run --rm -i setup bootstrap-admin -username admin
docker compose --env-file .env.prod -f docker-compose.external.yml up --build -d
```

Use `--build` on the first `run` if the setup image is not present. Supply a
hidden administrator password when prompted. The external path never creates
Atlas users or changes its replica-set configuration; provision runtime and
maintenance database users through your database host. Setup credentials must
have permissions to create collections/indexes and perform the bootstrap writes.
The runtime account needs CRUD access to the app's collections, not database
administration rights.

## Vercel (optional managed deployment)

Download the matching installer from [the alpha release](https://github.com/TaylorC19/media_library_manager/releases/tag/v0.1.0-alpha.1)
and run it from your workstation; do not compile Go locally
or configure a GitHub workflow. See [the setup-tool instructions](setup-tool.md)
for Docker and standalone executable commands. A private `.env.setup` requires
only `MONGODB_URI` (maintenance credentials) and `MONGODB_DATABASE`.

1. Create an Atlas/replica-set database and separate maintenance/runtime users.
2. Allow your workstation to connect while running setup. Configure the hosted
   app's database network access separately.
3. Run `mlm-setup database -env-file .env.setup`, then
   `mlm-setup bootstrap-admin -env-file .env.setup -username admin`. With Docker,
   use the read-only environment-file mount and hidden prompt shown in the
   setup-tool guide. No app secrets are needed for database initialization.
4. Set Vercel runtime variables from `.env.prod.example`: `APP_ENV=production`,
   `PORT=8080`, exact HTTPS `PUBLIC_ORIGIN`, `SESSION_COOKIE_SECURE=true`,
   `SESSION_COOKIE_NAME=__Host-mlm_session`, runtime `MONGODB_URI`,
   `MONGODB_DATABASE`, and `PROVIDER_KEYS_ENCRYPTION_KEY`. Generate the last
   value with `openssl rand -base64 32` and retain it separately from backups.
   Keep `SIGNUP_ENABLED=false`, `BACKUP_ENABLED=false`, `ENSURE_INDEXES=false`.
   Do not upload `.env.setup` or use its credentials in the web runtime.
5. Deploy from the repository root. The checked-in `vercel.json` selects the
   Container preset. Vercel builds the root Dockerfile's final `app` stage;
   the runtime settings from the preceding step configure its production mode
   and HTTP port. All templates and assets are
   embedded, and the MongoDB connection pool is reused and bounded. No manual
   JSON edits or custom Go build command are required.

Production configuration is explicit; the server does not load `.env.prod` for
you. Preview and Production need separate databases, credentials and encryption
keys. Set each `PUBLIC_ORIGIN` to the exact URL used to access that environment;
incorrect origins reject writes. Keep previews protected. Vercel networking,
function duration and database access must be checked for your plan. See the
[Vercel Container Runtime documentation](https://vercel.com/docs/functions/runtimes/container) and
[Static IP documentation](https://vercel.com/docs/networking/static-ips).

## Vercel Container Runtime with an external database

The repository defaults to Vercel's Container Runtime (beta). Initialize the
external database using the packaged setup tool first, following the preceding
Vercel steps. The Compose database and setup services do not run inside Vercel.

Import the repository with its root as the project root and use the checked-in
`vercel.json` as-is. It selects the Container preset. Set `PORT=8080` in the
Vercel project's environment settings so the router and web process use the
same port. The Dockerfile's final `app` stage is the
HTTP service; the maintenance `setup` stage runs separately on your workstation.

For an existing project previously configured for native Go, clear any dashboard
build-command or output-directory overrides and redeploy `main`. Confirm the
project framework is Container. Database credentials, HTTPS origin and encryption
key belong in the project's runtime environment settings, not in source control.

Set the production runtime variables listed above: runtime `MONGODB_URI`,
`MONGODB_DATABASE`, `PUBLIC_ORIGIN`, `PROVIDER_KEYS_ENCRYPTION_KEY`, secure session
cookie settings, `APP_ENV=production`, and disabled startup indexes/signup/backups.
Use the exact HTTPS origin users will visit. Keep the maintenance database URI on
your workstation. Choose a function region near the database and configure its
network access for Vercel's egress. Verify login, persistence and headers on the
actual deployment; no cloud project is configured by these instructions.

This documented path builds the Dockerfile from repository source and stores the
image in Vercel's registry. The public Docker download bundles are for Docker
hosts and local setup; they are not uploaded directly through `docker load` to
Vercel. See [Container Runtime](https://vercel.com/docs/functions/runtimes/container)
for current availability, port handling and platform limits.

## Database contents, secrets and recovery

The setup tool creates collections for users, sessions, media records, library
entries, scan logs, provider cache, request budgets, request leases, admin guard,
and encrypted personal provider keys. No collections or seed records need to be
created by hand. Existing data is not automatically moved into a new database.

Personal provider overrides use owner/provider-bound AES-256-GCM in MongoDB,
with the encryption key held in deployment secrets. Keep that key when moving
or restoring a database. Local development file-based overrides are not migrated
by the installer. Managed database snapshots include encrypted overrides;
custom app archives exclude them. Web database backup/restore is disabled in
production. See [backup and restore](backup.md) for maintenance procedures.

Production uses shared MongoDB budgets: provider limits of 60 requests/minute
and 1,000/day per user, at most eight concurrent provider requests per user,
and 500 writes/day, plus authentication limits. Before making a deployment
public, verify HTTPS login/logout, roles, origin rejection, search dialogs,
provider settings persistence, camera scanning, and logout/back navigation.
Local tests do not verify a cloud account's secrets or network configuration.

## Development

For host Go development, copy `.env.example` to `.env`, run
`./scripts/dev-db-up.sh`, then `./scripts/dev-web.sh`. The default Compose file
runs MongoDB only on localhost port 27017. Development reads assets from source;
production embeds them. Developer-only `go run ./cmd/dbsetup` and `cmd/users`
remain available for maintenance, but are not required for installation.
