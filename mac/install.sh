#!/bin/bash
# K10 Web Protection — Installer
# Run once after building: bash install.sh
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
APP_SRC="$SCRIPT_DIR/app/build/bin/K10 Web Protection.app"
APP_DST="/Applications/K10 Web Protection.app"
BINARY="$APP_DST/Contents/MacOS/K10WebProtection"
AGENT_SRC="$SCRIPT_DIR/com.k10webprotection.plist"
AGENT_DST="/Library/LaunchAgents/com.k10webprotection.plist"
WATCHDOG_SRC="$SCRIPT_DIR/k10_watchdog.sh"
WATCHDOG_DST="/usr/local/bin/k10_watchdog.sh"
PF_ANCHOR_DIR="/etc/pf.anchors"
PF_ANCHOR="$PF_ANCHOR_DIR/k10webprotection"
PF_CONF="/etc/pf.conf"

RED='\033[0;31m'; GRN='\033[0;32m'; NC='\033[0m'

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  K10 Web Protection — Installer"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

if [[ "$(uname)" != "Darwin" ]]; then
    echo -e "${RED}This installer is for macOS only.${NC}"; exit 1
fi

if [[ ! -d "$APP_SRC" ]]; then
    echo -e "${RED}App not found at $APP_SRC${NC}"
    echo "  Run 'wails build' inside the app/ directory first."
    exit 1
fi

if [[ $EUID -ne 0 ]]; then
    echo "Admin privileges required."
    exec sudo env PATH=/usr/bin:/bin:/usr/sbin:/sbin bash "$0" "$@"
fi

REAL_USER=$(logname 2>/dev/null || stat -f "%Su" /dev/console)
REAL_UID=$(id -u "$REAL_USER")

# ── 1. Copy app ────────────────────────────────────────────────────────────────
echo "[1/5] Installing app to /Applications…"
chflags -R nouchg "$APP_DST" 2>/dev/null || true
rm -rf "$APP_DST"
cp -r "$APP_SRC" "$APP_DST"
xattr -cr "$APP_DST"
codesign --force --deep --sign - "$APP_DST"
echo -e "  ${GRN}Done.${NC}"

# ── 2. Install watchdog script ─────────────────────────────────────────────────
echo "[2/5] Installing watchdog…"
cp "$WATCHDOG_SRC" "$WATCHDOG_DST"
chmod +x "$WATCHDOG_DST"
echo -e "  ${GRN}Done.${NC}"

# ── 3. Install LaunchAgents ────────────────────────────────────────────────────
echo "[3/5] Installing LaunchAgents…"
cp "$AGENT_SRC" "$AGENT_DST"
launchctl bootout gui/"$REAL_UID" "$AGENT_DST" 2>/dev/null || true
launchctl bootstrap gui/"$REAL_UID" "$AGENT_DST"

WATCHDOG_PLIST="$SCRIPT_DIR/com.k10webprotection.watchdog.plist"
WATCHDOG_AGENT_DST="/Library/LaunchAgents/com.k10webprotection.watchdog.plist"
cp "$WATCHDOG_PLIST" "$WATCHDOG_AGENT_DST"
launchctl bootout gui/"$REAL_UID" "$WATCHDOG_AGENT_DST" 2>/dev/null || true
launchctl bootstrap gui/"$REAL_UID" "$WATCHDOG_AGENT_DST"
echo -e "  ${GRN}Done.${NC}"

# ── 4. Block QUIC (UDP 443) via PF — prevents browsers bypassing TCP proxy ─────
echo "[4/5] Installing PF rules (block QUIC/UDP 443)…"
mkdir -p "$PF_ANCHOR_DIR"
cat > "$PF_ANCHOR" <<'PFRULES'
# K10 Web Protection — block QUIC so browsers cannot bypass the TCP proxy
# 'quick' means: first matching rule wins — fires before any later pass rules
block drop out quick proto udp to any port 443
PFRULES

# Add anchor reference to pf.conf if not already present
if ! grep -q "k10webprotection" "$PF_CONF" 2>/dev/null; then
    echo ""                                              >> "$PF_CONF"
    echo "# K10 Web Protection"                         >> "$PF_CONF"
    echo "anchor \"k10webprotection\""                  >> "$PF_CONF"
    echo "load anchor \"k10webprotection\" from \"$PF_ANCHOR\"" >> "$PF_CONF"
fi

# Enable PF, reload main ruleset (wires our anchor in), then load anchor rules
pfctl -E 2>/dev/null || true
pfctl -f "$PF_CONF" 2>/dev/null || true          # reload entire ruleset so our anchor is evaluated
pfctl -a k10webprotection -f "$PF_ANCHOR" 2>/dev/null || true
echo -e "  ${GRN}Done.${NC}"

# ── 5. Lock files ─────────────────────────────────────────────────────────────
echo "[5/5] Locking files…"
chflags uchg "$BINARY"
chflags uchg "$AGENT_DST"
chflags uchg "$WATCHDOG_AGENT_DST"
echo -e "  ${GRN}Done.${NC}"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "  ${GRN}K10 Web Protection installed successfully!${NC}"
echo "  Open the app and click Enable Protection."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
