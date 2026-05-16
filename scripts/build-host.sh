#!/usr/bin/env bash
# Build the Go host app for macOS as a universal (amd64 + arm64) binary.
# Usage:  ./scripts/build-host.sh
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dist="$root/dist"
mkdir -p "$dist"

cd "$root/host"
version="$(git -C "$root" rev-parse --short HEAD 2>/dev/null || echo dev)"

CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 \
    go build -ldflags "-X main.version=$version -s -w" \
    -o "$dist/hackintosh-amd64" ./cmd/hackintosh

CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 \
    go build -ldflags "-X main.version=$version -s -w" \
    -o "$dist/hackintosh-arm64" ./cmd/hackintosh

lipo -create -output "$dist/hackintosh" \
    "$dist/hackintosh-amd64" "$dist/hackintosh-arm64"

rm "$dist/hackintosh-amd64" "$dist/hackintosh-arm64"
chmod +x "$dist/hackintosh"
codesign --force --sign - "$dist/hackintosh" 2>/dev/null || true
echo "Built $dist/hackintosh (universal, version $version)"
