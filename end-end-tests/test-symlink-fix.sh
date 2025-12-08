#!/bin/bash
set -e

# Test script for verifying symlink behavior with --extract-strip-components
# Tests the fix for the bug where symlinks were incorrectly having their targets stripped
#
# Usage: ./test-symlink-fix.sh
# Requirements: ripvex binary (builds automatically if missing)

PYENV_URL="https://github.com/pyenv/pyenv/archive/refs/tags/v2.6.15.tar.gz"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RIPVEX_BIN="$SCRIPT_DIR/../build/ripvex"

echo "=== Testing Symlink Strip-Components Fix ==="
echo "Pyenv URL: $PYENV_URL"
echo ""

# Build ripvex if it doesn't exist
if [ ! -f "$RIPVEX_BIN" ]; then
    echo "Building ripvex..."
    make build
fi

# Create a temporary directory
TEST_DIR=$(mktemp -d)
echo "Test directory: $TEST_DIR"
echo ""

pushd "$TEST_DIR" > /dev/null

echo "1. Downloading and extracting pyenv archive with --extract-strip-components=1..."
"$RIPVEX_BIN" -U "$PYENV_URL" -x --extract-strip-components=1

echo ""
echo "2. Checking symlink target..."
SYMLINK_TARGET=$(readlink bin/pyenv)
echo "   bin/pyenv -> $SYMLINK_TARGET"

echo ""
echo "3. Verifying symlink target is correct..."
if [ "$SYMLINK_TARGET" = "../libexec/pyenv" ]; then
    echo "   ✅ PASS: Symlink target is correct (../libexec/pyenv)"
else
    echo "   ❌ FAIL: Expected ../libexec/pyenv, got $SYMLINK_TARGET"
    exit 1
fi

echo ""
echo "4. Testing symlink resolution..."
if head -n 1 bin/pyenv | grep -q "#!/usr/bin/env bash"; then
    echo "   ✅ PASS: Symlink resolves correctly (can read script)"
else
    echo "   ❌ FAIL: Symlink does not resolve correctly"
    exit 1
fi

echo ""
echo "5. Comparing with GNU tar behavior..."
# Extract with GNU tar for comparison
mkdir -p gnu-tar-test
pushd gnu-tar-test > /dev/null
curl -sL "$PYENV_URL" -o pyenv.tar.gz
tar --strip-components=1 -xzf pyenv.tar.gz

GNU_SYMLINK_TARGET=$(readlink bin/pyenv)
popd > /dev/null
echo "   GNU tar: bin/pyenv -> $GNU_SYMLINK_TARGET"
echo "   ripvex:  bin/pyenv -> $SYMLINK_TARGET"

if [ "$SYMLINK_TARGET" = "$GNU_SYMLINK_TARGET" ]; then
    echo "   ✅ PASS: Behavior matches GNU tar"
else
    echo "   ❌ FAIL: Behavior differs from GNU tar"
    exit 1
fi

echo ""
echo "=== All tests passed! Symlink strip-components fix is working correctly ==="

# Cleanup
popd > /dev/null
rm -rf "$TEST_DIR"
