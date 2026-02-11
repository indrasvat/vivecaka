#!/usr/bin/env bash
set -euo pipefail

# vivecaka installer
# Usage: curl -sSfL https://raw.githubusercontent.com/indrasvat/vivecaka/main/install.sh | bash
#        curl ... | bash -s -- --version v0.1.1 --dir ~/.local/bin

REPO="indrasvat/vivecaka"
BINARY="vivecaka"
DEFAULT_DIR="${HOME}/.local/bin"

# --- Colors (Catppuccin Mocha palette) ----------------------------------------

setup_colors() {
    if [[ -n "${NO_COLOR:-}" ]] || [[ ! -t 1 ]]; then
        BOLD=""
        RESET=""
        MAUVE=""
        BLUE=""
        GREEN=""
        RED=""
        YELLOW=""
        TEXT=""
        SUBTEXT=""
    else
        BOLD=$'\033[1m'
        RESET=$'\033[0m'
        MAUVE=$'\033[38;2;203;166;247m'
        BLUE=$'\033[38;2;137;180;250m'
        GREEN=$'\033[38;2;166;227;161m'
        RED=$'\033[38;2;243;139;168m'
        YELLOW=$'\033[38;2;249;226;175m'
        TEXT=$'\033[38;2;205;214;244m'
        SUBTEXT=$'\033[38;2;166;173;200m'
    fi
}

banner() {
    local cols
    cols=$(tput cols 2>/dev/null || echo 80)
    if [[ "${cols}" -lt 80 ]]; then
        return
    fi
    printf '\n'
    printf '%s' "${MAUVE}"
    printf '  ██▄  ▄██   ████     ██▄  ▄██   ▄████▄    ▄█████▄   ▄█████▄  ██ ▄██▀    ▄█████▄\n'
    printf '   ██  ██      ██      ██  ██   ██▄▄▄▄██  ██▀    ▀   ▀ ▄▄▄██  ██▄██      ▀ ▄▄▄██\n'
    printf '   ▀█▄▄█▀      ██      ▀█▄▄█▀   ██▀▀▀▀▀▀  ██        ▄██▀▀▀██  ██▀██▄    ▄██▀▀▀██\n'
    printf '    ████    ▄▄▄██▄▄▄    ████    ▀██▄▄▄▄█  ▀██▄▄▄▄█  ██▄▄▄███  ██  ▀█▄   ██▄▄▄███\n'
    printf '     ▀▀     ▀▀▀▀▀▀▀▀     ▀▀       ▀▀▀▀▀     ▀▀▀▀▀    ▀▀▀▀ ▀▀  ▀▀   ▀▀▀   ▀▀▀▀ ▀▀\n'
    printf '%s\n' "${RESET}"
}

info()       { printf '  %s→%s %s%s%s\n' "${BLUE}" "${RESET}" "${TEXT}" "$1" "${RESET}"; }
success()    { printf '  %s✓%s %s%s%s\n' "${GREEN}" "${RESET}" "${TEXT}" "$1" "${RESET}"; }
warn()       { printf '  %s! %s%s\n' "${YELLOW}" "$1" "${RESET}"; }
error_exit() { printf '  %s✗ %s%s\n' "${RED}" "$1" "${RESET}" >&2; exit 1; }

step() {
    local n="$1" total="$2" msg="$3"
    printf '\n%s%s[%s/%s]%s %s%s%s%s\n' "${BOLD}" "${MAUVE}" "${n}" "${total}" "${RESET}" "${BOLD}" "${TEXT}" "${msg}" "${RESET}"
}

# --- Argument parsing ---------------------------------------------------------

