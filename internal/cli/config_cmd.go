package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/shhac/git-wt/internal/config"
	"github.com/shhac/git-wt/internal/wt"
)

var (
	configGlobal bool
	configUnset  bool
)

// configCmd is the typed front door for git-wt's settings. It's
// roughly equivalent to `git config --local wt.<key>`, but adds:
//
//   - schema-aware --help that enumerates every key, its type/default,
//     and (for templated values) the supported ${...} variables
//   - validation at set time (type check + template var check)
//   - resolved-value display: shows the expanded path next to the raw
//     template, so you can see what gets used without creating a worktree
//
// Read precedence is local-over-global, mirroring `git config`'s normal
// scope walk. Write defaults to --local; pass --global for user-wide.
var configCmd = &cobra.Command{
	Use:   "config [<key> [<value>]]",
	Short: "Show or change git-wt settings (stored in git config wt.*)",
	Long:  configLongHelp(),
	Args:  cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		scope := config.ScopeLocal
		if configGlobal {
			scope = config.ScopeGlobal
		}

		if configUnset {
			if len(args) != 1 {
				return fmt.Errorf("--unset requires exactly one key name")
			}
			return runConfigUnset(ctx, args[0], scope)
		}
		switch len(args) {
		case 0:
			return runConfigList(ctx, os.Stdout)
		case 1:
			return runConfigShow(ctx, os.Stdout, args[0])
		case 2:
			return runConfigSet(ctx, args[0], args[1], scope)
		}
		return nil // unreachable: Args cap is MaximumNArgs(2)
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.Flags().BoolVar(&configGlobal, "global", false, "operate on the global ~/.gitconfig (default: --local)")
	configCmd.Flags().BoolVar(&configUnset, "unset", false, "remove the given key from the chosen scope")
}

// configLongHelp builds the long-form help text from the key registry
// so we can't drift between the schema and what users see.
func configLongHelp() string {
	var b strings.Builder
	b.WriteString("Show or change git-wt settings.\n\n")
	b.WriteString("Values are stored under the wt.* namespace in git config, so they're\n")
	b.WriteString("also reachable via `git config --get wt.<key>`. This command adds\n")
	b.WriteString("type/template validation and resolved-value display.\n\n")
	b.WriteString("Usage:\n")
	b.WriteString("  git-wt config                  list all settings + effective values\n")
	b.WriteString("  git-wt config <key>            show one setting (raw + resolved)\n")
	b.WriteString("  git-wt config <key> <value>    set (default scope: --local)\n")
	b.WriteString("  git-wt config --unset <key>    remove from the chosen scope\n\n")
	b.WriteString("Read precedence: --local beats --global. Built-in defaults apply\n")
	b.WriteString("when nothing is set anywhere.\n\n")
	b.WriteString("Available keys:\n")
	for _, k := range config.All {
		fmt.Fprintf(&b, "  wt.%-12s  %s [%s, default %s]\n", k.Name, k.Doc, k.Type, k.Default)
	}
	b.WriteString("\nTemplate variables (for path-shaped values):\n")
	for _, name := range config.KnownVars() {
		fmt.Fprintf(&b, "  ${%s}\n", name)
	}
	b.WriteString("\nExample:\n")
	b.WriteString("  git-wt config --global parentDir '${repoParent}/${repo}.worktrees'\n")
	return b.String()
}

// runConfigList prints every registered key with its effective value
// and the scope it came from.
func runConfigList(ctx context.Context, w io.Writer) error {
	entries, err := config.List(ctx)
	if err != nil {
		return err
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "KEY\tVALUE\tSOURCE")
	for _, e := range entries {
		val, src := "(unset)", "default"
		if e.IsSet {
			val, src = e.Value, e.Source.String()
		}
		_, _ = fmt.Fprintf(tw, "wt.%s\t%s\t%s\n", e.Key.Name, val, src)
	}
	return tw.Flush()
}

// runConfigShow prints one key in detail. For templated values, also
// shows the post-expansion result against the current repo so users
// can sanity-check their template without running a creation command.
func runConfigShow(ctx context.Context, w io.Writer, name string) error {
	k, err := findKeyOrError(name)
	if err != nil {
		return err
	}
	e, err := config.GetEffective(ctx, k)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(w, "key:      wt.%s\n", k.Name)
	_, _ = fmt.Fprintf(w, "type:     %s\n", k.Type)
	_, _ = fmt.Fprintf(w, "default:  %s\n", k.Default)
	if e.IsSet {
		_, _ = fmt.Fprintf(w, "value:    %s\n", e.Value)
		_, _ = fmt.Fprintf(w, "source:   --%s\n", e.Source)
	} else {
		_, _ = fmt.Fprintln(w, "value:    (unset)")
	}
	if line := resolvedLineFor(ctx, k, e); line != "" {
		_, _ = fmt.Fprintln(w, line)
	}
	return nil
}

// resolvedLineFor returns the `resolved: ...` text for a templated,
// set key, or "" when there's nothing to show (key isn't templated, or
// isn't set, or we're not inside a repo so VarsFor would be meaningless).
//
// Errors from ExpandPath are SURFACED, not swallowed: if someone wrote
// a bad template via raw `git config` (bypassing our set-time
// validation), `git-wt config <key>` should make that visible — not
// drop the line silently and leave the user wondering why nothing
// resolves.
func resolvedLineFor(ctx context.Context, k *config.Key, e config.Entry) string {
	if !k.Templated || !e.IsSet {
		return ""
	}
	repo, err := wt.Inspect(ctx, "")
	if err != nil {
		return "" // not in a repo; "resolved" is meaningless
	}
	resolved, err := config.ExpandPath(e.Value, config.VarsFor(repo.MainRoot))
	if err != nil {
		return fmt.Sprintf("resolved: (error: %v)", err)
	}
	return fmt.Sprintf("resolved: %s", resolved)
}

// runConfigSet validates and stores a value. Local scope requires being
// in a repo; global works anywhere.
func runConfigSet(ctx context.Context, name, value string, scope config.Scope) error {
	k, err := findKeyOrError(name)
	if err != nil {
		return err
	}
	return config.Set(ctx, k, value, scope)
}

// runConfigUnset is idempotent — unsetting a missing key is fine.
func runConfigUnset(ctx context.Context, name string, scope config.Scope) error {
	k, err := findKeyOrError(name)
	if err != nil {
		return err
	}
	return config.Unset(ctx, k, scope)
}

// findKeyOrError looks up a key in the registry, returning a helpful
// error (with the full set of known keys) when the lookup fails. This
// is the one-line preamble every config subcommand handler uses.
func findKeyOrError(name string) (*config.Key, error) {
	if k := config.Find(name); k != nil {
		return k, nil
	}
	names := make([]string, 0, len(config.All))
	for _, k := range config.All {
		names = append(names, "wt."+k.Name)
	}
	return nil, fmt.Errorf("unknown config key %q (known: %s)", name, strings.Join(names, ", "))
}
