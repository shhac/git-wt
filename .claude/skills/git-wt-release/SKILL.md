---
name: git-wt-release
description: This skill should be used when creating a new release for git-wt, including version bumping, changelog updates, tagging, and publishing through GitHub Actions. Invoked when creating releases, bumping versions, publishing releases, or preparing release tags.
allowed-tools: Read, Edit, Bash, Grep
---

# git-wt Release Process

Automate the complete release workflow for git-wt using GitHub Actions CI/CD.

## Instructions

### 1. Verify Prerequisites

Before starting the release:

```bash
# Run all tests
zig test src/main.zig
zig test src/integration_tests.zig

# Build and verify
zig build -Doptimize=ReleaseFast
./zig-out/bin/git-wt --version
```

All tests must pass before proceeding.

### 2. Update Release Dependencies

Three files must be updated for every release:

**File 1: CHANGELOG.md**
- Add new version section at the top
- Format: `## [X.Y.Z] - YYYY-MM-DD`
- Categories: Added, Fixed, Changed, Developer Notes
- The release workflow extracts this section for GitHub release notes

**File 2: build.zig**
- Update default version string on line 20
- Format: `const version_option = b.option([]const u8, "version", "Version string") orelse "X.Y.Z";`

**File 3: Tests**
- Ensure all 70+ tests pass
- Run both unit and integration tests

### 3. Create Release Commits

Use granular, focused commits:

```bash
# Commit version bump
git add CHANGELOG.md build.zig
git commit -m "chore[release]: bump version to X.Y.Z"
```

### 4. Create and Push Tag

```bash
# Create annotated tag
git tag -a vX.Y.Z -m "Release vX.Y.Z

Brief summary of changes

See CHANGELOG.md for full details."

# Push commit and tag
git push origin main
git push origin vX.Y.Z
```

### 5. Automated Release Process

Once the tag is pushed, GitHub Actions automatically:

1. **Builds for all platforms:**
   - macOS Universal (Intel + ARM combined)
   - macOS x86_64
   - macOS ARM64
   - Linux x86_64
   - Linux ARM64

2. **Creates GitHub Release:**
   - Extracts changelog section for release notes
   - Attaches all platform tarballs
   - Publishes release (not draft)

3. **Timeline:** ~10-15 minutes for full release

### 6. Verify Release

```bash
# Check release on GitHub
gh release view vX.Y.Z

# Download and test a platform binary
curl -L https://github.com/shhac/git-wt/releases/download/vX.Y.Z/git-wt-macos-universal.tar.gz | tar xz
./git-wt --version  # Should show new version
```

## Version Numbering

Follow semantic versioning (semver):

- **Major (X.0.0)**: Breaking changes
- **Minor (0.X.0)**: New features, backward compatible
- **Patch (0.0.X)**: Bug fixes, backward compatible

Examples:
- `v0.4.2` → Bug fixes and small improvements
- `v0.5.0` → New features (config files, new commands)
- `v1.0.0` → Stable API, production ready

## Hotfix Releases

For urgent fixes:

```bash
# 1. Create hotfix branch from main
git checkout -b hotfix/X.Y.Z

# 2. Make fix, update CHANGELOG.md and build.zig
# 3. Test thoroughly
# 4. Merge to main
git checkout main
git merge hotfix/X.Y.Z

# 5. Tag and push
git tag vX.Y.Z
git push origin main vX.Y.Z
```

## Manual Builds (Optional)

For testing or custom builds, use the manual build artifacts workflow:

1. Go to GitHub Actions → "Build Artifacts"
2. Click "Run workflow"
3. Select branch to build from
4. Download artifacts from workflow run (30-day retention)

## Troubleshooting Releases

**Release workflow failed:**
- Check GitHub Actions logs for build errors
- Verify CHANGELOG.md has correct format
- Ensure all tests pass locally first

**Missing platform in release:**
- Check workflow run for specific platform failure
- May need to rerun failed jobs in GitHub Actions

**Changelog extraction failed:**
- Verify CHANGELOG.md format matches: `## [X.Y.Z] - YYYY-MM-DD`
- Ensure there's content between version headers

**Zig API compatibility issues:**
- Check that code compiles with Zig 0.15.1
- Update API calls if Zig version changed
- Verify both local and CI use same Zig version

## Release Checklist

Before completing release:

- [ ] All tests pass (unit + integration)
- [ ] CHANGELOG.md updated with version section
- [ ] build.zig version updated
- [ ] Binary builds and shows correct version
- [ ] Changes committed with "chore[release]" message
- [ ] Tag created with proper format (vX.Y.Z)
- [ ] Tag pushed to GitHub
- [ ] GitHub Actions workflows triggered
- [ ] Release published with binaries
- [ ] Release verified by downloading artifact
