#!/usr/bin/env bash
set -e

mkdir -p tools/mtp

echo "==> Building zimap-mtp sidecar binary..."
go build -o tools/mtp/zimap-mtp ./cmd/zimap-mtp

echo "==> Built tools/mtp/zimap-mtp successfully."

# If app bundle exists, copy sidecar helper into it
for app in build/bin/ZiMap.app build/bin/zimap.app; do
    if [ -d "$app/Contents/MacOS" ]; then
        echo "==> Embedding zimap-mtp into $app/Contents/MacOS/..."
        cp tools/mtp/zimap-mtp "$app/Contents/MacOS/zimap-mtp"
    fi
done
