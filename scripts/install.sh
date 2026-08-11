#!/bin/sh

set -eu

repository="etherscan/etherscan-cli"
version=${ETHERSCAN_VERSION:-}
install_dir=${ETHERSCAN_INSTALL_DIR:-"$HOME/.local/bin"}
download_base=${ETHERSCAN_INSTALL_TEST_DOWNLOAD_BASE_URL:-}
update_path=1
uninstall=0
marker_name=.etherscan-cli-path-added
marker_content=etherscan-cli:path-added:v1

usage() {
    cat <<'EOF'
Install Etherscan CLI.

Usage: install.sh [options]

Options:
  --version VERSION         Install a specific version (for example, v1.1.0).
  --install-dir DIRECTORY   Install into DIRECTORY (default: ~/.local/bin).
  --no-path-update          Do not update the shell profile.
  --uninstall               Remove the CLI and saved configuration.
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
        --uninstall)
            uninstall=1
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

config_dir() {
    if [ -n "${XDG_CONFIG_HOME:-}" ]; then
        printf '%s\n' "$XDG_CONFIG_HOME/etherscan"
    else
        printf '%s\n' "$HOME/.etherscan"
    fi
}

profile_has_block() {
    profile=$1
    target=$2
    [ -f "$profile" ] && [ ! -L "$profile" ] || return 1
    awk -v target="$target" '
        previous == "# Etherscan CLI" && $0 == target { found = 1; exit }
        { previous = $0 }
        END { exit found ? 0 : 1 }
    ' "$profile"
}

remove_profile_block() {
    profile=$1
    target=$2
    profile_has_block "$profile" "$target" || return 0
    tmp=$(mktemp "${profile}.etherscan-uninstall.XXXXXX") || die "could not create a profile temporary file"
    if ! cp -p "$profile" "$tmp"; then
        rm -f "$tmp"
        die "could not preserve permissions for $profile"
    fi
    if ! awk -v target="$target" '
        {
            if (pending) {
                if ($0 == target) { pending = 0; next }
                print "# Etherscan CLI"
                pending = 0
            }
            if ($0 == "# Etherscan CLI") { pending = 1; next }
            print
        }
        END { if (pending) print "# Etherscan CLI" }
    ' "$profile" >"$tmp"; then
        rm -f "$tmp"
        die "could not update $profile"
    fi
    mv -f "$tmp" "$profile"
    printf 'Removed the PATH entry from %s\n' "$profile"
}

directory_empty_except_marker() {
    directory=$1
    marker=$2
    [ -d "$directory" ] || return 0
    for entry in "$directory"/.[!.]* "$directory"/..?* "$directory"/*; do
        [ -e "$entry" ] || [ -L "$entry" ] || continue
        [ "$entry" = "$marker" ] && continue
        return 1
    done
    return 0
}

if [ "$uninstall" -eq 1 ]; then
    binary="$install_dir/etherscan"
    marker="$install_dir/$marker_name"
    marker_valid=0
    if [ -f "$marker" ] && [ ! -L "$marker" ] && [ "$(cat "$marker")" = "$marker_content" ]; then
        marker_valid=1
    fi

    escaped_install_dir=$(printf '%s' "$install_dir" | sed 's/[\\"$`]/\\&/g')
    posix_path_line="export PATH=\"$escaped_install_dir:\$PATH\""
    fish_path_line="fish_add_path \"$escaped_install_dir\""
    legacy_provenance=0
    for candidate in "$HOME/.zshrc" "$HOME/.bashrc" "$HOME/.profile"; do
        if profile_has_block "$candidate" "$posix_path_line"; then
            legacy_provenance=1
        fi
    done
    if profile_has_block "$HOME/.config/fish/config.fish" "$fish_path_line"; then
        legacy_provenance=1
    fi

    removed=0
    if [ -e "$binary" ] || [ -L "$binary" ]; then
        rm -f "$binary"
        printf 'Removed %s\n' "$binary"
        removed=1
    fi

    if [ "$update_path" -eq 1 ] && { [ "$marker_valid" -eq 1 ] || [ "$legacy_provenance" -eq 1 ]; } && directory_empty_except_marker "$install_dir" "$marker"; then
        for candidate in "$HOME/.zshrc" "$HOME/.bashrc" "$HOME/.profile"; do
            remove_profile_block "$candidate" "$posix_path_line"
        done
        remove_profile_block "$HOME/.config/fish/config.fish" "$fish_path_line"
        if [ "$marker_valid" -eq 1 ]; then
            rm -f "$marker"
        fi
        rmdir "$install_dir" 2>/dev/null || true
        removed=1
    elif [ "$update_path" -eq 1 ] && [ -d "$install_dir" ]; then
        printf 'Left %s on PATH because ownership was not proven or the directory is shared.\n' "$install_dir"
    fi

    etherscan_config=$(config_dir)
    if [ -e "$etherscan_config" ] || [ -L "$etherscan_config" ]; then
        rm -rf "$etherscan_config"
        printf 'Removed %s\n' "$etherscan_config"
        removed=1
    fi
    if [ "$removed" -eq 1 ]; then
        printf 'Etherscan CLI uninstalled.\n'
    else
        printf 'Nothing to remove.\n'
    fi
    if [ -n "${ETHERSCAN_API_KEY:-}" ]; then
        printf 'note: ETHERSCAN_API_KEY remains set; unset it in your shell.\n' >&2
    fi
    exit 0
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

            # Match the exact line we would write (whole-line, fixed-string) so an
            # unrelated profile line that merely contains the path does not suppress
            # the update, and a genuine duplicate is not appended.
            if ! [ -f "$profile" ] || ! grep -Fx -e "$path_line" "$profile" >/dev/null 2>&1; then
                {
                    printf '\n# Etherscan CLI\n'
                    printf '%s\n' "$path_line"
                } >>"$profile"
                path_updated=1
                printf '%s' "$marker_content" >"$install_dir/$marker_name"
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
