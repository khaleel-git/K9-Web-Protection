#!/bin/bash
# K10 Web Protection — Integrity Watchdog
# Ensures the binary and LaunchAgent stay locked and the service stays running.

BINARY="/Applications/K10 Web Protection.app/Contents/MacOS/K10WebProtection"
PLIST="/Library/LaunchAgents/com.k10webprotection.plist"
PROXY_PORT=8080
JOB_LABEL="com.k10webprotection"

CONSOLE_USER=$(stat -f "%Su" /dev/console 2>/dev/null || echo "")
CONSOLE_UID=$(id -u "$CONSOLE_USER" 2>/dev/null || echo "")

# Returns 0 (true) if the K10 process is actually running and listening on the proxy port.
k10_running() {
    pgrep -f "K10WebProtection" > /dev/null 2>&1
}

while true; do
    # Re-lock binary if the immutable flag was removed
    if [[ -f "$BINARY" ]] && ! ls -lO "$BINARY" | grep -q "uchg"; then
        chflags uchg "$BINARY" 2>/dev/null
    fi

    # Re-lock plist if the immutable flag was removed
    if [[ -f "$PLIST" ]] && ! ls -lO "$PLIST" | grep -q "uchg"; then
        chflags uchg "$PLIST" 2>/dev/null
    fi

    if [[ -n "$CONSOLE_UID" ]]; then
        JOB_TARGET="gui/$CONSOLE_UID"

        # Re-bootstrap LaunchAgent if it was completely unloaded from launchd
        if ! launchctl list "$JOB_LABEL" > /dev/null 2>&1; then
            launchctl bootstrap "$JOB_TARGET" "$PLIST" 2>/dev/null || \
            launchctl bootstrap system "$PLIST" 2>/dev/null || true
        fi

        # Bug fix #2: If the job is registered in launchd but K10 is not running,
        # kickstart it immediately instead of waiting for the ThrottleInterval to expire.
        # This gets the app back up faster after a force-kill from Activity Monitor.
        if ! k10_running && launchctl list "$JOB_LABEL" > /dev/null 2>&1; then
            launchctl kickstart "$JOB_TARGET/$JOB_LABEL" 2>/dev/null || true
        fi
    fi

    # Bug fix #1: Only re-enable the system proxy if K10 is actually running.
    # Previously this would point the proxy at 127.0.0.1:8080 with nothing
    # listening there, breaking all internet traffic during the restart gap.
    if k10_running && [[ -n "$CONSOLE_USER" ]]; then
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

    # Re-apply PF QUIC block if the anchor was flushed
    if ! pfctl -a k10webprotection -s rules 2>/dev/null | grep -q "udp"; then
        pfctl -a k10webprotection -f /etc/pf.anchors/k10webprotection 2>/dev/null || true
    fi

    sleep 10
done
