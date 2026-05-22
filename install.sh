#!/usr/bin/env sh
set -eu

REPO="${TEAMS_QUIET_HOURS_REPO:-OcheOps/teams-quiet-hours}"
INSTALL_DIR="${TEAMS_QUIET_HOURS_INSTALL_DIR:-/usr/local/bin}"
BIN_NAME="teams-quiet-hours"

need() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "Missing required command: $1" >&2
    exit 1
  }
}

download() {
  url="$1"
  dest="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$dest"
  elif command -v wget >/dev/null 2>&1; then
    wget -q "$url" -O "$dest"
  else
    echo "Install curl or wget and try again." >&2
    exit 1
  fi
}

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"

case "$os" in
  linux) platform="linux" ;;
  darwin) platform="darwin" ;;
  *)
    echo "Unsupported OS: $os" >&2
    exit 1
    ;;
esac

case "$arch" in
  x86_64|amd64) goarch="amd64" ;;
  aarch64|arm64) goarch="arm64" ;;
  *)
    echo "Unsupported architecture: $arch" >&2
    exit 1
    ;;
esac

asset="teams-quiet-hours-${platform}-${goarch}"
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

base_url="https://github.com/${REPO}/releases/latest/download"
echo "Downloading ${asset} from GitHub Releases..."
download "${base_url}/${asset}" "${tmpdir}/${BIN_NAME}"
download "${base_url}/checksums.txt" "${tmpdir}/checksums.txt"

if command -v sha256sum >/dev/null 2>&1; then
  (cd "$tmpdir" && grep "  ${asset}\$" checksums.txt | sha256sum -c -)
elif command -v shasum >/dev/null 2>&1; then
  (cd "$tmpdir" && grep "  ${asset}\$" checksums.txt | shasum -a 256 -c -)
else
  echo "Warning: could not verify checksum because sha256sum/shasum is unavailable." >&2
fi

chmod +x "${tmpdir}/${BIN_NAME}"

echo "Installing to ${INSTALL_DIR}/${BIN_NAME}..."
if [ "$(id -u)" -eq 0 ]; then
  mkdir -p "$INSTALL_DIR"
  cp "${tmpdir}/${BIN_NAME}" "${INSTALL_DIR}/${BIN_NAME}"
else
  sudo mkdir -p "$INSTALL_DIR"
  sudo cp "${tmpdir}/${BIN_NAME}" "${INSTALL_DIR}/${BIN_NAME}"
fi

echo
echo "Starting guided setup. This needs Administrator/root access because browser policies and schedulers are system settings."
if [ "$(id -u)" -eq 0 ]; then
  "${INSTALL_DIR}/${BIN_NAME}" setup
else
  sudo "${INSTALL_DIR}/${BIN_NAME}" setup
fi

echo
echo "Installed. Run 'teams-quiet-hours doctor' to check the setup."

