package claudecode

import (
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
	"github.com/heidihowilson/skillscope/internal/harness"
)

type h struct{}

func (h) ID() string            { return "claude-code" }
func (h) Name() string          { return "Claude Code" }
func (h) Color() lipgloss.Color { return lipgloss.Color("#D97757") } // Anthropic accent orange

func (h) Scopes(ctx harness.Context) []harness.Scope {
	var scopes []harness.Scope
	home := ctx.HomeDir

	for _, p := range []string{
		filepath.Join(home, ".claude", "skills"),
		filepath.Join(home, ".config", "claude", "skills"),
	} {
		scopes = append(scopes, harness.Scope{Harness: "claude-code", Kind: harness.User, Path: p})
	}

	// User-scope plugins (~/.claude/plugins/...).
	scopes = append(scopes, pluginScopesIn(filepath.Join(home, ".claude", "plugins"))...)

	if ctx.RepoRoot != "" {
		scopes = append(scopes,
			harness.Scope{Harness: "claude-code", Kind: harness.Project, Path: filepath.Join(ctx.RepoRoot, ".claude", "skills")},
			harness.Scope{Harness: "claude-code", Kind: harness.ProjectLocal, Path: filepath.Join(ctx.RepoRoot, ".claude", "skills.local")},
		)
		// Project-scope plugins (<repo>/.claude/plugins/...). Per Claude
		// Code docs, `/plugin install --scope project` lands files here
		// and `.claude/settings.json` enables them for collaborators.
		scopes = append(scopes, pluginScopesIn(filepath.Join(ctx.RepoRoot, ".claude", "plugins"))...)
	}
	return scopes
}

// pluginScopesIn expands the known plugin layout patterns under `base`
// into individual Plugin scopes. Covers:
//
//   - flat:                          base/<plugin>/skills
//   - marketplace:                   base/marketplaces/<mkt>/plugins/<plugin>/skills
//   - marketplace (external):        base/marketplaces/<mkt>/external_plugins/<plugin>/skills
//   - cache + marketplace:           base/cache/marketplaces/<mkt>/plugins/<plugin>/skills
//   - cache + marketplace (ext):     base/cache/marketplaces/<mkt>/external_plugins/<plugin>/skills
//
// Claude Code docs mention a `~/.claude/plugins/cache/` location ("plugins
// are copied to a cache") but the empirical install layout is the non-
// cache marketplace path. Scanning both protects against either shape.
func pluginScopesIn(base string) []harness.Scope {
	patterns := []string{
		filepath.Join(base, "*", "skills"),
		filepath.Join(base, "marketplaces", "*", "plugins", "*", "skills"),
		filepath.Join(base, "marketplaces", "*", "external_plugins", "*", "skills"),
		filepath.Join(base, "cache", "marketplaces", "*", "plugins", "*", "skills"),
		filepath.Join(base, "cache", "marketplaces", "*", "external_plugins", "*", "skills"),
	}
	var out []harness.Scope
	seen := map[string]bool{}
	for _, pat := range patterns {
		matches, _ := filepath.Glob(pat)
		for _, m := range matches {
			if seen[m] {
				continue
			}
			seen[m] = true
			out = append(out, harness.Scope{
				Harness:  "claude-code",
				Kind:     harness.Plugin,
				Path:     m,
				ReadOnly: true,
			})
		}
	}
	return out
}

func init() { harness.Register(h{}) }
