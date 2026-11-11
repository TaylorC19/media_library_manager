# Packaged database setup tool

`mlm-setup` is a maintenance tool for installing Media Library Manager. Installers
need either Docker or an executable for their operating system, not Go or GitHub
Actions. It does not deploy the web app or require a GitHub fork.

## Docker

Download and extract the Docker bundle for your Docker architecture from
[the alpha release](https://github.com/TaylorC19/media_library_manager/releases/tag/v0.1.0-alpha.1), verify its checksum, then load the images:

```sh
docker load --input images.tar
```

To run against an external database, create
an owner-only `.env.setup` from `.env.setup.example`, containing the maintenance
`MONGODB_URI` and `MONGODB_DATABASE`. Then run on macOS/Linux:

```sh
docker run --rm --user "$(id -u):$(id -g)" \
  --mount "type=bind,source=$PWD/.env.setup,target=/config/setup.env,readonly" \
  media-library-manager-setup:0.1.0-alpha.1 database -env-file /config/setup.env

docker run --rm -it --user "$(id -u):$(id -g)" \
  --mount "type=bind,source=$PWD/.env.setup,target=/config/setup.env,readonly" \
  media-library-manager-setup:0.1.0-alpha.1 bootstrap-admin -env-file /config/setup.env -username admin
```

The administrator password prompt is hidden and asks for confirmation. For
noninteractive use, pipe a password from a private source with `-password-stdin`
and Docker's `-i` flag. Passwords are never accepted as command-line arguments.

## Standalone executable

Use the archive matching your operating system and architecture from the
[release assets](https://github.com/TaylorC19/media_library_manager/releases/tag/v0.1.0-alpha.1). Verify its SHA-256 against the release's
`SHA256SUMS`, then extract it. macOS and Linux use `./mlm-setup`; Windows uses
`.\mlm-setup.exe` (Windows tar can extract `.tar.gz` archives).

Copy the bundled `setup.env.example` to `.env.setup`, restrict its permissions,
and enter your maintenance connection string and database name. On macOS/Linux:

```sh
cp setup.env.example .env.setup
chmod 600 .env.setup
# Edit .env.setup before continuing.
./mlm-setup database -env-file .env.setup
./mlm-setup bootstrap-admin -env-file .env.setup -username admin
```

The tool reads only database configuration. Process environment overrides values
in the explicitly supplied file; it never automatically loads a development
`.env`. Unset unrelated `MONGODB_*` variables before switching installations.
Database credentials and secrets are not printed. Keep the maintenance URI off
the Vercel runtime and out of source control.

## Commands

- `init -origin https://library.example.com -dir .`: generate fresh private
  Compose configuration and a random initial administrator password. It refuses
  to overwrite existing files, including symlinks.
- `database`: create required collections, indexes, TTL cleanup and the admin
  transaction guard. Safe to repeat; it does not restore or replace records.
- `bootstrap-admin`: create the initial app administrator only if no admin
  exists. It never resets passwords or promotes an existing account. Run
  `database` first; bootstrap is transaction-protected.
- `version`: print the packaged version.

`-init-replica` and `-provision-runtime` are only for the bundled Compose database;
never use them against Atlas or another managed cluster. An existing standalone
MongoDB instance must be migrated to a replica set separately. The setup tool
will not reconfigure arbitrary deployments.

## Updates and recovery

Run the version-matched setup image or executable before deploying an updated
app, after taking a backup. It applies repeatable index setup and leaves library
data and existing passwords intact. If setup fails, fix the cause and rerun it;
Compose will not start the app after a failed setup service.

Keep `.env.prod`, `.env.setup`, `.bootstrap-admin-password`, MongoDB data and the
MongoDB config volume backed up privately. Keep the provider encryption key with
your deployment secrets as well. Do not regenerate credentials for existing
volumes: recover the original files. Database user password rotation and app
administrator recovery are separate maintenance operations.

## Maintainer packaging

```sh
./scripts/package-setup.sh 0.1.0
```

This produces Linux, macOS and Windows executables for amd64/arm64, each in a
versioned archive with instructions and an empty environment template, plus
`dist/setup/SHA256SUMS`. The output is ignored by Git. Publish reviewed archives
and checksums as release assets; building packages does not publish them.
A setup image can be built with `--target setup --build-arg VERSION=0.1.0`.

For a source build instead of a download:

```sh
docker build --target setup --build-arg VERSION=0.1.0-alpha.1 -t media-library-manager-setup:0.1.0-alpha.1 .
```

Maintainers can prepare both Docker bundles and all native installers with
`python3 scripts/package-docker-release.py 0.1.0-alpha.1`. This builds the app
and setup images for linux/amd64 and linux/arm64, saves them into versioned Docker
bundles with build-free Compose files, and writes a combined `SHA256SUMS`. Publish
only those reviewed assets; never attach local environment or QA files.
