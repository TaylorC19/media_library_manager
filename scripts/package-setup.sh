#!/bin/sh
# Maintainer packaging command; installers use its binaries without a Go toolchain.
set -eu
version=${1:-dev}
case "$version" in ''|*[!A-Za-z0-9._-]*) echo 'Version may contain only letters, digits, dot, underscore and hyphen.' >&2; exit 2;; esac
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
out=${SETUP_DIST_DIR:-"$root/dist/setup"}
mkdir -p "$out"
for platform in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64; do
  target_os=${platform%/*}
  target_arch=${platform#*/}
  stage=$(mktemp -d)
  suffix=
  if [ "$target_os" = windows ]; then suffix=.exe; fi
  CGO_ENABLED=0 GOOS="$target_os" GOARCH="$target_arch" go build -trimpath -ldflags "-s -w -X main.version=$version" -o "$stage/mlm-setup$suffix" "$root/cmd/setup"
  cp "$root/docs/setup-tool.md" "$stage/README.md"
  cp "$root/.env.setup.example" "$stage/setup.env.example"
  archive="mlm-setup_${version}_${target_os}_${target_arch}.tar.gz"
  COPYFILE_DISABLE=1 tar -C "$stage" -czf "$out/$archive" "mlm-setup$suffix" README.md setup.env.example
  rm -rf "$stage"
done
(cd "$out"; if command -v sha256sum >/dev/null 2>&1; then sha256sum mlm-setup_"$version"_*.tar.gz; else shasum -a 256 mlm-setup_"$version"_*.tar.gz; fi) > "$out/SHA256SUMS"
echo "Setup archives and SHA256SUMS created in $out"
