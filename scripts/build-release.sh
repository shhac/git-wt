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
)

for platform in "${platforms[@]}"; do
    echo "Building for $platform..."
    zig build -Doptimize=ReleaseFast -Dtarget=$platform
    
    # Copy to release directory with platform suffix
    cp "zig-out/bin/git-wt" "release/git-wt-$platform"
    
    # Create tarball for each platform
    cd release
    tar -czf "git-wt-$platform.tar.gz" "git-wt-$platform"
    rm "git-wt-$platform"  # Remove the binary, keep only tarball
    cd ..
done

# Create a universal macOS binary if both architectures are available
if [ -f "release/git-wt-x86_64-macos.tar.gz" ] && [ -f "release/git-wt-aarch64-macos.tar.gz" ]; then
    echo "Creating universal macOS binary..."
    cd release
    tar -xzf git-wt-x86_64-macos.tar.gz
    tar -xzf git-wt-aarch64-macos.tar.gz
    lipo -create -output git-wt-macos-universal git-wt-x86_64-macos git-wt-aarch64-macos
    tar -czf git-wt-macos-universal.tar.gz git-wt-macos-universal
    rm git-wt-x86_64-macos git-wt-aarch64-macos git-wt-macos-universal
    cd ..
fi

echo "Release builds complete! Files in release/:"
ls -lh release/

echo ""
echo "To create a GitHub release:"
echo "gh release create v$VERSION --title \"v$VERSION\" --generate-notes release/*.tar.gz"