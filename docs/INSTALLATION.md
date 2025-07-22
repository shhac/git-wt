# Installation Guide

## Requirements
- Zig 0.14.1 or later
- Git (obviously!)

## Build from Source

```bash
git clone git@github.com:shhac/git-wt.git
cd git-wt

# Debug build (for development)
zig build

# Release build (optimized)
zig build -Doptimize=ReleaseFast

# Run tests
zig build test

# Install to ~/.local/bin
cp zig-out/bin/git-wt ~/.local/bin/

# Or install system-wide
sudo cp zig-out/bin/git-wt /usr/local/bin/
```

## Building for Different Platforms

```bash
# macOS (Intel)
zig build -Doptimize=ReleaseFast -Dtarget=x86_64-macos

# macOS (Apple Silicon) 
zig build -Doptimize=ReleaseFast -Dtarget=aarch64-macos

# Linux x86_64
zig build -Doptimize=ReleaseFast -Dtarget=x86_64-linux

# Linux ARM64
zig build -Doptimize=ReleaseFast -Dtarget=aarch64-linux

# Windows
zig build -Doptimize=ReleaseFast -Dtarget=x86_64-windows
```

## Creating a Release

```bash
# 1. Update version in build.zig if needed
# 2. Build for all platforms
./scripts/build-release.sh  # (create this script with above commands)

# 3. Create GitHub release
gh release create v0.1.0 \
  --title "v0.1.0" \
  --notes "Initial release" \
  zig-out/bin/git-wt-*
```

## Version Management

The version is automatically generated from the build system:

```bash
# Check current version
git-wt --version
git-wt -v

# Set custom version during build
zig build -Dversion="1.2.3"
```