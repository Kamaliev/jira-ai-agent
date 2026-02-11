#!/bin/sh
set -e

REPO="Kamaliev/jira-ai-agent"
BINARY="sj"
INSTALL_DIR="/usr/local/bin"

detect_os() {
  case "$(uname -s)" in
    Linux*)  echo "linux" ;;
    Darwin*) echo "darwin" ;;
    MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
    *) echo "unsupported"; return 1 ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)  echo "amd64" ;;
    aarch64|arm64)  echo "arm64" ;;
    *) echo "unsupported"; return 1 ;;
  esac
}

main() {
  OS=$(detect_os)
  ARCH=$(detect_arch)

  if [ "$OS" = "unsupported" ] || [ "$ARCH" = "unsupported" ]; then
    echo "Error: unsupported platform $(uname -s)/$(uname -m)"
    exit 1
  fi

  # Get latest version if not specified
  VERSION="${1:-latest}"
  if [ "$VERSION" = "latest" ]; then
    VERSION=$(curl -sI "https://github.com/${REPO}/releases/latest" \
      | grep -i '^location:' | sed 's|.*/||' | tr -d '\r\n')
    if [ -z "$VERSION" ]; then
      echo "Error: could not determine latest version"
      exit 1
    fi
  fi

  EXT=""
  if [ "$OS" = "windows" ]; then EXT=".exe"; fi

  FILENAME="${BINARY}-${OS}-${ARCH}${EXT}"
  URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILENAME}"

  echo "Downloading ${BINARY} ${VERSION} for ${OS}/${ARCH}..."
  TMPDIR=$(mktemp -d)
  curl -sL -o "${TMPDIR}/${FILENAME}" "$URL"

  if [ ! -s "${TMPDIR}/${FILENAME}" ]; then
    echo "Error: download failed from ${URL}"
    rm -rf "$TMPDIR"
    exit 1
  fi

  chmod +x "${TMPDIR}/${FILENAME}"

  # Install
  if [ "$OS" = "windows" ]; then
    INSTALL_DIR="${USERPROFILE}/bin"
    mkdir -p "$INSTALL_DIR"
    mv "${TMPDIR}/${FILENAME}" "${INSTALL_DIR}/${BINARY}.exe"
    echo "Installed to ${INSTALL_DIR}/${BINARY}.exe"
    echo "Make sure ${INSTALL_DIR} is in your PATH."
  else
    if [ -w "$INSTALL_DIR" ]; then
      mv "${TMPDIR}/${FILENAME}" "${INSTALL_DIR}/${BINARY}"
    else
      echo "Installing to ${INSTALL_DIR} (requires sudo)..."
      sudo mv "${TMPDIR}/${FILENAME}" "${INSTALL_DIR}/${BINARY}"
    fi
    echo "Installed to ${INSTALL_DIR}/${BINARY}"
  fi

  rm -rf "$TMPDIR"
  echo "Done! Run 'sj' to start."
}

main "$@"
