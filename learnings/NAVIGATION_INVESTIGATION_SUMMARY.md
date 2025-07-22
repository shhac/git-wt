# Navigation Investigation Summary

## User Report
"gwt go is not navigating to the expected worktree"

## Investigation Results

After extensive testing, we found that **gwt go IS working correctly**. The confusion arose from testing methods that use pipes.

### What Works ✅

1. **Direct navigation**: `gwt go feature-branch`
2. **Interactive navigation**: `gwt go` (then type number and Enter)
3. **Here-string input**: `gwt go <<< "1"`
4. **Process substitution**: `gwt go < <(echo "1")`

### What Doesn't Work (and Why) ❌

**Piped input**: `echo "1" | gwt go`

This doesn't work because:
- Pipes create subshells in bash/zsh
- The right side of the pipe runs in a subshell
- Any `cd` command executed in a subshell doesn't affect the parent shell
- This is fundamental shell behavior, not a bug in git-wt

### Debug Process

We added debug output to the alias function:
```bash
>&2 echo "[DEBUG gwt] cd_cmd='$cd_cmd'"
>&2 echo "[DEBUG gwt] exit_code=$exit_code"
```

This revealed that:
- The fd3 mechanism correctly captures the cd command
- The alias function correctly executes the cd command
- When using pipes, the cd happens in a subshell

### Key Fix Implemented

We modified the alias to force `--no-tty` in interactive mode:
```bash
if [ $# -eq 0 ]; then
    # Interactive mode - force number selection for reliable fd3
    cd_cmd=$(GWT_USE_FD3=1 eval "$git_wt_bin" go --no-tty $flags 3>&1 1>&2)
else
    # Direct branch navigation
    cd_cmd=$(GWT_USE_FD3=1 eval "$git_wt_bin" go "$@" $flags 3>&1 1>&2)
fi
```

This ensures that interactive mode uses number-based selection, which works more reliably with fd3.

### Testing Recommendations

For automated testing, use:
- Here-strings: `gwt go <<< "1"`
- Direct navigation: `gwt go branch-name`
- NOT pipes: ~~`echo "1" | gwt go`~~

For manual testing:
- Run `gwt go` interactively
- Type the number when prompted
- Press Enter
- Verify with `pwd`

### Conclusion

The navigation feature is working as designed. The perceived issue was due to testing methodology (using pipes) that inherently runs commands in subshells.