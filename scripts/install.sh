#!/usr/bin/env bash
# ==============================================================================
# Lingo-Translate (formerly NST) Universal Installer
# Repository: https://github.com/ProjectErotic/Lingo-Translate
# ==============================================================================
set -e

REPO="ProjectErotic/Lingo-Translate"
APP_NAME="lingo"
DESKTOP_APP_NAME="lingo-desktop"
HUMAN_NAME="Lingo Translate"

# Determine Install Prefix
if [ "$(id -u)" -eq 0 ]; then
    INSTALL_DIR="/usr/local/bin"
    APP_DIR="/usr/local/share/applications"
    ICON_DIR="/usr/local/share/icons/hicolor/256x256/apps"
else
    INSTALL_DIR="${HOME}/.local/bin"
    APP_DIR="${HOME}/.local/share/applications"
    ICON_DIR="${HOME}/.local/share/icons/hicolor/256x256/apps"
fi

# Detect OS and Architecture
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "❌ Unsupported architecture: $ARCH"; exit 1 ;;
esac

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." 2>/dev/null && pwd || echo "$SCRIPT_DIR")"

# --- UNINSTALL MODE ---
if [ "$1" = "--uninstall" ] || [ "$1" = "uninstall" ]; then
    echo "🗑️  Uninstalling $HUMAN_NAME..."
    rm -f "${INSTALL_DIR}/lingo" "${INSTALL_DIR}/nst"
    rm -f "${INSTALL_DIR}/lingo-desktop" "${INSTALL_DIR}/nst-desktop"
    rm -f "${APP_DIR}/lingo.desktop" "${APP_DIR}/nst.desktop" "${APP_DIR}/Lingo.desktop" "${APP_DIR}/NST.desktop"
    rm -f "${ICON_DIR}/lingo.png" "${ICON_DIR}/nst.png"
    echo "✅ Successfully uninstalled $HUMAN_NAME."
    exit 0
fi

echo "=========================================================="
echo "🚀 Installing $HUMAN_NAME ($REPO)"
echo "   Target directory: $INSTALL_DIR"
echo "=========================================================="

mkdir -p "$INSTALL_DIR"
mkdir -p "$APP_DIR"
mkdir -p "$ICON_DIR"

SOURCE_BIN=""
SOURCE_DESKTOP_BIN=""

# Check if building or running from local repository
if [ -f "$ROOT_DIR/go.mod" ] && ([ "$1" = "--build" ] || [ ! -f "$ROOT_DIR/bin/lingo" ]); then
    echo "🔨 Building from source..."
    (cd "$ROOT_DIR" && CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/lingo ./cmd/lingo)
fi

# Look for local binaries
if [ -f "$ROOT_DIR/bin/lingo" ]; then
    SOURCE_BIN="$ROOT_DIR/bin/lingo"
elif [ -f "$SCRIPT_DIR/lingo" ]; then
    SOURCE_BIN="$SCRIPT_DIR/lingo"
elif [ -f "./lingo" ]; then
    SOURCE_BIN="./lingo"
fi

if [ -f "$ROOT_DIR/bin/lingo-desktop" ]; then
    SOURCE_DESKTOP_BIN="$ROOT_DIR/bin/lingo-desktop"
elif [ -f "$SCRIPT_DIR/lingo-desktop" ]; then
    SOURCE_DESKTOP_BIN="$SCRIPT_DIR/lingo-desktop"
elif [ -f "./lingo-desktop" ]; then
    SOURCE_DESKTOP_BIN="./lingo-desktop"
fi

# If no local binary, download from GitHub Releases
if [ -z "$SOURCE_BIN" ] || [ ! -f "$SOURCE_BIN" ]; then
    echo "🌐 Downloading latest pre-built release from GitHub ($REPO)..."
    TMP_DOWNLOAD_DIR="$(mktemp -d)"
    TAR_NAME="lingo-linux-${ARCH}.tar.gz"
    
    # Try fetching latest release metadata
    LATEST_JSON=$(curl -sL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null || echo "")
    DOWNLOAD_URL=$(echo "$LATEST_JSON" | grep -o "https://github.com/${REPO}/releases/download/[^\"]*${OS}-${ARCH}[^\"]*\.tar\.gz" | head -n 1 || true)
    
    # Fallback to legacy nst-v* release url pattern if lingo not tagged yet
    if [ -z "$DOWNLOAD_URL" ]; then
        DOWNLOAD_URL=$(echo "$LATEST_JSON" | grep -o "https://github.com/${REPO}/releases/download/[^\"]*${ARCH}[^\"]*\.tar\.gz" | head -n 1 || true)
    fi

    if [ -n "$DOWNLOAD_URL" ]; then
        echo "⬇️  Downloading from: $DOWNLOAD_URL"
        curl -fSL "$DOWNLOAD_URL" -o "${TMP_DOWNLOAD_DIR}/archive.tar.gz"
        tar -xzf "${TMP_DOWNLOAD_DIR}/archive.tar.gz" -C "$TMP_DOWNLOAD_DIR"
        
        # Locate binary in extracted dir
        EXTRACTED_BIN=$(find "$TMP_DOWNLOAD_DIR" -type f \( -name "lingo" -o -name "nst" \) | head -n 1)
        if [ -n "$EXTRACTED_BIN" ]; then
            SOURCE_BIN="$EXTRACTED_BIN"
        fi
        EXTRACTED_DESKTOP=$(find "$TMP_DOWNLOAD_DIR" -type f \( -name "lingo-desktop" -o -name "nst-desktop" \) | head -n 1)
        if [ -n "$EXTRACTED_DESKTOP" ]; then
            SOURCE_DESKTOP_BIN="$EXTRACTED_DESKTOP"
        fi
    fi
