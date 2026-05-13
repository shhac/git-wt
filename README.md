# git-wt

A fast, small CLI for managing git worktrees: create, navigate, list, and clean
up worktrees with an interactive picker, automatic project-config copying, and
a shell wrapper that `cd`s your parent shell.

**Website:** [git-wt.paulie.app](https://git-wt.paulie.app/)

## Features

- **Create worktrees** with one command, branched from any ref
- **Navigate** between worktrees with an arrow-key picker (or by branch name)
- **Remove** one or many with a multi-select picker; safety check on the main
  worktree; bounce-and-delete when you remove the worktree you're in
- **List** all worktrees with the current marker, sorted by last-modified
- **Clean** worktrees whose local branch is gone or whose upstream was deleted
- **Project config copy** controlled by `.git-wt-copy-files` (gitignore-style)
- **Shell integration** via a generated function that talks to the binary
  over a configurable file descriptor — no `eval`-quoting hazards
- **Branch names with slashes** become nested directories (`paul/feature` →
  `paul/feature/`)
- Built with Go, single static binary, zero runtime dependencies (besides git)

## Quick start

### Install

**From source** (requires Go 1.22+):

```bash
git clone https://github.com/shhac/git-wt.git && cd git-wt
go build -o git-wt ./cmd/git-wt
mv git-wt ~/.local/bin/    # or anywhere on $PATH
```

**Pre-built binaries** are published on [Releases](https://github.com/shhac/git-wt/releases) (Linux + macOS).

**Requirements:** Git. Go is only needed to build from source.

### Shell integration

Add to your shell config:

```bash
echo 'eval "$(git-wt alias gwt)"' >> ~/.zshrc   # or ~/.bashrc
```

Now `gwt go`/`gwt new`/`gwt rm` change directory in your parent shell;
everything else passes through unchanged.

### Basic usage

```bash
gwt new feature-branch              # creates <repo>/.gwt/feature-branch and cds in
gwt new paul/auth                   # nested: <repo>/.gwt/paul/auth
gwt go                              # interactive picker
gwt go feature-branch               # direct nav
gwt rm                              # multi-select picker
gwt rm feature-branch --delete-branch
gwt list                            # or `gwt ls`
gwt clean --dry-run                 # show what would be removed
```

The default trees directory is `<repo>/.gwt/` — you'll likely want to add
`.gwt/` to your `.gitignore` (the tool prints a one-line hint when it
isn't). Override per-invocation with `--parent-dir <path>`.

## Commands

| Command | Description |
|---|---|
| `new <branch>` | Create new worktree with branch. Flags: `--from <ref>`, `--parent-dir <path>`, `--no-copy`, `--copy-file-config <path>`. |
| `add [<leaf>] <branch\|remote-ref>` | Create a worktree for an existing local or remote branch. `<remote>/<rest>` (with a matching remote) creates a local branch tracking it; anything else resolves to a local branch. Optional `<leaf>` overrides the directory name. Flags: `--parent-dir <path>`, `--no-copy`, `--copy-file-config <path>`. |
| `rm [branch...]` | Remove worktree(s); interactive multi-select if no args. Flags: `--keep-branch`, `--delete-branch`, `--force`. |
| `go [branch]` | Navigate to a worktree. Suffix match works (`auth` → `paul/auth` if unique). |
| `list` (`ls`) | List worktrees. The first column is the branch, second is the location, third is mtime. |
| `clean` | Remove worktrees whose branch is gone (locally or upstream). Flags: `--dry-run`, `--no-fetch`, `--orphaned-only`, `--upstream-gone-only`. |
| `alias <name>` | Print a shell function wrapper. Flags: `--fd <N>`, `--plain`, `-n`, `--debug`. |

### Global flags

- `-h, --help` — show help
- `-v, --version` — print version
- `--debug` — verbose diagnostics
- `--plain` — no color, minimal formatting (also honors `NO_COLOR`)
- `-n, --non-interactive` — disable prompts (auto-detected when stdin isn't a TTY)
- `--fd <N>` — file descriptor for the wrapper protocol (default `3`, range `3-9`)

### Interactive picker keys

- `↑`/`↓` (or `k`/`j`) — navigate; `Home`/`End` (or `g`/`G`) — jump
- `Enter` — select / confirm
- `Space` — toggle (multi-select); `a` — toggle all (multi-select)
- `Esc`, `Ctrl-C`, `q` — cancel (silent exit, no navigation)

## How it works

By default git-wt creates worktrees inside the main repo at `<repo>/.gwt/`:

```
my-repo/
├── .git/
├── .gwt/                  # default trees dir (add to .gitignore)
│   ├── feature-a/
│   └── paul/
│       └── feature-auth/
└── ... (your code)
```

Override with `--parent-dir`:

```bash
gwt new feature-x --parent-dir ../sibling-trees   # outside the repo
gwt new feature-y --parent-dir ~/scratch          # absolute path
```

### Project-config copy

When you run `gwt new`, the tool copies project-local files from the main
worktree into the new one. The list is controlled by
`<repo>/.git-wt-copy-files` (or `--copy-file-config <path>`). Format is
gitignore-ish: `#` comments, glob patterns, `!`-prefixed exclusions. When
the file is absent, a built-in default copies `.env*`, `.claude/`,
`CLAUDE.local.md`, `.ai-cache/`. See `.git-wt-copy-files.example` in the
repo for the schema.

### Shell-wrapper protocol

`gwt go`/`gwt new` need to change the parent shell's cwd. The wrapper
opens fd `N` (default 3); the binary writes the destination path to fd N;
the wrapper `cd`s the captured value. If the binary is run directly (no
wrapper), it falls back to printing the path on stdout with a copy/paste
hint on stderr — you can use `cd "$(git-wt go feature-x)"` if you prefer.

## Build / test

```bash
go build -o git-wt ./cmd/git-wt    # build
go test -race ./...                # all tests (~95 unit + 17 e2e)
./git-wt --help
```

E2E tests run against fresh repos under `/tmp/` (Go's `t.TempDir()`),
never the project repo. CI runs the suite on Linux + macOS.

## Contributing

1. Fork the repository
2. Create a feature branch (`gwt new my-feature`)
3. Commit your changes
4. Push to the branch
5. Open a Pull Request

Please ensure `go test -race ./...` passes and `go vet ./...` is clean.

## License

MIT — see [LICENSE](LICENSE).
