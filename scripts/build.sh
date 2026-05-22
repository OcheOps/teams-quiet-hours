#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${ROOT_DIR}/dist"
VERSION="${VERSION:-dev}"

mkdir -p "$OUT_DIR"

build_one() {
  local goos="$1"
  local goarch="$2"
  local ext=""
  if [[ "$goos" == "windows" ]]; then
    ext=".exe"
  fi
  echo "building ${goos}/${goarch}"
  GOOS="$goos" GOARCH="$goarch" go build -buildvcs=false \
    -ldflags "-s -w -X main.version=${VERSION}" \
    -o "${OUT_DIR}/teams-quiet-hours-${goos}-${goarch}${ext}" \
    ./cmd/teams-quiet-hours
}

build_one linux amd64
build_one linux arm64
build_one darwin amd64
build_one darwin arm64
build_one windows amd64
