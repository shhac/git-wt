# Navigation and fd3 Mechanism

## Overview

The git-wt tool uses file descriptor 3 (fd3) to communicate directory change commands from the subprocess to the parent shell. This allows a CLI tool to effectively change the shell's working directory.

## How fd3 Works

1. Shell alias sets `GWT_USE_FD3=1` and redirects fd3: `3>&1 1>&2`
2. git-wt detects the environment variable and writes `cd /path` to fd3
3. Shell captures the output and executes the cd command

## Common Issues and Solutions

### Pipe Behavior

**Issue**: Commands like `echo "1" | gwt go` don't change directories.

**Reason**: Pipes create subshells in bash/zsh. The right side runs in a subshell, so any `cd` command doesn't affect the parent shell.

**Solution**: Use these alternatives for testing:
- Direct navigation: `gwt go feature-branch`
- Interactive: `gwt go` (then type number)
- Here-strings: `gwt go <<< "1"`
- Process substitution: `gwt go < <(echo "1")`

### The eval Pitfall

**Issue**: Using eval in the shell function breaks environment variable passing.

**Bad**:
```bash
cd_cmd=$(GWT_USE_FD3=1 eval "$git_wt_bin" go $flags 3>&1 1>&2)
```

**Good**:
```bash
cd_cmd=$(GWT_USE_FD3=1 "$git_wt_bin" go $flags 3>&1 1>&2)
```

**Why**: eval adds an unnecessary parsing layer that interferes with environment variable passing to the subprocess.

## Testing fd3

To verify fd3 is working:

```bash
# Direct test - should output "cd /path/to/repo"
echo "1" | GWT_USE_FD3=1 git-wt go --no-tty 3>&1 1>&2

# If you see "📁 Navigating to:" instead, fd3 is not enabled
```

## Key Learnings

1. **Shell behavior matters**: Understanding subshells is crucial for shell integration
2. **eval is rarely needed**: Direct execution is usually correct and simpler
3. **Test the mechanism**: Verify that environment variables are being detected, not just that the feature works
4. **Debug systematically**: Add debug output at each layer to understand where issues occur