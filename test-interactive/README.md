# Interactive Tests for git-wt (legacy)

> **Status: not currently runnable.** These `expect`-based scripts were
> written against the Zig implementation and target its key bindings,
> output format, and (in places) the now-removed `--show-command` flag.
> They have **not** been ported to the Go rewrite. Kept as historical
> reference only.
>
> The Go E2E suite — which exercises the built binary against fresh
> `/tmp/` repos via `os/exec` — lives in `e2e/` and runs as part of
> `go test -race -count=1 ./...`. New E2E tests should go there.

## Original purpose

These tests verified the Zig version's interactive features:

- Arrow-key navigation in `gwt go` (single-select)
- Multi-select with space-toggles in `gwt rm`
- Confirmation prompts
- ESC cancellation
- The Zig-era `--show-command` flag

They drive the binary through a pseudo-tty using
[`expect`](https://core.tcl-lang.org/expect/) with human-like keystroke
timing.

## Re-using these for the Go binary

If you want to bring them back:

1. Update the binary path (was `./zig-out/bin/git-wt`, now `./git-wt`).
2. Replace `--show-command` references — the Go rewrite always emits the
   wrapper-protocol path on fd N when fd N is open.
3. Replace `GWT_USE_FD3=1` env var — the Go rewrite uses the `--fd N`
   flag; the wrapper from `git-wt alias` baking it in.
4. Re-time the keystroke delays — bubbletea's input loop has different
   timing characteristics from the hand-rolled Zig terminal handling.

Most teams will be better served by writing new tests in `e2e/` using
Go's `os/exec` and `os.Pipe` (see `e2e/main_test.go::runWTFD` for the
fd-3 capture pattern).
