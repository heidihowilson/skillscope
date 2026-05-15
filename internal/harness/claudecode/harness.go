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

	// Plugin scopes. Real Claude Code layout is:
	//   ~/.claude/plugins/marketplaces/<marketplace>/{plugins,external_plugins}/<plugin>/skills
	// Older docs/code assumed a flat ~/.claude/plugins/<plugin>/skills, which doesn't
	// match what `/plugin install` actually creates — keep both for resilience.
	pluginBase := filepath.Join(home, ".claude", "plugins")
	addPlugin := func(p string) {
		scopes = append(scopes, harness.Scope{
			Harness:  "claude-code",
			Kind:     harness.Plugin,
			Path:     p,
			ReadOnly: true,
		})
	}
	for _, glob := range []string{
		filepath.Join(pluginBase, "*", "skills"),
		filepath.Join(pluginBase, "marketplaces", "*", "plugins", "*", "skills"),
		filepath.Join(pluginBase, "marketplaces", "*", "external_plugins", "*", "skills"),
	} {
		matches, _ := filepath.Glob(glob)
		for _, m := range matches {
			addPlugin(m)
		}
	}

	if ctx.RepoRoot != "" {
		scopes = append(scopes,
			harness.Scope{Harness: "claude-code", Kind: harness.Project, Path: filepath.Join(ctx.RepoRoot, ".claude", "skills")},
			harness.Scope{Harness: "claude-code", Kind: harness.ProjectLocal, Path: filepath.Join(ctx.RepoRoot, ".claude", "skills.local")},
		)
	}
	return scopes
}

func init() { harness.Register(h{}) }
