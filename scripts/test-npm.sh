#!/bin/sh

set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repository_root=$(CDPATH= cd -- "$script_dir/.." && pwd)
version=0.0.0-development
temp_dir=$(mktemp -d 2>/dev/null || mktemp -d -t etherscan-npm-test)
trap 'rm -rf "$temp_dir"' EXIT HUP INT TERM

case "$(uname -s)" in
    Linux) npm_os=linux ;;
    Darwin) npm_os=darwin ;;
    *) printf 'unsupported test OS\n' >&2; exit 1 ;;
esac
case "$(uname -m)" in
    x86_64|amd64) npm_arch=x64 ;;
    arm64|aarch64) npm_arch=arm64 ;;
    *) printf 'unsupported test architecture\n' >&2; exit 1 ;;
esac

platform_name="cli-${npm_os}-${npm_arch}"
platform_stage="$temp_dir/$platform_name"
bundle_dir="$temp_dir/bundle"
prefix_dir="$temp_dir/global prefix"
missing_prefix="$temp_dir/missing prefix"
mkdir -p "$platform_stage" "$bundle_dir"
export GOCACHE="$temp_dir/go-cache"
export npm_config_cache="$temp_dir/npm-cache"

(cd "$repository_root" && npm run test:release)
(cd "$repository_root" && go build -buildvcs=false -ldflags "-X main.version=$version" -o "$bundle_dir/etherscan" ./cmd/etherscan)
cp "$repository_root/npm/$platform_name/package.json" "$platform_stage/package.json"
cp "$repository_root/LICENSE" "$platform_stage/LICENSE"
cp "$bundle_dir/etherscan" "$platform_stage/etherscan"
chmod 0755 "$platform_stage/etherscan"

platform_tarball_name=$(npm pack "$platform_stage" --pack-destination "$temp_dir" | tail -n 1)
umbrella_tarball_name=$(cd "$repository_root" && npm pack --pack-destination "$temp_dir" | tail -n 1)
platform_tarball="$temp_dir/$platform_tarball_name"
umbrella_tarball="$temp_dir/$umbrella_tarball_name"

npm install --global --ignore-scripts --prefix "$prefix_dir" "$platform_tarball" "$umbrella_tarball"
[ "$("$prefix_dir/bin/etherscan" version)" = "$version" ] || { printf 'global npm installation returned an unexpected version\n' >&2; exit 1; }

[ "$(npx --yes --package "$platform_tarball" --package "$umbrella_tarball" etherscan version)" = "$version" ] || { printf 'npx returned an unexpected version\n' >&2; exit 1; }

if "$prefix_dir/bin/etherscan" definitely-not-a-command >/dev/null 2>&1; then
    printf 'npm launcher did not forward a failing command or its exit code\n' >&2
    exit 1
fi

npm install --global --ignore-scripts --omit=optional --prefix "$missing_prefix" "$umbrella_tarball"
missing_output=$("$missing_prefix/bin/etherscan" version 2>&1 || true)
printf '%s' "$missing_output" | grep -F -- 'without --omit=optional' >/dev/null || {
    printf 'missing platform package did not produce an actionable error: %s\n' "$missing_output" >&2
    exit 1
}

printf 'npm package tests passed.\n'
