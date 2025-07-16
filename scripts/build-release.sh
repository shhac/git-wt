#!/bin/bash
set -e

# Build script for creating git-wt releases

VERSION=${1:-$(grep 'version' src/main.zig | grep -o '[0-9]\+\.[0-9]\+\.[0-9]\+')}

if [ -z "$VERSION" ]; then
    echo "Usage: $0 [version]"
    echo "Could not determine version from src/main.zig"
    exit 1
fi

echo "Building git-wt v$VERSION for multiple platforms..."

# Create release directory
mkdir -p release

# Build for each platform
platforms=(
    "x86_64-macos"
    "aarch64-macos"
    "x86_64-linux"
    "aarch64-linux"
    "x86_64-windows"
)

for platform in "${platforms[@]}"; do
    echo "Building for $platform..."
    zig build -Doptimize=ReleaseFast -Dtarget=$platform
    
    # Determine output name
    if [[ $platform == *"windows"* ]]; then
        ext=".exe"
    else
        ext=""
    fi
    
    # Copy to release directory with platform suffix
    cp "zig-out/bin/git-wt$ext" "release/git-wt-$platform$ext"
done

echo "Release builds complete! Files in release/:"
ls -la release/

echo ""
echo "To create a GitHub release:"
echo "gh release create v$VERSION --title \"v$VERSION\" --generate-notes release/git-wt-*"