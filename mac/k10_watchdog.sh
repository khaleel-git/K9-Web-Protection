#!/bin/bash
# K10 Web Protection — Integrity Watchdog
# Ensures the binary and LaunchAgent stay locked and the service stays running.

BINARY="/Applications/K10 Web Protection.app/Contents/MacOS/K10WebProtection"
PLIST="/Library/LaunchAgents/com.k10webprotection.plist"
PROXY_PORT=8080

CONSOLE_USER=$(stat -f "%Su" /dev/console 2>/dev/null || echo "")
CONSOLE_UID=$(id -u "$CONSOLE_USER" 2>/dev/null || echo "")

while true; do
    # Re-lock binary if flag was removed
    if [[ -f "$BINARY" ]] && ! ls -lO "$BINARY" | grep -q "uchg"; then
        chflags uchg "$BINARY" 2>/dev/null
    fi

    # Re-lock plist if flag was removed
    if [[ -f "$PLIST" ]] && ! ls -lO "$PLIST" | grep -q "uchg"; then
        chflags uchg "$PLIST" 2>/dev/null
    fi

    # Re-bootstrap LaunchAgent if it was unloaded
    if [[ -n "$CONSOLE_UID" ]] && ! launchctl list | grep -q "com.k10webprotection"; then
        launchctl bootstrap "gui/$CONSOLE_UID" "$PLIST" 2>/dev/null || \
        launchctl bootstrap system "$PLIST" 2>/dev/null || true
    fi

    # Re-enable system proxy if it was turned off
    if [[ -n "$CONSOLE_USER" ]]; then
        networksetup -listallnetworkservices | grep -v "An asterisk" | grep -v "^$" | while read SERVICE; do
            STATE=$(networksetup -getwebproxy "$SERVICE" 2>/dev/null | grep "^Enabled:" | awk '{print $2}')
            if [[ "$STATE" != "Yes" ]]; then
                networksetup -setwebproxy            "$SERVICE" 127.0.0.1 $PROXY_PORT 2>/dev/null || true
                networksetup -setsecurewebproxy      "$SERVICE" 127.0.0.1 $PROXY_PORT 2>/dev/null || true
                networksetup -setwebproxystate       "$SERVICE" on        2>/dev/null || true
                networksetup -setsecurewebproxystate "$SERVICE" on        2>/dev/null || true
            fi
        done
    fi

    sleep 10
done
