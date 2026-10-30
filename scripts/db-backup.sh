#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)

if [ -f "$REPO_ROOT/.env" ]; then
	# shellcheck disable=SC1091
	. "$REPO_ROOT/.env"
fi

MONGODB_URI="${MONGODB_URI:-mongodb://localhost:27017}"
MONGODB_DATABASE="${MONGODB_DATABASE:-media_library}"

if ! command -v mongodump >/dev/null 2>&1; then
	echo "mongodump not found. Install MongoDB Database Tools or run via Docker:" >&2
	echo "  docker compose exec mongo mongodump --uri=... --db=... --out=/tmp/backup" >&2
	exit 1
fi

TIMESTAMP=$(date -u +%Y%m%dT%H%M%SZ)
OUT_DIR="$REPO_ROOT/backups/mongodump-$TIMESTAMP"
mkdir -p "$REPO_ROOT/backups"

mongodump --uri="$MONGODB_URI" --db="$MONGODB_DATABASE" --out="$OUT_DIR"
echo "backup written to $OUT_DIR"
