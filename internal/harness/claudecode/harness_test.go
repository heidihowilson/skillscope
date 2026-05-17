package claudecode_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/heidihowilson/skillscope/internal/harness"
	"github.com/heidihowilson/skillscope/internal/scan"

	_ "github.com/heidihowilson/skillscope/internal/harness/claudecode"
)

func testdataDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine caller path")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "testdata")
}

// TestPluginScopeDiscoversMarketplaceSkills guards the bug Matt hit on first
// install: only "user" skills were showing up because the harness scanned a
// flat ~/.claude/plugins/<plugin>/skills layout that doesn't exist. Real
// installs land under marketplaces/<mkt>/{plugins,external_plugins}/<plugin>/skills.
func TestPluginScopeDiscoversMarketplaceSkills(t *testing.T) {
	td := testdataDir(t)
	ctx := harness.Context{HomeDir: filepath.Join(td, "home")}

	s := scan.Scanner{Harnesses: harness.All()}
	skills := s.Scan(ctx)

	want := map[string]bool{
		"marketplace-skill":   false, // user-scope marketplace plugin
		"external-skill":      false, // user-scope marketplace external_plugin
		"cached-plugin-skill": false, // user-scope under ~/.claude/plugins/cache/...
	}
	for _, sk := range skills {
		if sk.Scope.Harness != "claude-code" || sk.Scope.Kind != harness.Plugin {
			continue
		}
		if _, ok := want[sk.Name]; ok {
			want[sk.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("expected plugin-scope skill %q, not found", name)
		}
	}
}

// TestPluginScopeDiscoversProjectInstalledSkills covers `/plugin install
// --scope project`. Those plugins land in the repo under
// <repo>/.claude/plugins/marketplaces/<mkt>/{plugins,external_plugins}/<plugin>/skills
// and are shared with teammates via `.claude/settings.json`. The first
// fix only updated the user-scope path; project scope was a separate bug.
func TestPluginScopeDiscoversProjectInstalledSkills(t *testing.T) {
	td := testdataDir(t)
	ctx := harness.Context{
		HomeDir:  filepath.Join(td, "home"),
		RepoRoot: filepath.Join(td, "project"),
	}

	s := scan.Scanner{Harnesses: harness.All()}
	skills := s.Scan(ctx)

	want := map[string]bool{
		"project-plugin-skill":   false,
		"project-external-skill": false,
	}
	for _, sk := range skills {
		if sk.Scope.Harness != "claude-code" || sk.Scope.Kind != harness.Plugin {
			continue
		}
		if _, ok := want[sk.Name]; ok {
			want[sk.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("expected project-scope plugin skill %q, not found", name)
		}
	}
}
