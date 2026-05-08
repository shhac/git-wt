---
description: Build, release, and publish git-wt to Homebrew
argument-hint: <patch|minor|major>
---

# Release

Perform a full release: version bump, build, GitHub release, and Homebrew tap update.

## Arguments

- `$ARGUMENTS` — version bump type: `patch`, `minor`, or `major`

## Instructions

You are performing a release of the `git-wt` CLI (Go version). Follow these steps exactly.

### Pre-flight

1. Confirm the working tree is clean (`git status --short`). If not, stop and ask the user.
2. Run `make test` and `make lint`. If either fails, stop and fix.
3. Confirm we're on the default branch (typically `main`). If on a feature branch (e.g. `migrate-to-go`), ask the user whether to merge first or release from the current branch.
4. Determine the current version from the latest git tag (`git describe --tags --abbrev=0`) and show the user what bump will happen. If no tag exists or the tag is from the Zig era (`v0.6.x`), the first Go release should be `v0.7.0` — confirm with the user before proceeding.

### Step 1: Version bump, tag, and push

Calculate the new version by bumping the current tag:

```bash
current=$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "0.0.0")
IFS='.' read -r major minor patch <<< "$current"
```

Apply the bump type ($ARGUMENTS):
- `patch`: increment patch
- `minor`: increment minor, reset patch to 0
- `major`: increment major, reset minor and patch to 0

Then tag and push:

```bash
git tag "v${new_version}"
git push origin "$(git rev-parse --abbrev-ref HEAD)" "v${new_version}"
```

### Step 2: Build cross-platform binaries

If `goreleaser` is installed, prefer it (it handles archives, checksums, and the GitHub release in one step):

```bash
goreleaser release --clean
```

Otherwise build manually:

```bash
rm -rf dist/
mkdir -p dist
LDFLAGS="-s -w -X github.com/shhac/git-wt/internal/version.Version=${new_version}"

GOOS=darwin  GOARCH=arm64 go build -ldflags "$LDFLAGS" -o "dist/git-wt-darwin-arm64"      ./cmd/git-wt
GOOS=darwin  GOARCH=amd64 go build -ldflags "$LDFLAGS" -o "dist/git-wt-darwin-amd64"      ./cmd/git-wt
GOOS=linux   GOARCH=amd64 go build -ldflags "$LDFLAGS" -o "dist/git-wt-linux-amd64"       ./cmd/git-wt
GOOS=linux   GOARCH=arm64 go build -ldflags "$LDFLAGS" -o "dist/git-wt-linux-arm64"       ./cmd/git-wt
GOOS=windows GOARCH=amd64 go build -ldflags "$LDFLAGS" -o "dist/git-wt-windows-amd64.exe" ./cmd/git-wt

# Tarball each non-Windows binary as a tar.gz containing a stable `git-wt`
# entry name so Homebrew's `bin.install "git-wt"` works without renames.
cd dist
for triple in darwin-arm64 darwin-amd64 linux-amd64 linux-arm64; do
  mkdir -p "stage-${triple}"
  cp "git-wt-${triple}" "stage-${triple}/git-wt"
  tar -C "stage-${triple}" -czf "git-wt-${triple}.tar.gz" git-wt
  rm -rf "stage-${triple}"
done
zip "git-wt-windows-amd64.zip" "git-wt-windows-amd64.exe"
shasum -a 256 *.tar.gz *.zip > checksums-sha256.txt
cd ..
```

Smoke-test the native binary before proceeding:

```bash
./dist/git-wt-darwin-arm64 --version          # should print: git-wt version <new_version>
./dist/git-wt-darwin-arm64 --help | head -5
./dist/git-wt-darwin-arm64 alias gwt | grep -q '^gwt() {'   # wrapper renders
```

### Step 3: Create GitHub release

If goreleaser handled it (Step 2), skip this step. Otherwise:

```bash
prev_tag=$(git tag --sort=-v:refname | grep -v "^v0\.6" | head -2 | tail -1 || echo "")
if [ -n "$prev_tag" ] && [ "$prev_tag" != "v${new_version}" ]; then
  notes=$(git log --pretty=format:"- %s" "${prev_tag}..v${new_version}" --no-merges | grep -v "^- v[0-9]")
else
  notes="First Go release. See README and AGENTS.md for the rewrite details; the legacy Zig codebase is preserved on the \`zig-cli\` branch."
fi

gh release create "v${new_version}" \
  dist/*.tar.gz dist/*.zip dist/checksums-sha256.txt \
  --title "v${new_version}" \
  --notes "$notes"
```

Verify: `gh release view "v${new_version}"`

### Step 4: Update Homebrew tap

The tap is at `~/projects-personal/homebrew-tap` (sibling repo). The formula
already exists at `Formula/git-wt.rb` — but the file is from the Zig era and
uses `aarch64-macos` / `x86_64-linux` URL conventions. The Go release uses
the standard goreleaser conventions (`darwin-arm64`, `linux-amd64`, etc.).
**Confirm with the user** before rewriting the URL convention; once
agreed, the new formula should be:

```ruby
class GitWt < Formula
  desc "Fast CLI for managing git worktrees with enhanced features"
  homepage "https://github.com/shhac/git-wt"
  version "<new_version>"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/shhac/git-wt/releases/download/v<new_version>/git-wt-darwin-arm64.tar.gz"
      sha256 "<sha256-from-checksums-file>"
    end
    on_intel do
      url "https://github.com/shhac/git-wt/releases/download/v<new_version>/git-wt-darwin-amd64.tar.gz"
      sha256 "<sha256-from-checksums-file>"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/shhac/git-wt/releases/download/v<new_version>/git-wt-linux-arm64.tar.gz"
      sha256 "<sha256-from-checksums-file>"
    end
    on_intel do
      url "https://github.com/shhac/git-wt/releases/download/v<new_version>/git-wt-linux-amd64.tar.gz"
      sha256 "<sha256-from-checksums-file>"
    end
  end

  def install
    bin.install "git-wt"
  end

  def caveats
    <<~EOS
      Enable shell integration so `gwt go` / `gwt new` change directory:

        # zsh (~/.zshrc) or bash (~/.bashrc)
        eval "$(git-wt alias gwt)"

      Then restart your shell or `source` your rc file.

      Without the alias the binary still works in scripts:
        cd "$(git-wt go feature-branch)"
    EOS
  end

  test do
    assert_match "git-wt version <new_version>", shell_output("#{bin}/git-wt --version")
    assert_match "worktree", shell_output("#{bin}/git-wt --help")
  end
end
```

Read the SHAs from `dist/checksums-sha256.txt`, fill them in, then commit:

```bash
cd ../homebrew-tap
git add Formula/git-wt.rb
git commit -m "git-wt ${new_version}"
git push
cd -
```

**IMPORTANT:** Always `cd` back to the git-wt repo after updating the tap.

### Step 5: Report

Show the user:

- New version number
- GitHub release URL (`gh release view "v${new_version}" --json url --jq .url`)
- Homebrew tap commit URL (if updated)
- Install command for new users:    `brew install shhac/tap/git-wt`
- Upgrade command for existing users: `brew upgrade shhac/tap/git-wt`
- Reminder to bump `internal/version/version.go`'s `0.7.0-dev` default if
  the next development cycle has started (optional; the build always uses
  ldflags so this only affects `go run` and unbuilt invocations).
