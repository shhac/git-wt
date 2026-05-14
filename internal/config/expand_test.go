package config

import (
	"strings"
	"testing"
)

func sampleVars() Vars {
	return Vars{
		Repo:       "myrepo",
		RepoPath:   "/u/p/myrepo",
		RepoParent: "/u/p",
		Home:       "/home/me",
	}
}

func TestExpandPath_AllVars(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"${repo}", "myrepo"},
		{"${repoPath}", "/u/p/myrepo"},
		{"${repoParent}", "/u/p"},
		{"${home}", "/home/me"},
		{"${repoParent}/${repo}.worktrees", "/u/p/myrepo.worktrees"},
		{"${home}/wt/${repo}", "/home/me/wt/myrepo"},
		{"plain/path", "plain/path"},
		{"", ""},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got, err := ExpandPath(c.in, sampleVars())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestExpandPath_UnknownVarErrors(t *testing.T) {
	_, err := ExpandPath("${nope}/${repo}", sampleVars())
	if err == nil {
		t.Fatal("expected error for unknown var")
	}
	if !strings.Contains(err.Error(), "nope") {
		t.Errorf("error should mention the bad var: %v", err)
	}
	if !strings.Contains(err.Error(), "repoPath") {
		t.Errorf("error should list known vars: %v", err)
	}
}

func TestExpandPath_MultipleUnknownVarsDedupedAndSorted(t *testing.T) {
	_, err := ExpandPath("${zeta}/${alpha}/${zeta}/${mu}", sampleVars())
	if err == nil {
		t.Fatal("expected error")
	}
	// Bad names should be sorted: alpha, mu, zeta — and zeta deduped.
	msg := err.Error()
	idxA := strings.Index(msg, "alpha")
	idxM := strings.Index(msg, "mu")
	idxZ := strings.Index(msg, "zeta")
	if idxA < 0 || idxM < 0 || idxZ < 0 {
		t.Fatalf("missing names in error: %v", err)
	}
	if idxA >= idxM || idxM >= idxZ {
		t.Errorf("expected sorted alpha, mu, zeta in: %v", err)
	}
	if strings.Count(msg, "zeta") != 1 {
		t.Errorf("expected deduped names in: %v", err)
	}
}

func TestExpandPath_DollarEscape(t *testing.T) {
	got, err := ExpandPath("$$repo/${repo}", sampleVars())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "$repo/myrepo"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestKnownVarsReturnsCopy(t *testing.T) {
	a := KnownVars()
	a[0] = "mutated"
	b := KnownVars()
	if b[0] == "mutated" {
		t.Error("KnownVars should return a defensive copy")
	}
}
