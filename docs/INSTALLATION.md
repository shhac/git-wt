# Installation Guide

## Requirements
- Zig 0.15.1 or later
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

# Windows (use WSL2)
# Install in WSL2 with the Linux build:
zig build -Doptimize=ReleaseFast -Dtarget=x86_64-linux
```

## Creating a Release

Releases are automated via GitHub Actions. Push a version tag to trigger:

```bash
git tag -a v0.5.1 -m "Release v0.5.1"
git push origin v0.5.1
```

GitHub Actions will build for all platforms and publish the release automatically.

## Version Management

The version is automatically generated from the build system:

```bash
# Check current version
git-wt --version
git-wt -v

# Set custom version during build
zig build -Dversion="1.2.3"
```