#!/usr/bin/env bash
set -e

TARGET_ARCH="${1:-${GOARCH:-$(go env GOARCH)}}"
TARGET_OS="${GOOS:-$(go env GOOS)}"

mkdir -p tools/mtp

echo "==> Building zimap-mtp sidecar binary for ${TARGET_OS}/${TARGET_ARCH}..."

if [ "$TARGET_OS" = "darwin" ]; then
    ensure_static_libusb() {
        local arch="$1"
        local c_arch="$arch"
        local host_triplet="arm-apple-darwin"
        if [ "$arch" = "amd64" ]; then
            c_arch="x86_64"
            host_triplet="x86_64-apple-darwin"
        fi
        local prefix="/tmp/zimap-libusb-$arch"
        if [ ! -f "$prefix/lib/libusb-1.0.a" ]; then
            echo "==> Building static libusb for $arch ($c_arch)..." >&2
            local src_dir="/tmp/libusb-1.0.27"
            if [ ! -d "$src_dir" ]; then
                mkdir -p /tmp
                curl -sL https://github.com/libusb/libusb/releases/download/v1.0.27/libusb-1.0.27.tar.bz2 | tar -xj -C /tmp
            fi
            (
                cd "$src_dir"
                [ -f Makefile ] && make distclean >/dev/null 2>&1 || true
                ./configure --host="$host_triplet" CFLAGS="-arch $c_arch" --prefix="$prefix" --disable-udev --disable-shared --enable-static >/dev/null 2>&1
                make -j4 >/dev/null 2>&1
                make install >/dev/null 2>&1
            )
            rm -f "$prefix/lib/"*.dylib
            if [ -f "$prefix/lib/pkgconfig/libusb-1.0.pc" ]; then
                sed -i '' "s|-lusb-1.0|$prefix/lib/libusb-1.0.a -lobjc -framework IOKit -framework CoreFoundation -framework Security|g" "$prefix/lib/pkgconfig/libusb-1.0.pc"
            fi
        fi
        echo "$prefix"
    }

    LIBUSB_PREFIX=$(ensure_static_libusb "$TARGET_ARCH")
    export PKG_CONFIG_PATH="$LIBUSB_PREFIX/lib/pkgconfig:$PKG_CONFIG_PATH"
    export CGO_CFLAGS="-I$LIBUSB_PREFIX/include/libusb-1.0"
    export CGO_LDFLAGS="$LIBUSB_PREFIX/lib/libusb-1.0.a -lobjc -framework IOKit -framework CoreFoundation -framework Security"
    export CGO_ENABLED=1
    export GOOS=darwin
    export GOARCH="$TARGET_ARCH"

    go build -a -ldflags="-s -w" -o tools/mtp/zimap-mtp ./cmd/zimap-mtp
else
    # Non-darwin build (e.g. windows or linux)
    GOOS="$TARGET_OS" GOARCH="$TARGET_ARCH" go build -ldflags="-s -w" -o tools/mtp/zimap-mtp ./cmd/zimap-mtp
fi

chmod +x tools/mtp/zimap-mtp
echo "==> Built tools/mtp/zimap-mtp successfully for ${TARGET_OS}/${TARGET_ARCH}."

# If app bundle exists, copy sidecar helper into it
for app in build/bin/ZiMap.app build/bin/zimap.app; do
    if [ -d "$app/Contents/MacOS" ]; then
        echo "==> Embedding zimap-mtp into $app/Contents/MacOS/..."
        cp tools/mtp/zimap-mtp "$app/Contents/MacOS/zimap-mtp"
        chmod +x "$app/Contents/MacOS/zimap-mtp"
    fi
done
