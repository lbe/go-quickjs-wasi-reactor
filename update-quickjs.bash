#!/bin/bash
set -euo pipefail

# QuickJS WASI Reactor Update Script
# Downloads the reactor variant from quickjs-ng/quickjs releases

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="quickjs-ng/quickjs"
ASSET_NAME="qjs-wasi-reactor.wasm"
OUTPUT_NAME="qjs-wasi.wasm"

# Use tag from first argument, or find the latest release
TAG="${1:-}"
if [ -z "$TAG" ]; then
    echo "Fetching latest release from $REPO..."
    TAG=$(gh release view --repo "$REPO" --json tagName --jq '.tagName')
fi

if [ -z "$TAG" ] || [ "$TAG" = "null" ]; then
    echo "Error: Could not determine release tag"
    exit 1
fi

echo "Release: $TAG"
echo "Downloading $ASSET_NAME..."

# Download the WASM file
gh release download "$TAG" --repo "$REPO" --pattern "$ASSET_NAME" --output "$SCRIPT_DIR/$OUTPUT_NAME" --clobber

echo "Downloaded $OUTPUT_NAME ($(wc -c < "$SCRIPT_DIR/$OUTPUT_NAME" | tr -d ' ') bytes)"

# Regenerate qjs-wasi.go and qjs-wasi.dat from the downloaded .wasm
echo "Running wasm2go to regenerate Go bindings..."
wasm2go -o "$SCRIPT_DIR/qjs-wasi/qjs-wasi.go" -unsafe -embed "$SCRIPT_DIR/qjs-wasi/qjs-wasi.dat" "$SCRIPT_DIR/$OUTPUT_NAME"
echo "Generated qjs-wasi.go ($(wc -l < "$SCRIPT_DIR/qjs-wasi/qjs-wasi.go" | tr -d ' ') lines)"

# Generate version info Go file
echo "Generating version.go..."
cat > "$SCRIPT_DIR/version.go" << EOF
package quickjswasi

// QuickJS-NG WASI Reactor version information
const (
	// Version is the QuickJS-NG reactor version
	Version = "$TAG"
	// DownloadURL is the URL where this WASM file was downloaded from
	DownloadURL = "https://github.com/$REPO/releases/download/$TAG/$ASSET_NAME"
)
EOF

echo "Generated version.go with version $TAG"
echo ""
echo "Update complete! (wasm2go regeneration done)"
