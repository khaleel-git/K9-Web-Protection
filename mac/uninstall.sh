#!/bin/bash
# K10 Web Protection — Uninstaller
set -e

RED='\033[0;31m'; GRN='\033[0;32m'; NC='\033[0m'

if [[ $EUID -ne 0 ]]; then
    echo "Admin privileges required."
    exec sudo env PATH=/usr/bin:/bin:/usr/sbin:/sbin bash "$0" "$@"
fi

REAL_UID=$(id -u "$(logname 2>/dev/null || stat -f "%Su" /dev/console)")

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  K10 Web Protection — Uninstaller"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# ── 1. Stop services ───────────────────────────────────────────────────────────
echo "[1/4] Stopping services…"
launchctl bootout gui/"$REAL_UID" /Library/LaunchAgents/com.k10webprotection.watchdog.plist 2>/dev/null || true
launchctl bootout gui/"$REAL_UID" /Library/LaunchAgents/com.k10webprotection.plist 2>/dev/null || true
echo -e "  ${GRN}Done.${NC}"

# ── 2. Unlock files ────────────────────────────────────────────────────────────
echo "[2/4] Unlocking files…"
chflags -R nouchg "/Applications/K10 Web Protection.app" 2>/dev/null || true
chflags nouchg /Library/LaunchAgents/com.k10webprotection.plist 2>/dev/null || true
chflags nouchg /Library/LaunchAgents/com.k10webprotection.watchdog.plist 2>/dev/null || true
echo -e "  ${GRN}Done.${NC}"

# ── 3. Remove files ────────────────────────────────────────────────────────────
echo "[3/4] Removing files…"
rm -rf "/Applications/K10 Web Protection.app"
rm -f /Library/LaunchAgents/com.k10webprotection.plist
rm -f /Library/LaunchAgents/com.k10webprotection.watchdog.plist
rm -f /usr/local/bin/k10_watchdog.sh
echo -e "  ${GRN}Done.${NC}"

# ── 4. Remove PF rules ────────────────────────────────────────────────────────
echo "[4/5] Removing PF rules…"
pfctl -a k10webprotection -F rules 2>/dev/null || true
sed -i '' '/k10webprotection/d' /etc/pf.conf 2>/dev/null || true
rm -f /etc/pf.anchors/k10webprotection
echo -e "  ${GRN}Done.${NC}"

# ── 5. Clear system proxy ──────────────────────────────────────────────────────
echo "[5/5] Clearing system proxy…"
networksetup -listallnetworkservices 2>/dev/null | grep -v "An asterisk" | grep -v "^$" | while read -r svc; do
    networksetup -setwebproxystate       "$svc" off 2>/dev/null || true
    networksetup -setsecurewebproxystate "$svc" off 2>/dev/null || true
done
echo -e "  ${GRN}Done.${NC}"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "  ${GRN}K10 Web Protection uninstalled.${NC}"
echo "  Config kept at ~/.k10webprotection — remove manually if needed."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
