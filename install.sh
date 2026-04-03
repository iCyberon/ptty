#!/bin/sh
set -e

REPO="iCyberon/ptty"
BINARY="ptty"

main() {
    printf "\n  \033[1;34mptty installer\033[0m\n\n"

    need_cmd curl
    need_cmd tar

    detect_platform
    get_latest_version

    printf "  Platform:  \033[1m%s/%s\033[0m\n" "$OS" "$ARCH"
    printf "  Version:   \033[1m%s\033[0m\n\n" "$VERSION"

    choose_install_dir
    printf "\n"

    download_and_install

    printf "\n  \033[1;32mInstalled %s %s to %s\033[0m\n\n" "$BINARY" "$VERSION" "$INSTALL_DIR/$BINARY"

    if ! command -v "$BINARY" > /dev/null 2>&1; then
        printf "  \033[33mNote:\033[0m %s is not in your PATH.\n" "$INSTALL_DIR"
        printf "  Add it with:\n\n"
        printf "    export PATH=\"%s:\$PATH\"\n\n" "$INSTALL_DIR"
    fi
}

detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case "$OS" in
        darwin) OS="darwin" ;;
        linux)  OS="linux" ;;
        *)      err "Unsupported OS: $OS. Use install.ps1 for Windows." ;;
    esac

    case "$ARCH" in
        x86_64|amd64)  ARCH="amd64" ;;
        arm64|aarch64) ARCH="arm64" ;;
        *)             err "Unsupported architecture: $ARCH" ;;
    esac
}

get_latest_version() {
    VERSION=$(curl -sSf "https://api.github.com/repos/$REPO/releases/latest" \
        | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"//;s/".*//')

    if [ -z "$VERSION" ]; then
        err "Failed to fetch latest version from GitHub"
    fi
}

choose_install_dir() {
    default_dir="/usr/local/bin"
    alt_dir="$HOME/.local/bin"

    if [ "$(id -u)" = "0" ]; then
        default_dir="/usr/local/bin"
    elif [ ! -w "/usr/local/bin" ] 2>/dev/null; then
        default_dir="$alt_dir"
    fi

    printf "  Where should %s be installed?\n\n" "$BINARY"
    printf "    1) %s" "$default_dir"
    if [ "$default_dir" = "$alt_dir" ]; then
        printf " (default)\n"
        printf "    2) /usr/local/bin (requires sudo)\n"
    else
        printf " (default)\n"
        printf "    2) %s\n" "$alt_dir"
    fi
    printf "    3) Custom path\n\n"
    printf "  Choice [1]: "
    read -r choice < /dev/tty

    case "$choice" in
        ""|1) INSTALL_DIR="$default_dir" ;;
        2)
            if [ "$default_dir" = "$alt_dir" ]; then
                INSTALL_DIR="/usr/local/bin"
            else
                INSTALL_DIR="$alt_dir"
            fi
            ;;
        3)
            printf "  Path: "
            read -r custom_path < /dev/tty
            INSTALL_DIR="$custom_path"
            ;;
        *)  INSTALL_DIR="$default_dir" ;;
    esac

    mkdir -p "$INSTALL_DIR" 2>/dev/null || true
    if [ ! -w "$INSTALL_DIR" ]; then
        printf "\n  \033[33m%s requires elevated permissions.\033[0m\n" "$INSTALL_DIR"
        printf "  Re-run with: curl -sSL <url> | sudo sh\n"
        exit 1
    fi
}

download_and_install() {
    archive_name="${BINARY}_${OS}_${ARCH}.tar.gz"
    checksum_name="checksums.txt"
    tag_version=$(echo "$VERSION" | sed 's/^v//')
    download_url="https://github.com/$REPO/releases/download/$VERSION"

    tmp=$(mktemp -d)
    trap 'rm -rf "$tmp"' EXIT

    printf "  Downloading %s..." "$archive_name"
    curl -sSfL "$download_url/$archive_name" -o "$tmp/$archive_name"
    printf " \033[32mdone\033[0m\n"

    printf "  Verifying checksum..."
    curl -sSfL "$download_url/$checksum_name" -o "$tmp/$checksum_name"

    expected=$(grep "$archive_name" "$tmp/$checksum_name" | awk '{print $1}')
    if [ -z "$expected" ]; then
        err "Checksum not found for $archive_name"
    fi

    if command -v sha256sum > /dev/null 2>&1; then
        actual=$(sha256sum "$tmp/$archive_name" | awk '{print $1}')
    elif command -v shasum > /dev/null 2>&1; then
        actual=$(shasum -a 256 "$tmp/$archive_name" | awk '{print $1}')
    else
        printf " \033[33mskipped\033[0m (no sha256sum/shasum)\n"
        actual="$expected"
    fi

    if [ "$actual" != "$expected" ]; then
        err "Checksum mismatch!\n  Expected: $expected\n  Got:      $actual"
    fi
    printf " \033[32mdone\033[0m\n"

    printf "  Extracting..."
    tar -xzf "$tmp/$archive_name" -C "$tmp"
    printf " \033[32mdone\033[0m\n"

    printf "  Installing to %s..." "$INSTALL_DIR"
    mv "$tmp/$BINARY" "$INSTALL_DIR/$BINARY"
    chmod +x "$INSTALL_DIR/$BINARY"
    printf " \033[32mdone\033[0m\n"
}

need_cmd() {
    if ! command -v "$1" > /dev/null 2>&1; then
        err "Required command not found: $1"
    fi
}

err() {
    printf "\n  \033[1;31mError:\033[0m %b\n\n" "$1" >&2
    exit 1
}

main
