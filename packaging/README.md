# Packaging

The project is release-first: average users should install prebuilt assets instead of building from source.

Current release assets:

- Linux amd64/arm64 binaries
- macOS amd64/arm64 binaries
- Windows amd64 binary
- `install.sh` for Linux/macOS
- `install.ps1` for Windows
- SHA-256 checksums
- Debian packages for amd64/arm64

Build locally:

```sh
VERSION=0.4.0 ./scripts/build.sh
VERSION=0.4.0 ./packaging/deb/build-deb.sh
./scripts/checksums.sh
```

`dpkg-deb` is required for Debian packages.

Roadmap:

- `.rpm` packages for Fedora/RHEL
- Optional APT repository
- Homebrew tap
- macOS `.pkg` installer
- Windows MSI or winget manifest

