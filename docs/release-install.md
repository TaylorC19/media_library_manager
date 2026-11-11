# Install Media Library Manager @VERSION@

This bundle contains prebuilt Linux Docker images for the app and setup tool,
Compose files, and the MongoDB startup script. It needs Docker with Compose;
no Go installation, source checkout, registry login, or GitHub Actions is needed.
On macOS and Windows, Docker Desktop runs these Linux images. Choose the bundle
matching Docker's architecture (amd64/x86_64 or arm64/aarch64).

Before extracting, verify this archive against the release's SHA256SUMS.
On macOS use `shasum -a 256`; on Linux use `sha256sum`.

## Fresh self-hosted installation

Extract the archive and open a terminal inside its directory. In a macOS/Linux shell:

```sh
docker load --input images.tar
docker run --rm --user "$(id -u):$(id -g)" \
  --mount "type=bind,source=$PWD,target=/config" \
  media-library-manager-setup:@VERSION@ init \
  -dir /config -origin https://library.example.com
docker compose --env-file .env.prod up -d
```

Replace the origin with the exact HTTPS address you will use. Configure an HTTPS
reverse proxy to forward to 127.0.0.1:8080. The stack exposes no MongoDB host port.
On Windows PowerShell, use `${PWD}` in the bind mount and `--user 0:0` instead
of the Unix `id` expression; run the Docker commands on a single line.

Setup creates the authenticated replica set, collections, indexes, a restricted
runtime database user and the initial app administrator. The app starts after
setup succeeds. The username is `admin`; its random password is in the private
`.bootstrap-admin-password` file. No passwords are printed. Keep that file,
`.env.prod`, `.env.setup`, and both MongoDB volumes backed up securely.
Never regenerate credentials or delete volumes to repair an existing install.

Check startup using `docker compose --env-file .env.prod logs --tail 50 setup app`.
Stop using `docker compose --env-file .env.prod down` (without `-v` to keep data).
Setup is repeatable and never resets existing account or database passwords.
The `init` command is only for a fresh installation and refuses to overwrite files.
Existing installations need a backup and a planned upgrade; this bundle does not
automatically adopt older unauthenticated MongoDB volumes.

## Vercel or another external database

Load the images as above, then create an owner-only `.env.setup` from the supplied
`.env.setup.example` containing maintenance `MONGODB_URI` and `MONGODB_DATABASE`.
The database must already be an Atlas/replica-set cluster with network access
from your workstation. These commands do not reconfigure its replica set:

```sh
docker run --rm --user "$(id -u):$(id -g)" \
  --mount "type=bind,source=$PWD/.env.setup,target=/config/setup.env,readonly" \
  media-library-manager-setup:@VERSION@ database -env-file /config/setup.env

docker run --rm -it --user "$(id -u):$(id -g)" \
  --mount "type=bind,source=$PWD/.env.setup,target=/config/setup.env,readonly" \
  media-library-manager-setup:@VERSION@ bootstrap-admin -env-file /config/setup.env -username admin
```

The password prompt is hidden. Alternatively, use the release's standalone
`mlm-setup` executable for your OS/architecture; its archive includes a guide.
Deploy the web app separately with runtime credentials and production settings
from `.env.prod.example`. Keep maintenance credentials off the web runtime.
See https://github.com/TaylorC19/media_library_manager/blob/v@VERSION@/docs/deployment.md
for HTTPS, encryption keys, database permissions, previews and recovery details.

For Compose with an external database, configure `.env.prod` and `.env.setup`,
then use `docker compose --env-file .env.prod -f compose.external.yml` instead
of the default Compose file. Bootstrap the first administrator explicitly.

Docker images here are distributed as downloads: `docker load` makes them
available locally. They are not hosted at a container-registry pull URL.
