#!/usr/bin/env bash
set -e

REPO="blackshieldpt/docker-volume-backup"
VERSION=${1:-latest}
BIN="docker-volume-backup"

if [ "$VERSION" = "latest" ]; then
  VERSION=$(curl -s https://api.github.com/repos/$REPO/releases/latest | grep tag_name | cut -d '"' -f4)
fi

OS=$(uname | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case $ARCH in
  x86_64) ARCH=amd64 ;;
  aarch64) ARCH=arm64 ;;
esac

URL="https://github.com/$REPO/releases/download/$VERSION"
BIN_URL="$URL/$BIN-$OS-$ARCH"
CHECKSUM_URL="$URL/$BIN-$OS-$ARCH.sha256"
SIG_URL="$URL/$BIN-$OS-$ARCH.asc"

echo "Downloading $BIN $VERSION for $OS/$ARCH..."
curl -fsSL "$BIN_URL" -o "$BIN"

echo "Verifying binary..."
if command -v gpg >/dev/null 2>&1; then
  curl -fsSL "$SIG_URL" -o "$BIN.asc"
  if gpg --verify "$BIN.asc" "$BIN" 2>/dev/null; then
    echo "GPG signature verified"
    rm -f "$BIN.asc"
  else
    echo "GPG verification failed - falling back to checksum"
    curl -fsSL "$CHECKSUM_URL" -o "$BIN.sha256"
    sha256sum -c "$BIN.sha256" || { echo "Checksum mismatch"; exit 1; }
    rm -f "$BIN.sha256"
  fi
else
  curl -fsSL "$CHECKSUM_URL" -o "$BIN.sha256"
  sha256sum -c "$BIN.sha256" || { echo "Checksum mismatch"; exit 1; }
  rm -f "$BIN.sha256"
fi

chmod +x "$BIN"
sudo mv "$BIN" /usr/local/bin/

echo "Installed $BIN!"
