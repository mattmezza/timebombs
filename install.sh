#!/usr/bin/env bash
#
# timebombs installer — Linux and macOS (amd64 / arm64).
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/mattmezza/timebombs/main/install.sh | bash
#
# Environment overrides:
#   VERSION       Release tag (default: latest). Accepts "v0.4.0" or "0.4.0".
#   INSTALL_DIR   Where to drop the binary (default: /usr/local/bin, falls
#                 back to $HOME/.local/bin if the system dir isn't writable).

set -euo pipefail

REPO="mattmezza/timebombs"
VERSION="${VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-}"

# Normalize "0.4.0" → "v0.4.0" for convenience.
if [ "$VERSION" != "latest" ] && [ "${VERSION#v}" = "$VERSION" ]; then
  VERSION="v$VERSION"
fi

os=$(uname -s)
arch=$(uname -m)

case "$arch" in
  aarch64) arch=arm64 ;;
  amd64)   arch=x86_64 ;;
  x86_64|arm64) ;;
  *) echo "unsupported architecture: $arch" >&2; exit 1 ;;
esac

case "$os" in
  Linux|Darwin) ;;
  *) echo "unsupported OS: $os (Linux or macOS only)" >&2; exit 1 ;;
esac

if [ "$VERSION" = "latest" ]; then
  url="https://github.com/$REPO/releases/latest/download/timebombs_${os}_${arch}.tar.gz"
else
  url="https://github.com/$REPO/releases/download/${VERSION}/timebombs_${os}_${arch}.tar.gz"
fi

# Pick an install dir if the caller didn't.
SUDO=""
if [ -z "$INSTALL_DIR" ]; then
  if [ -w /usr/local/bin ]; then
    INSTALL_DIR=/usr/local/bin
  elif command -v sudo >/dev/null 2>&1 && sudo -n true >/dev/null 2>&1; then
    INSTALL_DIR=/usr/local/bin
    SUDO=sudo
  else
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
  fi
fi

# Make sure the install dir exists. If it doesn't, create it — using sudo
# if needed for system paths.
if [ ! -d "$INSTALL_DIR" ]; then
  if ! mkdir -p "$INSTALL_DIR" 2>/dev/null; then
    if command -v sudo >/dev/null 2>&1; then
      SUDO=sudo
      $SUDO mkdir -p "$INSTALL_DIR"
    else
      echo "cannot create $INSTALL_DIR (no sudo)" >&2
      exit 1
    fi
  fi
fi

# If we still can't write to it, we need sudo.
if [ ! -w "$INSTALL_DIR" ]; then
  if command -v sudo >/dev/null 2>&1; then
    SUDO=sudo
  else
    echo "install dir $INSTALL_DIR is not writable and sudo is unavailable" >&2
    exit 1
  fi
fi

echo "Installing timebombs $VERSION ($os/$arch) → $INSTALL_DIR"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

if ! curl -fsSL "$url" | tar -xz -C "$tmp" timebombs; then
  echo "download/extract failed for $url" >&2
  echo "check that the release exists: https://github.com/$REPO/releases" >&2
  exit 1
fi

$SUDO install -m 0755 "$tmp/timebombs" "$INSTALL_DIR/timebombs"

echo ""
"$INSTALL_DIR/timebombs" version || true

# Heads-up if ~/.local/bin isn't on PATH.
case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    echo ""
    echo "Note: $INSTALL_DIR is not on your PATH. Add it:"
    echo "  echo 'export PATH=\"$INSTALL_DIR:\$PATH\"' >> ~/.bashrc  # or ~/.zshrc"
    ;;
esac
