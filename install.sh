#!/bin/sh

set -eu

REPOSITORY="etherscan/etherscan-cli"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

fail() {
  printf 'etherscan installer: %s\n' "$*" >&2
  exit 1
}

command -v curl >/dev/null 2>&1 || fail "curl is required"
command -v tar >/dev/null 2>&1 || fail "tar is required"

case "$(uname -s)" in
  Darwin) os=darwin ;;
  Linux) os=linux ;;
  *) fail "only macOS and Linux are supported by this installer" ;;
esac

case "$(uname -m)" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) fail "unsupported architecture: $(uname -m)" ;;
esac

if [ -n "${VERSION:-}" ]; then
  case "$VERSION" in
    v*) tag=$VERSION ;;
    *) tag="v$VERSION" ;;
  esac
else
  tag=$(curl -fsSL "https://api.github.com/repos/$REPOSITORY/releases/latest" |
    sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')
fi

[ -n "$tag" ] || fail "could not determine the latest release"
version=${tag#v}
archive="etherscan_${version}_${os}_${arch}.tar.gz"
base_url="https://github.com/$REPOSITORY/releases/download/$tag"
tmp_dir=$(mktemp -d)
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

printf 'Downloading etherscan %s for %s/%s...\n' "$tag" "$os" "$arch"
curl -fsSL "$base_url/$archive" -o "$tmp_dir/$archive"
curl -fsSL "$base_url/checksums.txt" -o "$tmp_dir/checksums.txt"

expected=$(awk -v name="$archive" '$2 == name { print $1 }' "$tmp_dir/checksums.txt")
[ -n "$expected" ] || fail "release checksum not found for $archive"

if command -v sha256sum >/dev/null 2>&1; then
  actual=$(sha256sum "$tmp_dir/$archive" | awk '{ print $1 }')
elif command -v shasum >/dev/null 2>&1; then
  actual=$(shasum -a 256 "$tmp_dir/$archive" | awk '{ print $1 }')
else
  fail "sha256sum or shasum is required to verify the download"
fi

[ "$actual" = "$expected" ] || fail "checksum verification failed"
tar -xzf "$tmp_dir/$archive" -C "$tmp_dir"
[ -f "$tmp_dir/etherscan" ] || fail "release archive does not contain etherscan"

if [ ! -d "$INSTALL_DIR" ]; then
  mkdir -p "$INSTALL_DIR" 2>/dev/null ||
    fail "cannot create $INSTALL_DIR; set INSTALL_DIR to a writable directory"
fi

if [ ! -w "$INSTALL_DIR" ]; then
  fail "$INSTALL_DIR is not writable; rerun with sudo or set INSTALL_DIR"
fi

install -m 0755 "$tmp_dir/etherscan" "$INSTALL_DIR/etherscan"
printf 'Installed etherscan to %s/etherscan\n' "$INSTALL_DIR"