usage() {
    printf '%s%svivecaka installer%s\n\n' "${BOLD}" "${TEXT}" "${RESET}"
    printf '%sUsage:%s\n' "${SUBTEXT}" "${RESET}"
    printf '  curl -sSfL https://raw.githubusercontent.com/%s/main/install.sh | bash\n' "${REPO}"
    printf '  curl ... | bash -s -- [OPTIONS]\n\n'
    printf '%sOptions:%s\n' "${SUBTEXT}" "${RESET}"
    printf '  %s--version VERSION%s  Install specific version (e.g. v0.1.1)\n' "${TEXT}" "${RESET}"
    printf '  %s--dir DIRECTORY%s    Install directory (default: %s)\n' "${TEXT}" "${RESET}" "${DEFAULT_DIR}"
    printf '  %s--help%s             Show this help\n' "${TEXT}" "${RESET}"
    exit 0
}

parse_args() {
    VERSION=""
    INSTALL_DIR="${DEFAULT_DIR}"

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --version)
                [[ $# -lt 2 ]] && error_exit "--version requires a value"
                VERSION="$2"
                shift 2
                ;;
            --dir)
                [[ $# -lt 2 ]] && error_exit "--dir requires a value"
                INSTALL_DIR="$2"
                shift 2
                ;;
            --help)
                usage
                ;;
            *)
                error_exit "Unknown option: $1 (use --help for usage)"
                ;;
        esac
    done
}

# --- Dependency checks --------------------------------------------------------

check_dependencies() {
    if command -v curl >/dev/null 2>&1; then
        DOWNLOADER="curl"
        success "Using curl for downloads"
    elif command -v wget >/dev/null 2>&1; then
        DOWNLOADER="wget"
        success "Using wget for downloads"
    else
        error_exit "curl or wget is required"
    fi

    if command -v shasum >/dev/null 2>&1; then
        HASHER="shasum"
        success "Using shasum for verification"
    elif command -v sha256sum >/dev/null 2>&1; then
        HASHER="sha256sum"
        success "Using sha256sum for verification"
    else
        error_exit "shasum or sha256sum is required"
    fi
}

# --- Platform detection -------------------------------------------------------

detect_platform() {
    local os arch

    os="$(uname -s)"
    case "${os}" in
        Darwin) OS="darwin" ;;
        Linux)  error_exit "Linux support coming soon. Follow ${REPO} for updates." ;;
        *)      error_exit "Unsupported operating system: ${os}" ;;
    esac

    arch="$(uname -m)"
    case "${arch}" in
        x86_64)  ARCH="amd64" ;;
        arm64)   ARCH="arm64" ;;
        aarch64) ARCH="arm64" ;;
        *)       error_exit "Unsupported architecture: ${arch}" ;;
    esac

    success "Platform: ${OS}/${ARCH}"
}

# --- Version resolution -------------------------------------------------------

get_latest_version() {
    local url="https://api.github.com/repos/${REPO}/releases/latest"
    local response

    if [[ "${DOWNLOADER}" == "curl" ]]; then
        response="$(curl -sSfL "${url}" 2>/dev/null)" || error_exit "Failed to query GitHub releases"
    else
        response="$(wget -qO- "${url}" 2>/dev/null)" || error_exit "Failed to query GitHub releases"
    fi

    VERSION="$(echo "${response}" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/' || true)"
    [[ -z "${VERSION}" ]] && error_exit "Could not determine latest version"
}

# --- Download helpers ---------------------------------------------------------

build_download_url() {
    local version_no_v="${VERSION#v}"
    TARBALL="vivecaka_${version_no_v}_${OS}_${ARCH}.tar.gz"
    TARBALL_URL="https://github.com/${REPO}/releases/download/${VERSION}/${TARBALL}"
    CHECKSUMS_URL="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"
}

download_file() {
    local url="$1" dest="$2"
    if [[ "${DOWNLOADER}" == "curl" ]]; then
        curl -sSfL -o "${dest}" "${url}" 2>/dev/null
    else
        wget -q -O "${dest}" "${url}" 2>/dev/null
    fi
}

# --- Checksum verification ----------------------------------------------------

