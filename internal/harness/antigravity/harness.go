package antigravity

import (
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
	"github.com/sethgho/skillscope/internal/harness"
)

type h struct{}

func (h) ID() string            { return "antigravity" }
func (h) Name() string          { return "Antigravity" }
func (h) Color() lipgloss.Color { return lipgloss.Color("#F59E0B") }

func (h) Scopes(ctx harness.Context) []harness.Scope {
	scopes := []harness.Scope{
		{Harness: "antigravity", Kind: harness.User, Path: filepath.Join(ctx.HomeDir, ".antigravity", "skills")},
	}
	if ctx.RepoRoot != "" {
		scopes = append(scopes, harness.Scope{
			Harness: "antigravity", Kind: harness.Project,
			Path: filepath.Join(ctx.RepoRoot, ".antigravity", "skills"),
		})
	}
	return scopes
}

func init() { harness.Register(h{}) }
