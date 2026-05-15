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
		"marketplace-skill": false,
		"external-skill":    false,
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
