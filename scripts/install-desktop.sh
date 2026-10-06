#!/bin/sh
# Install the Fact Rally binary and desktop entry for an Omarchy / Linux apps menu.
set -e
cd "$(dirname "$0")/.."

mkdir -p ~/.local/bin
mkdir -p ~/.local/share/applications
mkdir -p ~/.local/share/icons/hicolor/512x512/apps

if [ -f dist/factrally-linux ]; then
    cp dist/factrally-linux ~/.local/bin/factrally
else
    go build -o ~/.local/bin/factrally ./cmd/factrally
fi

chmod +x ~/.local/bin/factrally
cp assets/factrally.desktop ~/.local/share/applications/
if [ -f assets/AppIcon.png ]; then
    cp assets/AppIcon.png ~/.local/share/icons/hicolor/512x512/apps/factrally.png
fi

if command -v update-desktop-database >/dev/null 2>&1; then
    update-desktop-database ~/.local/share/applications/
fi

echo "Installed Fact Rally to ~/.local/bin/factrally"
echo "Desktop entry: ~/.local/share/applications/factrally.desktop"
