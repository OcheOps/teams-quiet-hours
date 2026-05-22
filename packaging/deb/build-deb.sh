#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
VERSION="${VERSION:-0.0.0}"
OUT_DIR="${ROOT_DIR}/dist"

command -v dpkg-deb >/dev/null 2>&1 || {
  echo "dpkg-deb is required to build Debian packages." >&2
  exit 127
}

build_deb() {
  local arch="$1"
  local deb_arch="$2"
  local binary="${OUT_DIR}/teams-quiet-hours-linux-${arch}"
  local pkgroot
  pkgroot="$(mktemp -d)"

  mkdir -p "${pkgroot}/DEBIAN" "${pkgroot}/usr/local/bin" "${pkgroot}/usr/share/doc/teams-quiet-hours"
  cp "$binary" "${pkgroot}/usr/local/bin/teams-quiet-hours"
  chmod 0755 "${pkgroot}/usr/local/bin/teams-quiet-hours"
  cp "${ROOT_DIR}/README.md" "${pkgroot}/usr/share/doc/teams-quiet-hours/README.md"
  cp "${ROOT_DIR}/LICENSE" "${pkgroot}/usr/share/doc/teams-quiet-hours/LICENSE"

  cat > "${pkgroot}/DEBIAN/control" <<EOF
Package: teams-quiet-hours
Version: ${VERSION}
Section: utils
Priority: optional
Architecture: ${deb_arch}
Maintainer: teams-quiet-hours contributors
Description: Quiet Microsoft Teams outside work hours
 A small utility that silences or blocks Microsoft Teams distractions outside
 configured work hours without muting the whole machine.
EOF

  dpkg-deb --build --root-owner-group "$pkgroot" "${OUT_DIR}/teams-quiet-hours_${VERSION}_${deb_arch}.deb"
  rm -rf "$pkgroot"
}

build_deb amd64 amd64
build_deb arm64 arm64
