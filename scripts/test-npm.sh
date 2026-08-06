#!/bin/sh

set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repository_root=$(CDPATH= cd -- "$script_dir/.." && pwd)
version=0.0.0-development
temp_dir=$(mktemp -d 2>/dev/null || mktemp -d -t etherscan-npm-test)
trap 'rm -rf "$temp_dir"' EXIT HUP INT TERM

fixture_dir="$temp_dir/fixtures"
bundle_dir="$temp_dir/bundle"
prefix_dir="$temp_dir/global prefix"
ignored_prefix="$temp_dir/ignored prefix"
invalid_prefix="$temp_dir/invalid prefix"
mkdir -p "$fixture_dir" "$bundle_dir"

case "$(uname -s)" in
    Linux) os=linux ;;
    Darwin) os=darwin ;;
    *) printf 'unsupported test OS\n' >&2; exit 1 ;;
esac
case "$(uname -m)" in
    x86_64|amd64) arch=amd64 ;;
    arm64|aarch64) arch=arm64 ;;
    *) printf 'unsupported test architecture\n' >&2; exit 1 ;;
esac

archive_name="etherscan_${version}_${os}_${arch}.tar.gz"
archive_path="$fixture_dir/$archive_name"
checksum_path="$fixture_dir/checksums.txt"
export GOCACHE="$temp_dir/go-cache"
export npm_config_cache="$temp_dir/npm-cache"

(cd "$repository_root" && npm run test:release)
(cd "$repository_root" && go build -buildvcs=false -ldflags "-X main.version=$version" -o "$bundle_dir/etherscan" ./cmd/etherscan)
tar -czf "$archive_path" -C "$bundle_dir" etherscan
if command -v sha256sum >/dev/null 2>&1; then
    hash=$(sha256sum "$archive_path" | awk '{ print tolower($1) }')
else
    hash=$(shasum -a 256 "$archive_path" | awk '{ print tolower($1) }')
fi
printf '%s  %s\n' "$hash" "$archive_name" >"$checksum_path"

tarball_name=$(cd "$repository_root" && npm pack --pack-destination "$temp_dir" | tail -n 1)
tarball="$temp_dir/$tarball_name"
export ETHERSCAN_INSTALL_TEST_DOWNLOAD_BASE_URL="$fixture_dir"

npm install --global --prefix "$prefix_dir" "$tarball"
[ "$("$prefix_dir/bin/etherscan" version)" = "$version" ] || { printf 'global npm installation returned an unexpected version\n' >&2; exit 1; }

[ "$(npx --yes --package "$tarball" etherscan version)" = "$version" ] || { printf 'npx returned an unexpected version\n' >&2; exit 1; }

if "$prefix_dir/bin/etherscan" definitely-not-a-command >/dev/null 2>&1; then
    printf 'npm launcher did not forward a failing command or its exit code\n' >&2
    exit 1
fi

npm install --global --ignore-scripts --prefix "$ignored_prefix" "$tarball"
if "$ignored_prefix/bin/etherscan" version >/dev/null 2>&1; then
    printf 'launcher succeeded after lifecycle scripts were disabled\n' >&2
    exit 1
fi

if (cd "$repository_root" && ETHERSCAN_INSTALL_TEST_ARCH=unsupported node npm/postinstall.js) >/dev/null 2>&1; then
    printf 'npm postinstall accepted an unsupported architecture\n' >&2
    exit 1
fi

printf '%064d  %s\n' 0 "$archive_name" >"$checksum_path"
if npm install --global --prefix "$invalid_prefix" "$tarball" >/dev/null 2>&1; then
    printf 'npm installation accepted an invalid release checksum\n' >&2
    exit 1
fi

printf 'npm package tests passed.\n'