verify_checksum() {
    local checksums_file="$1" tarball_file="$2"
    local expected actual

    expected="$(grep "${TARBALL}" "${checksums_file}" | awk '{print $1}' || true)"
    [[ -z "${expected}" ]] && error_exit "Checksum not found for ${TARBALL} in checksums.txt"

    if [[ "${HASHER}" == "shasum" ]]; then
        actual="$(shasum -a 256 "${tarball_file}" | awk '{print $1}')"
    else
        actual="$(sha256sum "${tarball_file}" | awk '{print $1}')"
    fi

    if [[ "${expected}" != "${actual}" ]]; then
        error_exit "Checksum mismatch! Expected ${expected}, got ${actual}"
    fi
}

# --- Installation -------------------------------------------------------------

install_binary() {
    local tmpdir="$1"
    tar -xzf "${tmpdir}/${TARBALL}" -C "${tmpdir}"
    mkdir -p "${INSTALL_DIR}"
    install -m 755 "${tmpdir}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
}

check_path() {
    case ":${PATH}:" in
        *":${INSTALL_DIR}:"*) return ;;
    esac

    warn "${INSTALL_DIR} is not in your PATH"

    local shell_name rc_file
    shell_name="$(basename "${SHELL:-/bin/bash}")"
    # shellcheck disable=SC2088  # tilde is intentional for display
    case "${shell_name}" in
        zsh)  rc_file="~/.zshrc" ;;
        bash) rc_file="~/.bashrc" ;;
        fish) rc_file="~/.config/fish/config.fish" ;;
        *)    rc_file="your shell config" ;;
    esac

    info "Add this to ${rc_file}:"
    # shellcheck disable=SC2016  # $PATH is literal display text, not expansion
    printf '\n  %sexport PATH="%s:$PATH"%s\n\n' "${SUBTEXT}" "${INSTALL_DIR}" "${RESET}"
}

# --- Cleanup ------------------------------------------------------------------

cleanup() {
    if [[ -n "${TMPDIR_CREATED:-}" ]]; then
        rm -rf "${TMPDIR_CREATED}"
    fi
}

# --- Main ---------------------------------------------------------------------

main() {
    setup_colors
    banner
    parse_args "$@"

    step 1 6 "Checking dependencies"
    check_dependencies

    step 2 6 "Detecting platform"
    detect_platform

    step 3 6 "Finding latest release"
    if [[ -n "${VERSION}" ]]; then
        success "Version: ${VERSION} (requested)"
    else
        get_latest_version
        success "Version: ${VERSION}"
    fi

    build_download_url

    local tmpdir
    tmpdir="$(mktemp -d)"
    TMPDIR_CREATED="${tmpdir}"
    trap cleanup EXIT

    step 4 6 "Downloading ${BINARY}"
    download_file "${TARBALL_URL}" "${tmpdir}/${TARBALL}" \
        || error_exit "Download failed. Check that ${VERSION} exists at github.com/${REPO}/releases"
    success "Downloaded ${TARBALL}"

    step 5 6 "Verifying checksum"
    download_file "${CHECKSUMS_URL}" "${tmpdir}/checksums.txt" \
        || error_exit "Failed to download checksums"
    verify_checksum "${tmpdir}/checksums.txt" "${tmpdir}/${TARBALL}"
    success "Checksum verified (SHA-256)"

    step 6 6 "Installing to ${INSTALL_DIR}"
    install_binary "${tmpdir}"
    success "Installed ${BINARY} ${VERSION}"

    # macOS: clear Gatekeeper quarantine flag
    if [[ "${OS}" == "darwin" ]]; then
        xattr -d com.apple.quarantine "${INSTALL_DIR}/${BINARY}" 2>/dev/null || true
        success "Cleared macOS quarantine flag"
    fi

    printf '\n  %s✓%s %s%sInstallation complete!%s\n\n' "${GREEN}" "${RESET}" "${BOLD}" "${TEXT}" "${RESET}"

    check_path

    info "Run ${BOLD}vivecaka${RESET}${TEXT} in any GitHub repo to get started"
    info "Configuration: ~/.config/vivecaka/config.toml"
    printf '\n'
}

main "$@"
