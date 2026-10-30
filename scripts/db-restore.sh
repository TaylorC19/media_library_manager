#!/bin/sh
set -eu

if [ $# -lt 1 ] || [ -z "${1:-}" ]; then
	echo "usage: $0 <mongodump-archive-dir>" >&2
	echo "  Restores into MONGODB_DATABASE from a mongodump output directory (contains a folder named after the database)." >&2
	exit 2
fi

ARCHIVE=$1
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)

if [ -f "$REPO_ROOT/.env" ]; then
	# shellcheck disable=SC1091
	. "$REPO_ROOT/.env"
fi

MONGODB_URI="${MONGODB_URI:-mongodb://localhost:27017}"
MONGODB_DATABASE="${MONGODB_DATABASE:-media_library}"

if [ ! -d "$ARCHIVE" ]; then
	echo "archive directory not found: $ARCHIVE" >&2
	exit 1
fi

if ! command -v mongorestore >/dev/null 2>&1; then
	echo "mongorestore not found. Install MongoDB Database Tools." >&2
	exit 1
fi

RESTORE_PATH="$ARCHIVE"
if [ -d "$ARCHIVE/$MONGODB_DATABASE" ]; then
	RESTORE_PATH="$ARCHIVE/$MONGODB_DATABASE"
fi

echo "WARNING: this will drop and replace collections in database $MONGODB_DATABASE on $MONGODB_URI"
echo "Restoring from $RESTORE_PATH"
mongorestore --uri="$MONGODB_URI" --db="$MONGODB_DATABASE" --drop "$RESTORE_PATH"
echo "restore complete"
