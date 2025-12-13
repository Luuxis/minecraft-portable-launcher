#!/bin/bash

# Build script for Minecraft Launcher

echo "Building Minecraft Launcher..."

# Create bin directory if it doesn't exist
mkdir -p bin

# Build for macOS
echo "Building for macOS..."
GOOS=darwin GOARCH=amd64 go build -o bin/minecraft-launcher-macos

# Add icon to macOS binary (requires icon.icns in resources/)
if [ -f "resources/icon.icns" ]; then
    echo "Adding icon to macOS app..."
    # Create app bundle structure
    mkdir -p "bin/Minecraft Launcher.app/Contents/MacOS"
    mkdir -p "bin/Minecraft Launcher.app/Contents/Resources"
    mv bin/minecraft-launcher-macos "bin/Minecraft Launcher.app/Contents/MacOS/minecraft-launcher"
    cp resources/icon.icns "bin/Minecraft Launcher.app/Contents/Resources/"
    
    # Create Info.plist
    cat > "bin/Minecraft Launcher.app/Contents/Info.plist" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>minecraft-launcher</string>
    <key>CFBundleIconFile</key>
    <string>icon.icns</string>
    <key>CFBundleIdentifier</key>
    <string>com.minecraft.launcher</string>
    <key>CFBundleName</key>
    <string>Minecraft Launcher</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
</dict>
</plist>
EOF
fi

# Build for Windows (without console)
echo "Building for Windows..."

# Compile resource file if icon exists
if [ -f "resources/icon.ico" ]; then
    echo "Compiling Windows resource file..."
    # Use windres if available (for cross-compilation)
    if command -v x86_64-w64-mingw32-windres &> /dev/null; then
        x86_64-w64-mingw32-windres -i icon.rc -o icon.syso -O coff
    elif command -v windres &> /dev/null; then
        windres -i icon.rc -o icon.syso -O coff
    else
        echo "Warning: windres not found, building without icon"
    fi
fi

GOOS=windows GOARCH=amd64 go build -ldflags "-H windowsgui" -o bin/minecraft-launcher.exe

# Clean up resource file
rm -f icon.syso

# Build for Linux
echo "Building for Linux..."
GOOS=linux GOARCH=amd64 go build -o bin/minecraft-launcher-linux

echo "Build complete! Binaries are in the bin/ directory"
