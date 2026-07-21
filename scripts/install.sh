#!/bin/sh

set -eu

repository="etherscan/etherscan-cli"
version=${ETHERSCAN_VERSION:-}
install_dir=${ETHERSCAN_INSTALL_DIR:-"$HOME/.local/bin"}
download_base=${ETHERSCAN_INSTALL_TEST_DOWNLOAD_BASE_URL:-}
update_path=1

usage() {
    cat <<'EOF'
Install Etherscan CLI.

Usage: install.sh [options]

Options:
  --version VERSION         Install a specific version (for example, v1.1.0).
  --install-dir DIRECTORY   Install into DIRECTORY (default: ~/.local/bin).
  --no-path-update          Do not update the shell profile.
  -h, --help                Show this help.
EOF
}

die() {
    printf 'error: %s\n' "$*" >&2
    exit 1
}

while [ "$#" -gt 0 ]; do
    case "$1" in
        --version)
            [ "$#" -ge 2 ] || die "--version requires a value"
            version=$2
            shift 2
            ;;
        --install-dir)
            [ "$#" -ge 2 ] || die "--install-dir requires a value"
            install_dir=$2
            shift 2
            ;;
        --no-path-update)
            update_path=0
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            die "unknown option: $1"
            ;;
    esac
done

if printf '%s' "$install_dir" | LC_ALL=C grep '[[:cntrl:]]' >/dev/null 2>&1; then
    die "the installation directory cannot contain control characters"
fi

fetch_stdout() {
    url=$1
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL -A etherscan-cli-installer "$url"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- --user-agent=etherscan-cli-installer "$url"
    else
        die "curl or wget is required"
    fi
}

fetch_file() {
    base=$1
    name=$2
    destination=$3

    if [ -d "$base" ]; then
        cp "$base/$name" "$destination"
        return
    fi

    case "$base" in
        file://*)
            cp "${base#file://}/$name" "$destination"
            ;;
        https://*)
            if command -v curl >/dev/null 2>&1; then
                curl -fsSL -A etherscan-cli-installer "$base/$name" -o "$destination"
            elif command -v wget >/dev/null 2>&1; then
                wget -q --user-agent=etherscan-cli-installer "$base/$name" -O "$destination"
            else
                die "curl or wget is required"
            fi
            ;;
        *)
            die "invalid download base URL or directory: $base"
            ;;
    esac
}

if [ -z "$version" ] || [ "$version" = latest ]; then
    [ -z "$download_base" ] || die "a version is required with the installer test download source"
    release_json=$(fetch_stdout "https://api.github.com/repos/$repository/releases/latest")
    version=$(printf '%s\n' "$release_json" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | sed -n '1p')
    [ -n "$version" ] || die "could not resolve the latest Etherscan CLI version"
fi

case "$version" in
    v*) tag=$version; release_version=${version#v} ;;
    *) tag="v$version"; release_version=$version ;;
esac

printf '%s\n' "$tag" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+([-+][0-9A-Za-z.-]+)?$' || die "invalid release version: $tag"

system_name=${ETHERSCAN_INSTALL_TEST_OS:-$(uname -s)}
case "$system_name" in
    Linux|linux) os=linux ;;
    Darwin|darwin) os=darwin ;;
    *) die "unsupported operating system: $system_name" ;;
esac

machine_arch=${ETHERSCAN_INSTALL_TEST_ARCH:-$(uname -m)}
case "$machine_arch" in
    x86_64|amd64) arch=amd64 ;;
    arm64|aarch64) arch=arm64 ;;
    *) die "unsupported architecture: $machine_arch. Etherscan CLI supports amd64 and arm64." ;;
esac

archive_name="etherscan_${release_version}_${os}_${arch}.tar.gz"
if [ -z "$download_base" ]; then
    download_base="https://github.com/$repository/releases/download/$tag"
fi

temp_dir=$(mktemp -d 2>/dev/null || mktemp -d -t etherscan-install)
trap 'rm -rf "$temp_dir"' EXIT HUP INT TERM
archive_path="$temp_dir/$archive_name"
checksum_path="$temp_dir/checksums.txt"
source_executable="$temp_dir/etherscan"

printf 'Downloading Etherscan CLI %s for %s/%s...\n' "$release_version" "$os" "$arch"
fetch_file "$download_base" "$archive_name" "$archive_path"
fetch_file "$download_base" checksums.txt "$checksum_path"

expected_hash=$(awk -v name="$archive_name" '$2 == name || $2 == ("*" name) { print tolower($1); exit }' "$checksum_path")
[ -n "$expected_hash" ] || die "no checksum was published for $archive_name"
printf '%s\n' "$expected_hash" | grep -Eq '^[0-9a-f]{64}$' || die "invalid checksum published for $archive_name"

if command -v sha256sum >/dev/null 2>&1; then
    actual_hash=$(sha256sum "$archive_path" | awk '{ print tolower($1) }')
elif command -v shasum >/dev/null 2>&1; then
    actual_hash=$(shasum -a 256 "$archive_path" | awk '{ print tolower($1) }')
else
    die "sha256sum or shasum is required to verify the download"
fi

[ "$actual_hash" = "$expected_hash" ] || die "checksum verification failed for $archive_name"

entry_count=$(tar -tzf "$archive_path" | awk '$0 == "etherscan" { count++ } END { print count + 0 }')
[ "$entry_count" -eq 1 ] || die "$archive_name must contain exactly one root-level etherscan"
tar -xOzf "$archive_path" etherscan >"$source_executable"
[ -s "$source_executable" ] || die "$archive_name contains an empty etherscan executable"

mkdir -p "$install_dir"
staged_executable="$install_dir/.etherscan.new.$$"
cp "$source_executable" "$staged_executable"
chmod 0755 "$staged_executable"
mv -f "$staged_executable" "$install_dir/etherscan"

path_updated=0
if [ "$update_path" -eq 1 ]; then
    case ":$PATH:" in
        *:"$install_dir":*) ;;
        *)
            shell_name=${SHELL:-sh}
            shell_name=${shell_name##*/}
            escaped_install_dir=$(printf '%s' "$install_dir" | sed 's/[\\"$`]/\\&/g')
            if [ "$shell_name" = fish ]; then
                profile="$HOME/.config/fish/config.fish"
                mkdir -p "$(dirname "$profile")"
                path_line="fish_add_path \"$escaped_install_dir\""
            else
                case "$shell_name" in
                    zsh) profile="$HOME/.zshrc" ;;
                    bash) profile="$HOME/.bashrc" ;;
                    *) profile="$HOME/.profile" ;;
                esac
                path_line="export PATH=\"$escaped_install_dir:\$PATH\""
            fi

            if ! [ -f "$profile" ] || ! grep -F -e "$install_dir" "$profile" >/dev/null 2>&1; then
                {
                    printf '\n# Etherscan CLI\n'
                    printf '%s\n' "$path_line"
                } >>"$profile"
                path_updated=1
            fi
            ;;
    esac
fi

printf '\nEtherscan CLI %s installed successfully.\n' "$release_version"
printf 'Installed to: %s\n' "$install_dir/etherscan"
if [ "$update_path" -eq 0 ]; then
    printf 'Add %s to PATH to run etherscan from any directory.\n' "$install_dir"
elif [ "$path_updated" -eq 1 ]; then
    printf 'Open a new terminal, then run: etherscan version\n'
else
    printf 'Run: etherscan version\n'
fi
