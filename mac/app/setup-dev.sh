#!/bin/bash
# K9 Web Protection — Dev environment setup
# Run this once to install Go, Wails, and start the dev server

set -e
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  K9 Web Protection — Dev Setup"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# ── 1. Install Go via Homebrew ─────────────────────────────────────────────
if ! command -v go &>/dev/null; then
  echo "[1/4] Installing Go…"
  brew install go
  export PATH="$PATH:$(go env GOPATH)/bin"
else
  echo "[1/4] Go already installed: $(go version)"
fi

# ── 2. Install Wails CLI ──────────────────────────────────────────────────
if ! command -v wails &>/dev/null; then
  echo "[2/4] Installing Wails CLI…"
  go install github.com/wailsapp/wails/v2/cmd/wails@latest
  export PATH="$PATH:$(go env GOPATH)/bin"
else
  echo "[2/4] Wails already installed: $(wails version)"
fi

# ── 3. Install and build frontend ────────────────────────────────────────
echo "[3/4] Installing and building frontend…"
cd "$(dirname "$0")/frontend"
npm install
npm run build
cd ..

# ── 4. Download Go modules ────────────────────────────────────────────────
echo "[4/4] Downloading Go modules…"
go mod tidy

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Setup complete!"
echo ""
echo "  To run in dev mode:"
echo "    wails dev"
echo ""
echo "  To build the .app:"
echo "    wails build"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
