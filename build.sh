#!/bin/bash
set -e

echo "========================================"
echo "  LLM Proxy Build Script"
echo "========================================"

# Clean old builds
rm -rf ./dist
mkdir -p dist

# Build with Wails
echo "[BUILD] Compiling with Wails..."
wails build -ldflags="-s -w"

# Copy artifacts
echo "[COPY] Copying build output..."
cp build/bin/llm-proxy.exe dist/
if [ -f .env ]; then
  cp .env dist/
fi

echo "========================================"
echo "  Build Complete!"
echo "========================================"
echo "Output: dist/llm-proxy.exe"
