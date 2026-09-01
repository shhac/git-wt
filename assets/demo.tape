# git-wt demo recording
# Run: vhs assets/demo.tape
#
# Everything runs against a throwaway repo under /tmp created by
# assets/demo-setup.sh. No real repository or remote is touched.
Output assets/demo.gif
Output assets/demo.webm

Set Theme "Catppuccin Mocha"
Set FontFamily "Fira Code"
Set FontSize 15
Set Width 760
Set Height 520
Set Padding 16
Set Framerate 5
Set TypingSpeed 50ms
Set CursorBlink false

# Hidden setup: throwaway repo + the shell alias that makes `cd` work
Hide
Type "bash assets/demo-setup.sh"
Enter
Sleep 2s
Type "cd /tmp/wt-demo/acme-api"
Enter
Sleep 300ms
Type 'eval "$(git-wt alias gwt)"'
Enter
Sleep 500ms
Type "clear"
Enter
Sleep 300ms
Show

# A new worktree, and the shell follows it in
Type "gwt new feature/login"
Sleep 200ms
Enter
Sleep 2.5s

# The branch name nests; project config came along
Type "pwd && ls -a"
Sleep 200ms
Enter
Sleep 3s

# An existing branch, no new branch created
Type "gwt add fix/rate-limit"
Sleep 200ms
Enter
Sleep 2.5s

# Everything on the shelf, newest first
Type "gwt ls"
Sleep 200ms
Enter
Sleep 3s

# Hop straight back by name — the shell moves with you
Type "gwt go main && pwd"
Sleep 200ms
Enter
Sleep 2.5s

# Clean up, taking the branch with it
Type "gwt rm feature/login --delete-branch"
Sleep 200ms
Enter
Sleep 2s
Enter
Sleep 2s

# Gone from the shelf
Type "gwt ls"
Sleep 200ms
Enter
Sleep 3s
