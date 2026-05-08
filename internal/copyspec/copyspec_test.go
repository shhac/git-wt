package copyspec

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	in := strings.NewReader(`# comment
.env
.env.*

# blank above and another comment
.claude/
!.env.production
trailing-slash/
`)
	got, err := Parse(in)
	if err != nil {
		t.Fatal(err)
	}
	wantInc := []string{".env", ".env.*", ".claude", "trailing-slash"}
	wantExc := []string{".env.production"}
	if !reflect.DeepEqual(got.Includes, wantInc) {
		t.Errorf("includes = %v, want %v", got.Includes, wantInc)
	}
	if !reflect.DeepEqual(got.Excludes, wantExc) {
		t.Errorf("excludes = %v, want %v", got.Excludes, wantExc)
	}
}

func TestMatch_DefaultsAndExclusions(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{".env", ".env.dev", ".env.production"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}

	spec := &Spec{
		Includes: []string{".env", ".env.*", ".claude"},
		Excludes: []string{".env.production"},
	}
	matches, err := spec.Match(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{".claude", ".env", ".env.dev"}
	sort.Strings(want)
	if !reflect.DeepEqual(matches, want) {
		t.Errorf("matches = %v, want %v", matches, want)
	}
}

func TestMatch_NonglobLiteralPath(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "exact-name"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	spec := &Spec{Includes: []string{"exact-name"}}
	matches, err := spec.Match(root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(matches, []string{"exact-name"}) {
		t.Errorf("matches = %v, want [exact-name]", matches)
	}
}

func TestMatch_NoMatchesQuietlyReturnsEmpty(t *testing.T) {
	root := t.TempDir()
	spec := &Spec{Includes: []string{".env", "*.local"}}
	matches, err := spec.Match(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Errorf("expected empty, got %v", matches)
	}
}

func TestLoad_MissingFileReturnsDefaults(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	spec, err := Load(missing)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(spec.Includes, Defaults().Includes) {
		t.Errorf("expected defaults, got %v", spec.Includes)
	}
}

func TestLoad_FileReplacesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".git-wt-copy-files")
	body := "only-this-file\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	spec, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(spec.Includes, []string{"only-this-file"}) {
		t.Errorf("expected [only-this-file], got %v", spec.Includes)
	}
	if reflect.DeepEqual(spec.Includes, Defaults().Includes) {
		t.Errorf("expected file to override defaults; spec equals defaults")
	}
}
