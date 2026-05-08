package wt

import "testing"

func TestValidateBranchName(t *testing.T) {
	good := []string{"main", "feature/auth", "paul/wip", "fix-123", "feat_foo"}
	for _, s := range good {
		if err := ValidateBranchName(s); err != nil {
			t.Errorf("ValidateBranchName(%q) returned error: %v", s, err)
		}
	}
	bad := []string{
		"",
		"-startswithdash",
		"/leading-slash",
		"trailing-slash/",
		"double//slash",
		"has space",
		"has..dotdot",
		"name.lock",
		"control\x01char",
		"caret^",
		"colon:foo",
		"backslash\\",
		"@{badref}",
	}
	for _, s := range bad {
		if err := ValidateBranchName(s); err == nil {
			t.Errorf("ValidateBranchName(%q) should have errored", s)
		}
	}
}

func TestValidateBranchName_LongName(t *testing.T) {
	// 256 char name — should be rejected
	long := make([]byte, 256)
	for i := range long {
		long[i] = 'a'
	}
	if err := ValidateBranchName(string(long)); err == nil {
		t.Errorf("expected long branch name to be rejected")
	}
}
