#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)

cd "$REPO_ROOT"

export APP_ENV="${APP_ENV:-development}"
export PORT="${PORT:-8080}"
export DEFAULT_LOCALE="${DEFAULT_LOCALE:-en}"
export MONGODB_URI="${MONGODB_URI:-mongodb://localhost:27017}"
export MONGODB_DATABASE="${MONGODB_DATABASE:-media_library}"

exec go run ./cmd/web
