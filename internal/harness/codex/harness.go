package codex

import (
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
	"github.com/sethgho/skillscope/internal/harness"
)

type h struct{}

func (h) ID() string            { return "codex" }
func (h) Name() string          { return "Codex CLI" }
func (h) Color() lipgloss.Color { return lipgloss.Color("#10A37F") }

func (h) Scopes(ctx harness.Context) []harness.Scope {
	scopes := []harness.Scope{
		{Harness: "codex", Kind: harness.User, Path: filepath.Join(ctx.HomeDir, ".codex", "skills")},
	}
	if ctx.RepoRoot != "" {
		scopes = append(scopes, harness.Scope{
			Harness: "codex", Kind: harness.Project,
			Path: filepath.Join(ctx.RepoRoot, ".codex", "skills"),
		})
	}
	return scopes
}

func init() { harness.Register(h{}) }