fi

if [ -z "$SOURCE_BIN" ] || [ ! -f "$SOURCE_BIN" ]; then
    echo "❌ Error: Could not locate or build 'lingo' binary."
    echo "   Please run 'make build' or ensure Go 1.22+ is installed."
    exit 1
fi

# Install CLI binary
echo "📦 Installing CLI binary to ${INSTALL_DIR}/lingo..."
cp --remove-destination "$SOURCE_BIN" "${INSTALL_DIR}/lingo" 2>/dev/null || (rm -f "${INSTALL_DIR}/lingo" && cp "$SOURCE_BIN" "${INSTALL_DIR}/lingo")
chmod 755 "${INSTALL_DIR}/lingo"

# Create backward-compat symlink 'nst' -> 'lingo'
ln -sf "${INSTALL_DIR}/lingo" "${INSTALL_DIR}/nst"
echo "🔗 Created backward compatibility symlink: ${INSTALL_DIR}/nst -> lingo"

# Install Desktop binary if available
if [ -n "$SOURCE_DESKTOP_BIN" ] && [ -f "$SOURCE_DESKTOP_BIN" ]; then
    echo "📦 Installing Desktop GUI binary to ${INSTALL_DIR}/lingo-desktop..."
    cp --remove-destination "$SOURCE_DESKTOP_BIN" "${INSTALL_DIR}/lingo-desktop" 2>/dev/null || (rm -f "${INSTALL_DIR}/lingo-desktop" && cp "$SOURCE_DESKTOP_BIN" "${INSTALL_DIR}/lingo-desktop")
    chmod 755 "${INSTALL_DIR}/lingo-desktop"
    ln -sf "${INSTALL_DIR}/lingo-desktop" "${INSTALL_DIR}/nst-desktop"
fi

# Install Icon
ICON_SRC=""
if [ -f "$ROOT_DIR/build/appicon.png" ]; then
    ICON_SRC="$ROOT_DIR/build/appicon.png"
elif [ -f "$SCRIPT_DIR/icon.png" ]; then
    ICON_SRC="$SCRIPT_DIR/icon.png"
elif [ -f "./icon.png" ]; then
    ICON_SRC="./icon.png"
fi

if [ -n "$ICON_SRC" ] && [ -f "$ICON_SRC" ]; then
    cp "$ICON_SRC" "${ICON_DIR}/lingo.png"
    ln -sf "${ICON_DIR}/lingo.png" "${ICON_DIR}/nst.png"
    echo "🎨 Installed application icon to ${ICON_DIR}/lingo.png"
fi

# Install Desktop Shortcut
cat <<EOF > "${APP_DIR}/lingo.desktop"
[Desktop Entry]
Version=1.0
Type=Application
Name=Lingo Translate
Comment=Game Translation Suite (Visual Novel & RPG Translation)
Exec=${INSTALL_DIR}/lingo app
Icon=lingo
Terminal=false
Categories=Utility;Development;
StartupNotify=true
EOF
chmod 644 "${APP_DIR}/lingo.desktop"

# Backward compatibility desktop entry
ln -sf "${APP_DIR}/lingo.desktop" "${APP_DIR}/nst.desktop"
echo "🖥️  Installed desktop entry to ${APP_DIR}/lingo.desktop"

# Refresh desktop database if available
if command -v update-desktop-database >/dev/null 2>&1; then
    update-desktop-database "$APP_DIR" >/dev/null 2>&1 || true
fi

# Check PATH
case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *)
        echo ""
        echo "⚠️  Note: $INSTALL_DIR is not currently in your \$PATH."
        echo "   Add the following line to your ~/.bashrc or ~/.zshrc:"
        echo "     export PATH=\"$INSTALL_DIR:\$PATH\""
        ;;
esac

echo ""
echo "🎉 Installation complete!"
echo "   Run 'lingo help' (or legacy 'nst help') to get started."
echo "   Run 'lingo ui' to launch the web translation dashboard."
