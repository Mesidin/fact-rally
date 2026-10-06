#!/bin/sh
# Build a macOS .app you can put in /Applications and drag onto the Dock.
# Run this on a Mac, from the repo root:
#   sh scripts/macos-app.sh
set -e
cd "$(dirname "$0")/.."

NAME="Fact Rally"
BIN="factrally"
PKG="./cmd/factrally"
IDENT="dev.erickson.mathskill"
APP="${NAME}.app"

rm -rf "$APP"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"

if [ -f "assets/AppIcon.icns" ]; then
    cp "assets/AppIcon.icns" "$APP/Contents/Resources/AppIcon.icns"
elif [ -f "assets/icon.icns" ]; then
    cp "assets/icon.icns" "$APP/Contents/Resources/AppIcon.icns"
fi

if command -v lipo >/dev/null 2>&1; then
    echo "Building universal binary (Intel + Apple Silicon)..."
    GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$APP/Contents/MacOS/${BIN}-intel" "$PKG"
    GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$APP/Contents/MacOS/${BIN}-arm" "$PKG"
    lipo -create -output "$APP/Contents/MacOS/$BIN" "$APP/Contents/MacOS/${BIN}-intel" "$APP/Contents/MacOS/${BIN}-arm"
    rm "$APP/Contents/MacOS/${BIN}-intel" "$APP/Contents/MacOS/${BIN}-arm"
else
    echo "Building binary for target architecture..."
    GOOS=darwin CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$APP/Contents/MacOS/$BIN" "$PKG"
fi

cat > "$APP/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleExecutable</key>
  <string>$BIN</string>
  <key>CFBundleIconFile</key>
  <string>AppIcon</string>
  <key>CFBundleIdentifier</key>
  <string>$IDENT</string>
  <key>CFBundleName</key>
  <string>$NAME</string>
  <key>CFBundleDisplayName</key>
  <string>$NAME</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>CFBundleVersion</key>
  <string>1.0</string>
  <key>LSMinimumSystemVersion</key>
  <string>11.0</string>
  <key>NSHighResolutionCapable</key>
  <true/>
</dict>
</plist>
EOF

echo "Built $APP"
echo "Copy it to Applications on any Mac, then drag it onto the Dock:"
echo "  cp -R \"$APP\" /Applications/"
echo "  xattr -cr /Applications/\"$APP\"   # Clear Gatekeeper quarantine flag if transferred via AirDrop/USB"
