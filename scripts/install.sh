#!/bin/sh
set -eu

# rsync233 - macOS/Linux installer
# Usage: curl -fsSL https://raw.githubusercontent.com/neko233-com/rsync233/main/scripts/install.sh | sh
# Or:    curl -fsSL .../install.sh | sh -s -- v1.0.0
# Windows: use scripts/install.ps1 instead.

VERSION="${1:-latest}"
BINARY_NAME="rsync233"
REPO="neko233-com/rsync233"

detect_os() {
    case "$(uname -s)" in
        Linux*) echo "linux" ;;
        Darwin*) echo "darwin" ;;
        *) echo "unsupported" ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64) echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *) echo "unsupported" ;;
    esac
}

normalize_version() {
    v="$1"
    v="${v#v}"
    v="${v#V}"
    printf '%s' "$v"
}

download() {
    url="$1"
    dest="$2"
    if command -v curl >/dev/null 2>&1; then
        http_code="$(curl -sSL -o "$dest" -w "%{http_code}" "$url" 2>/dev/null || true)"
        if [ "$http_code" = "200" ]; then
            return 0
        fi

        rm -f "$dest"
        if [ "$http_code" = "404" ]; then
            echo "No GitHub Release asset was found for this install target." >&2
            echo "This installer downloads published release binaries, but this repository does not currently have that release." >&2
            echo "Install from source instead: go install github.com/${REPO}/cmd/${BINARY_NAME}@latest" >&2
            exit 1
        fi

        echo "Download failed from ${url} (HTTP ${http_code:-unknown})." >&2
        exit 1
    elif command -v wget >/dev/null 2>&1; then
        if wget -qO "$dest" "$url"; then
            return 0
        fi

        rm -f "$dest"
        echo "Download failed from ${url}. The repository may not have a published GitHub Release for this version yet." >&2
        echo "Install from source instead: go install github.com/${REPO}/cmd/${BINARY_NAME}@latest" >&2
        exit 1
    else
        echo "curl or wget is required." >&2
        exit 1
    fi
}

install_binary() {
    os="$1"
    arch="$2"
    ver="$3"
    asset="${BINARY_NAME}-${os}-${arch}"
    if [ "$ver" = "latest" ]; then
        url="https://github.com/${REPO}/releases/latest/download/${asset}"
    else
        url="https://github.com/${REPO}/releases/download/v${ver}/${asset}"
    fi

    install_dir="${INSTALL_DIR:-/usr/local/bin}"
    tmpdir="$(mktemp -d)"
    trap 'rm -rf "$tmpdir"' EXIT INT TERM

    echo "Downloading ${url}..."
    download "$url" "${tmpdir}/${BINARY_NAME}"
    chmod +x "${tmpdir}/${BINARY_NAME}"

    if [ -w "$install_dir" ]; then
        mv -f "${tmpdir}/${BINARY_NAME}" "${install_dir}/${BINARY_NAME}"
    else
        sudo mv -f "${tmpdir}/${BINARY_NAME}" "${install_dir}/${BINARY_NAME}"
    fi

    echo "Installed to ${install_dir}/${BINARY_NAME}"
}

OS="$(detect_os)"
ARCH="$(detect_arch)"

if [ "$OS" = "unsupported" ]; then
    echo "Unsupported operating system."
    echo "Windows users: run install.ps1 in PowerShell or CMD."
    echo "  irm https://raw.githubusercontent.com/${REPO}/main/scripts/install.ps1 | iex"
    exit 1
fi

if [ "$ARCH" = "unsupported" ]; then
    echo "Unsupported architecture: $(uname -m)" >&2
    exit 1
fi

if [ "$VERSION" != "latest" ] && [ -n "$VERSION" ]; then
    VERSION="$(normalize_version "$VERSION")"
else
    VERSION="latest"
fi

echo "Detected: ${OS}/${ARCH}"
echo "Installing ${BINARY_NAME} (${VERSION})..."
install_binary "$OS" "$ARCH" "$VERSION"
echo "Installed successfully. Run: rsync233 version"
