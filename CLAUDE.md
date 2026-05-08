# git-wt Project

A Go CLI tool for managing git worktrees with enhanced features like automatic
setup, configuration copying, and interactive navigation.

> **Migration in progress.** The project was originally written in Zig; the Go
> rewrite lives on the `migrate-to-go` branch. The Zig codebase is preserved on
> the `zig-cli` branch for reference. See `.ai-cache/plan-go-migration.md` for
> the migration spec.

## Stack
- **Language:** Go (module `github.com/shhac/git-wt`)
- **CLI framework:** [Cobra](https://github.com/spf13/cobra)
- **Prompts/picker:** [huh](https://github.com/charmbracelet/huh)
- **Styling:** [lipgloss](https://github.com/charmbracelet/lipgloss)

## Commands (target parity)

| Command | Description |
|---|---|
| `new <branch>` | Create worktree with config copy |
| `rm [branch...]` | Remove worktree(s); multi-select picker |
| `go [branch]` | Navigate; interactive picker |
| `list` | List worktrees |
| `clean` | Remove worktrees for deleted branches |
| `alias <name>` | Print shell function wrapper |

## Global flags

- `-h, --help` — Cobra
- `-v, --version` — Cobra
- `--debug` — diagnostic output
- `--plain` — no color, minimal formatting (also honours `NO_COLOR` env)
- `-n, --non-interactive` — explicit override; auto-detected when stdin is not a TTY
- `--fd <N>` — file descriptor for shell-wrapper protocol (default 3)

## Wrapper protocol (fd<N>)

The tool is invoked two ways:
- **Wrapper mode** (the shell function from `git-wt alias gwt` opens fd N):
  the binary writes the target path to fd N; the shell function reads it and
  `cd`s the parent shell.
- **Bare mode** (no fd N): the binary writes the path to stdout, with a hint
  on stderr; supports `cd "$(git-wt go branch)"`.

## Layout

```
cmd/git-wt/main.go     # Entry point — Cobra root command
internal/
  git/                 # Git subprocess wrapper
  wt/                  # Worktree path/name handling
  ui/                  # huh + lipgloss helpers
  config/              # TOML config loading
  lock/                # File-based locking
  fd/                  # Wrapper-protocol fd handling
cmd/git-wt/cmd/        # Cobra subcommands (new, rm, go, list, clean, alias)
```

## Build / test

```bash
go build -o git-wt ./cmd/git-wt
go test ./...
./git-wt --help
```

## Reference (legacy / red-green tests)

- `test-interactive/*.exp` — expect-based interactive flows (Zig-era; binary-agnostic)
- `scripts/test-shell-integration.sh` — wrapper-protocol coverage (Zig-era; binary-agnostic)

These are kept as red-green targets during the migration. They invoke `git-wt`
without caring about the implementation language.
