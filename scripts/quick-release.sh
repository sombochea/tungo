#!/usr/bin/env bash

# Quick interactive release helper
# Usage: ./scripts/quick-release.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "╔════════════════════════════════════════╗"
echo "║     TunGo SDK Release Helper          ║"
echo "╚════════════════════════════════════════╝"
echo ""

# Ask which SDK to release
echo "Which SDK would you like to release?"
echo "1) Both (Python & Node.js)"
echo "2) Python only"
echo "3) Node.js only"
read -p "Select (1-3): " sdk_choice

case $sdk_choice in
    1)
        SDK_TYPE="both"
        ;;
    2)
        SDK_TYPE="python"
        ;;
    3)
        SDK_TYPE="node"
        ;;
    *)
        echo "Invalid choice"
        exit 1
        ;;
esac

# Get current versions
PYTHON_CURRENT=$(grep "^version" "$SCRIPT_DIR/../sdk/python/pyproject.toml" | sed 's/version = "\(.*\)"/\1/')
NODE_CURRENT=$(grep '"version"' "$SCRIPT_DIR/../sdk/node/package.json" | head -1 | sed 's/.*"version": "\(.*\)".*/\1/')

echo ""
echo "Current versions:"
echo "  Python: $PYTHON_CURRENT"
echo "  Node.js: $NODE_CURRENT"
echo ""

# Ask for new version
read -p "Enter new version (e.g., 1.0.1): " VERSION

if [[ -z "$VERSION" ]]; then
    echo "Version is required"
    exit 1
fi

# Confirm
echo ""
echo "Ready to release:"
echo "  SDK: $SDK_TYPE"
echo "  Version: $VERSION"
echo ""
read -p "Continue? (y/N) " -n 1 -r
echo

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Release cancelled"
    exit 1
fi

# Run the release script
"$SCRIPT_DIR/release-sdk.sh" "$SDK_TYPE" "$VERSION"
