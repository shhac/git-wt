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

### Tab completion (optional)

Install once per shell:

```bash
# zsh — path must be in $fpath
git-wt completion zsh > "${fpath[1]}/_git-wt"

# bash
git-wt completion bash > ~/.local/share/bash-completion/completions/git-wt

# fish
git-wt completion fish > ~/.config/fish/completions/git-wt.fish
```

Then `gwt go <TAB>` lists your worktree branches, `gwt rm <TAB>` lists
removable branches (excluding the main worktree and anything already
typed), and `gwt add <TAB>` lists local + remote refs that don't have
a worktree yet. Candidates carry descriptions matching the columns
from `gwt ls` (location + recency), so shells that support
completion-with-descriptions — zsh, fish, bash-v2, powershell — show
e.g. `feat-x  -- #feat-x  2h  3m` next to each branch. The shell
function from `git-wt alias gwt` binds itself to `git-wt`'s completion
automatically — no second setup step. Opt out with
`git-wt alias gwt --no-completion`.

### Basic usage

```bash
gwt new feature-branch              # creates <repo>/.worktrees/feature-branch and cds in
gwt new paul/auth                   # nested: <repo>/.worktrees/paul/auth
gwt go                              # interactive picker
gwt go feature-branch               # direct nav
gwt rm                              # multi-select picker
gwt rm feature-branch --delete-branch
gwt list                            # or `gwt ls`
gwt clean --dry-run                 # show what would be removed
```

The default trees directory is `<repo>/.worktrees/` — you'll likely want to add
`.worktrees/` to your `.gitignore` (the tool prints a one-line hint when it
isn't). Override per-invocation with `--parent-dir <path>`.

## Commands

| Command | Description |
|---|---|
| `new <branch>` | Create new worktree with branch. Flags: `--from <ref>`, `--parent-dir <path>`, `--no-copy`, `--copy-file-config <path>`. |
| `add [<leaf>] <branch\|remote-ref>` | Create a worktree for an existing local or remote branch. `<remote>/<rest>` (with a matching remote) creates a local branch tracking it; anything else resolves to a local branch. Optional `<leaf>` overrides the directory name. Flags: `--parent-dir <path>`, `--no-copy`, `--copy-file-config <path>`. |
| `eject [<leaf>]` | Move the currently-checked-out branch into a new worktree. Stashes uncommitted changes (tracked + untracked), switches the main tree to `main`/`master` (or `--base`), creates the worktree, and restores the changes inside it. Refuses if HEAD is detached, the current branch is the base, or run from within a non-main worktree. Flags: `--parent-dir <path>`, `--base <branch>`. |
| `rm [branch...]` | Remove worktree(s); interactive multi-select if no args. Deletion is parallel with a live progress line, and copes with read-only/immutable files that make `git worktree remove` die half-way. Also accepts the name of a leftover directory under the trees dir (the debris of an interrupted removal). Flags: `--keep-branch`, `--delete-branch`, `--force`. |
| `go [branch]` | Navigate to a worktree. Suffix match works (`auth` → `paul/auth` if unique). |
| `list` (`ls`) | List worktrees. The first column is the branch, second is the location, third is mtime. |
| `clean` | Remove worktrees whose branch is gone (locally or upstream). Flags: `--dry-run`, `--no-fetch`, `--orphaned-only`, `--upstream-gone-only`. |
| `alias <name>` | Print a shell function wrapper. Flags: `--fd <N>`, `--plain`, `-n`, `--debug`, `--no-completion`. |
| `completion <bash\|zsh\|fish\|powershell>` | Print a shell completion script. See [Tab completion](#tab-completion-optional). |
| `config [<key> [<value>]]` | Show or change persistent settings (stored in `git config wt.*`). See [Configuration](#configuration). |

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

By default git-wt creates worktrees inside the main repo at `<repo>/.worktrees/`:

```
my-repo/
├── .git/
├── .worktrees/                  # default trees dir (add to .gitignore)
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

Or set a persistent default — see [Configuration](#configuration) below.

### Project-config copy

When you run `gwt new`, the tool copies project-local files from the main
worktree into the new one. The list is controlled by
`<repo>/.git-wt-copy-files` (or `--copy-file-config <path>`). Format is
gitignore-ish: `#` comments, glob patterns, `!`-prefixed exclusions. When
the file is absent, a built-in default copies `.env*`, `.claude/`,
`CLAUDE.local.md`, `.ai-cache/`. See `.git-wt-copy-files.example` in the
repo for the schema.

### Configuration

Persistent settings live under the `wt.*` namespace in `git config`, so
they're scope-aware (per-clone via `--local`, per-user via `--global`)
and reachable from vanilla `git config` too. The `git-wt config`
subcommand adds type validation, template-variable resolution, and a
schema-aware `--help`.

```bash
git-wt config                                    # list every key + effective value
git-wt config parentDir                          # show one key (raw + resolved)
git-wt config parentDir '../wt-${repo}'          # set in --local (current repo only)
git-wt config --global parentDir '${repoParent}/${repo}.worktrees'
git-wt config --unset parentDir                  # remove from --local
```

| Key | Type | Default | Notes |
|---|---|---|---|
| `wt.parentDir` | string | `${repoPath}/.worktrees` | Parent directory for new worktrees. Supports `${...}` substitution. |
| `wt.plain` | bool | `false` | Always run with `--plain` (also honors `NO_COLOR`). |
| `wt.fd` | int | `3` | Default fd for the wrapper protocol (`3-9`). |

**Template variables** (path-shaped values only):

- `${repo}` — basename of the main worktree (e.g. `git-wt`)
- `${repoPath}` — absolute path to the main worktree
- `${repoParent}` — directory containing the main worktree
- `${home}` — `$HOME`

Use `$$` for a literal `$`. Unknown variables error at `config` time —
they won't sit silently in your gitconfig waiting to break a future
`new`/`add`/`eject`.

**Precedence** (most-specific wins):

1. explicit CLI flag (`--parent-dir`, `--plain`, `--fd`)
2. `git config --local wt.<key>` (this clone)
3. `git config --global wt.<key>` (user-wide)
4. built-in default

The headline use case for templating is **one global setting that
adapts per repo**. Set
`git config --global wt.parentDir '${repoParent}/${repo}.worktrees'`
once and every repo you work in gets its worktrees in a sibling
directory like `~/code/myrepo.worktrees/`.

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
