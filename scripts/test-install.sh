#!/bin/sh

set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
installer="$script_dir/install.sh"
temp_dir=$(mktemp -d 2>/dev/null || mktemp -d -t etherscan-installer-test)
trap 'rm -rf "$temp_dir"' EXIT HUP INT TERM

fixture_dir="$temp_dir/fixtures"
bundle_dir="$temp_dir/bundle"
install_dir="$temp_dir/install dir"
version=9.9.9-test.1

case "$(uname -s)" in
    Linux) os=linux ;;
    Darwin) os=darwin ;;
    MINGW*|MSYS*|CYGWIN*)
        os=linux
        export ETHERSCAN_INSTALL_TEST_OS=linux
        ;;
    *) printf 'unsupported test OS\n' >&2; exit 1 ;;
esac

case "$(uname -m)" in
    x86_64|amd64) arch=amd64 ;;
    arm64|aarch64) arch=arm64 ;;
    *) printf 'unsupported test architecture\n' >&2; exit 1 ;;
esac
export ETHERSCAN_INSTALL_TEST_ARCH=$arch

archive_name="etherscan_${version}_${os}_${arch}.tar.gz"
archive_path="$fixture_dir/$archive_name"
checksum_path="$fixture_dir/checksums.txt"
mkdir -p "$fixture_dir"
export ETHERSCAN_INSTALL_TEST_DOWNLOAD_BASE_URL=$fixture_dir

write_fixture() {
    content=$1
    rm -rf "$bundle_dir"
    mkdir -p "$bundle_dir"
    printf '%s' "$content" >"$bundle_dir/etherscan"
    chmod 0755 "$bundle_dir/etherscan"
    tar -czf "$archive_path" -C "$bundle_dir" etherscan
    if command -v sha256sum >/dev/null 2>&1; then
        hash=$(sha256sum "$archive_path" | awk '{ print tolower($1) }')
    else
        hash=$(shasum -a 256 "$archive_path" | awk '{ print tolower($1) }')
    fi
    printf '%s  %s\n' "$hash" "$archive_name" >"$checksum_path"
}

write_fixture first
sh "$installer" --version "$version" --install-dir "$install_dir" --no-path-update
[ "$(cat "$install_dir/etherscan")" = first ] || { printf 'fresh installation failed\n' >&2; exit 1; }

write_fixture second
sh "$installer" --version "$version" --install-dir "$install_dir" --no-path-update
[ "$(cat "$install_dir/etherscan")" = second ] || { printf 'reinstallation failed\n' >&2; exit 1; }

profile_home="$temp_dir/profile-home"
profile_install_dir="$profile_home/bin with spaces"
mkdir -p "$profile_home"
HOME="$profile_home" SHELL=/bin/sh sh "$installer" --version "$version" --install-dir "$profile_install_dir"
grep -F "$profile_install_dir" "$profile_home/.profile" >/dev/null || { printf 'PATH profile update failed\n' >&2; exit 1; }
HOME="$profile_home" SHELL=/bin/sh sh "$installer" --version "$version" --install-dir "$profile_install_dir"
[ "$(grep -Fc '# Etherscan CLI' "$profile_home/.profile")" -eq 1 ] || { printf 'PATH profile update was not idempotent\n' >&2; exit 1; }

printf '%064d  %s\n' 0 "$archive_name" >"$checksum_path"
if sh "$installer" --version "$version" --install-dir "$install_dir" --no-path-update >/dev/null 2>&1; then
    printf 'installer accepted an invalid checksum\n' >&2
    exit 1
fi
[ "$(cat "$install_dir/etherscan")" = second ] || { printf 'failed verification changed installation\n' >&2; exit 1; }

if ETHERSCAN_INSTALL_TEST_DOWNLOAD_BASE_URL="http://example.invalid" sh "$installer" --version "$version" --install-dir "$install_dir" --no-path-update >/dev/null 2>&1; then
    printf 'installer accepted an insecure download URL\n' >&2
    exit 1
fi

# Uninstall removes a provenanced dedicated installation, its exact profile block, and config.
write_fixture third
uninstall_home="$temp_dir/uninstall-home"
config_home="$temp_dir/config-home"
dedicated_dir="$temp_dir/dedicated dir"
mkdir -p "$uninstall_home" "$config_home/etherscan"
printf 'api_key = "test"\n' >"$config_home/etherscan/config.toml"
HOME="$uninstall_home" SHELL=/bin/sh sh "$installer" --version "$version" --install-dir "$dedicated_dir" >/dev/null
[ -f "$dedicated_dir/.etherscan-cli-path-added" ] || { printf 'installer did not record PATH provenance\n' >&2; exit 1; }
chmod 0600 "$uninstall_home/.profile"
HOME="$uninstall_home" XDG_CONFIG_HOME="$config_home" sh "$installer" --install-dir "$dedicated_dir" --uninstall >/dev/null
[ ! -e "$dedicated_dir" ] || { printf 'provenanced uninstall left its directory\n' >&2; exit 1; }
[ ! -e "$config_home/etherscan" ] || { printf 'uninstall left config behind\n' >&2; exit 1; }
grep -F "$dedicated_dir" "$uninstall_home/.profile" >/dev/null && { printf 'uninstall left PATH block behind\n' >&2; exit 1; }
[ "$(stat -c %a "$uninstall_home/.profile" 2>/dev/null || stat -f %Lp "$uninstall_home/.profile")" = 600 ] || { printf 'profile permissions changed\n' >&2; exit 1; }

# A matching hand-written line without the adjacent marker must survive.
manual_home="$temp_dir/manual-home"
manual_dir="$temp_dir/manual dir"
mkdir -p "$manual_home" "$manual_dir"
printf 'manual\n' >"$manual_dir/etherscan"
manual_line="export PATH=\"$manual_dir:\$PATH\""
printf '%s\n' "$manual_line" >"$manual_home/.profile"
HOME="$manual_home" XDG_CONFIG_HOME="$config_home" sh "$installer" --install-dir "$manual_dir" --uninstall >/dev/null
grep -Fx "$manual_line" "$manual_home/.profile" >/dev/null || { printf 'uninstall removed a hand-written PATH line\n' >&2; exit 1; }
[ -d "$manual_dir" ] || { printf 'uninstall removed an unprovenanced directory\n' >&2; exit 1; }

# A shared directory retains unrelated files, PATH, and provenance.
shared_dir="$temp_dir/shared dir"
HOME="$uninstall_home" SHELL=/bin/sh sh "$installer" --version "$version" --install-dir "$shared_dir" >/dev/null
printf 'keep\n' >"$shared_dir/other"
HOME="$uninstall_home" XDG_CONFIG_HOME="$config_home" sh "$installer" --install-dir "$shared_dir" --uninstall >/dev/null
[ -e "$shared_dir/other" ] || { printf 'shared uninstall removed another file\n' >&2; exit 1; }
[ -e "$shared_dir/.etherscan-cli-path-added" ] || { printf 'shared uninstall removed provenance\n' >&2; exit 1; }
grep -F "$shared_dir" "$uninstall_home/.profile" >/dev/null || { printf 'shared uninstall removed PATH\n' >&2; exit 1; }

# Repeated uninstall is a no-op.
HOME="$uninstall_home" XDG_CONFIG_HOME="$config_home" sh "$installer" --install-dir "$dedicated_dir" --uninstall >/dev/null

printf 'Shell installer tests passed.\n'
