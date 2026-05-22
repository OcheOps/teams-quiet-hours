#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${ROOT_DIR}/dist"

cd "$OUT_DIR"
rm -f checksums.txt
if command -v sha256sum >/dev/null 2>&1; then
  sha256sum * > checksums.txt
else
  shasum -a 256 * > checksums.txt
fi
