#!/bin/bash
# K19 Integrity Enforcer

BINARY="/Applications/K9 Web Protection.app/Contents/MacOS/K9 Web Protection"
PLIST="/Library/LaunchAgents/com.k9webprotection.plist"

while true; do
    # 1. Check if files are missing (Agar delete ho gayi hain toh ye script kuch nahi kar sakegi)
    # Isliye uchg manually lagana zaroori hai.

    # 2. Re-apply locks if they were somehow removed
    if [[ $(ls -lO "$BINARY" | grep -c "uchg") -eq 0 ]]; then
        sudo chflags uchg "$BINARY"
    fi

    if [[ $(ls -lO "$PLIST" | grep -c "uchg") -eq 0 ]]; then
        sudo chflags uchg "$PLIST"
    fi

    # 3. Ensure the Service is LOADED
    if ! launchctl list | grep -q "com.k9webprotection"; then
        launchctl bootstrap gui/$(id -u khaleel) "$PLIST" 2>/dev/null
    fi

    sleep 10
done
