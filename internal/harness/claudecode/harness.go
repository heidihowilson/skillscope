package claudecode

import (
	"os"
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

	// Plugin scopes: expand ~/.claude/plugins/*/skills
	pluginBase := filepath.Join(home, ".claude", "plugins")
	if entries, err := os.ReadDir(pluginBase); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				scopes = append(scopes, harness.Scope{
					Harness:  "claude-code",
					Kind:     harness.Plugin,
					Path:     filepath.Join(pluginBase, e.Name(), "skills"),
					ReadOnly: true,
				})
			}
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
