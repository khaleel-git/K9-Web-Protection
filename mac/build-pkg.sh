#!/bin/bash
# K10 Web Protection — Package Builder
# Produces a distributable macOS .pkg installer.
#
# Usage:
#   bash build-pkg.sh            # build for current arch (arm64 or amd64)
#   bash build-pkg.sh --universal # build universal binary (Intel + Apple Silicon)
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
VERSION="1.0.0"
BUNDLE_ID="com.k10webprotection"
APP_NAME="K10 Web Protection"
STAGE="$SCRIPT_DIR/.pkg-stage"
PKG_COMPONENT="$SCRIPT_DIR/.K10WebProtection-component.pkg"
PKG_OUTPUT="$SCRIPT_DIR/K10WebProtection-$VERSION.pkg"

RED='\033[0;31m'; GRN='\033[0;32m'; YLW='\033[1;33m'; NC='\033[0m'

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  K10 Web Protection — Package Builder v$VERSION"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# ── Check tools ─────────────────────────────────────────────────────────────────
for tool in wails pkgbuild productbuild; do
    if ! command -v "$tool" &>/dev/null; then
        echo -e "${RED}Error: '$tool' not found.${NC}"
        [[ "$tool" == "wails" ]] && echo "  Install: go install github.com/wailsapp/wails/v2/cmd/wails@latest"
        exit 1
    fi
done

# ── 1. Build Wails app ──────────────────────────────────────────────────────────
echo "[1/4] Building app…"
cd "$SCRIPT_DIR/app"
if [[ "$1" == "--universal" ]]; then
    echo "  Building universal binary (Intel + Apple Silicon)…"
    wails build -clean -platform "darwin/universal"
else
    wails build -clean
fi
APP_SRC="$SCRIPT_DIR/app/build/bin/$APP_NAME.app"
if [[ ! -d "$APP_SRC" ]]; then
    echo -e "${RED}Build failed — app not found at $APP_SRC${NC}"; exit 1
fi
echo -e "  ${GRN}Done.${NC}"

# ── 2. Stage files ───────────────────────────────────────────────────────────────
echo "[2/4] Staging files…"
rm -rf "$STAGE"
mkdir -p "$STAGE/Applications"
mkdir -p "$STAGE/Library/LaunchAgents"
mkdir -p "$STAGE/usr/local/bin"
mkdir -p "$STAGE/etc/pf.anchors"

cp -r "$APP_SRC"                                          "$STAGE/Applications/"
cp "$SCRIPT_DIR/com.k10webprotection.plist"               "$STAGE/Library/LaunchAgents/"
cp "$SCRIPT_DIR/com.k10webprotection.watchdog.plist"      "$STAGE/Library/LaunchAgents/"
cp "$SCRIPT_DIR/k10_watchdog.sh"                          "$STAGE/usr/local/bin/"
chmod +x "$STAGE/usr/local/bin/k10_watchdog.sh"

cat > "$STAGE/etc/pf.anchors/k10webprotection" << 'PFRULES'
# K10 Web Protection — block QUIC so browsers cannot bypass the TCP proxy
block drop out quick proto udp to any port 443
PFRULES

echo -e "  ${GRN}Done.${NC}"

# ── 3. Build component .pkg ───────────────────────────────────────────────────────
echo "[3/4] Building component package…"
pkgbuild \
    --root "$STAGE" \
    --scripts "$SCRIPT_DIR/pkg-scripts" \
    --identifier "$BUNDLE_ID" \
    --version "$VERSION" \
    --install-location "/" \
    --ownership recommended \
    "$PKG_COMPONENT"
echo -e "  ${GRN}Done.${NC}"

# ── 4. Wrap with productbuild ────────────────────────────────────────────────────
echo "[4/4] Building distributable installer…"
productbuild \
    --distribution "$SCRIPT_DIR/pkg-resources/distribution.xml" \
    --resources "$SCRIPT_DIR/pkg-resources" \
    --package-path "$(dirname "$PKG_COMPONENT")" \
    "$PKG_OUTPUT"
echo -e "  ${GRN}Done.${NC}"

# Cleanup temp files
rm -f "$PKG_COMPONENT"
rm -rf "$STAGE"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "  ${GRN}Package ready:${NC} K10WebProtection-$VERSION.pkg"
echo ""
echo -e "  ${YLW}Note:${NC} Package is unsigned. Users will see a Gatekeeper"
echo "  warning. To distribute without warnings, sign with:"
echo "    productsign --sign 'Developer ID Installer: ...' \\"
echo "      K10WebProtection-$VERSION.pkg K10WebProtection-$VERSION-signed.pkg"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
