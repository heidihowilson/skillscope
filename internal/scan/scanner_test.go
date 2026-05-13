package scan_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/sethgho/skillscope/internal/harness"
	"github.com/sethgho/skillscope/internal/scan"

	_ "github.com/sethgho/skillscope/internal/harness/antigravity"
	_ "github.com/sethgho/skillscope/internal/harness/claudecode"
	_ "github.com/sethgho/skillscope/internal/harness/codex"
	_ "github.com/sethgho/skillscope/internal/harness/cursor"
	_ "github.com/sethgho/skillscope/internal/harness/opencode"
)

func testdataDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine caller path")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata")
}

func TestScanFixtures(t *testing.T) {
	td := testdataDir(t)
	ctx := harness.Context{
		CWD:      filepath.Join(td, "project"),
		RepoRoot: filepath.Join(td, "project"),
		HomeDir:  filepath.Join(td, "home"),
	}

	s := scan.Scanner{Harnesses: harness.All()}
	skills := s.Scan(ctx)

	if len(skills) == 0 {
		t.Fatal("expected skills, got none")
	}

	// Check all five harnesses produce at least one skill.
	harnessSeen := map[string]bool{}
	for _, sk := range skills {
		harnessSeen[sk.Scope.Harness] = true
	}
	for _, h := range []string{"claude-code", "codex", "cursor", "opencode", "antigravity"} {
		if !harnessSeen[h] {
			t.Errorf("expected skills for harness %q, found none", h)
		}
	}
}

func TestParseFrontmatter(t *testing.T) {
	td := testdataDir(t)
	ctx := harness.Context{
		HomeDir: filepath.Join(td, "home"),
	}
	s := scan.Scanner{Harnesses: harness.All()}
	skills := s.Scan(ctx)

	found := map[string]*scan.SkillRecord{}
	for i := range skills {
		found[skills[i].Name] = &skills[i]
	}

	// test-skill should parse cleanly.
	ts, ok := found["test-skill"]
	if !ok {
		t.Fatal("test-skill not found")
	}
	if ts.ParseErr != nil {
		t.Errorf("test-skill parse error: %v", ts.ParseErr)
	}
	if ts.Description == "" {
		t.Error("test-skill description empty")
	}

	// bad-fm should record a parse error but still be returned.
	bf, ok := found["bad-fm"]
	if !ok {
		t.Fatal("bad-fm not found")
	}
	if bf.ParseErr == nil {
		t.Error("bad-fm: expected parse error, got nil")
	}
}

func TestScanUserAndProjectScopes(t *testing.T) {
	td := testdataDir(t)
	ctx := harness.Context{
		CWD:      filepath.Join(td, "project"),
		RepoRoot: filepath.Join(td, "project"),
		HomeDir:  filepath.Join(td, "home"),
	}
	s := scan.Scanner{Harnesses: harness.All()}
	skills := s.Scan(ctx)

	scopesSeen := map[string]bool{}
	for _, sk := range skills {
		scopesSeen[sk.Scope.Kind.String()] = true
	}
	for _, kind := range []string{"user", "project", "local"} {
		if !scopesSeen[kind] {
			t.Errorf("expected skills with scope %q, found none", kind)
		}
	}
}

func TestFindRepoRoot(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := mkdir(gitDir); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(dir, "a", "b", "c")
	if err := mkdirAll(sub); err != nil {
		t.Fatal(err)
	}
	got := scan.FindRepoRoot(sub)
	if got != dir {
		t.Errorf("FindRepoRoot(%q) = %q, want %q", sub, got, dir)
	}
}

func mkdir(p string) error    { return os.Mkdir(p, 0o755) }
func mkdirAll(p string) error { return os.MkdirAll(p, 0o755) }
