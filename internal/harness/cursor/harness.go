package cursor

import (
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
	"github.com/heidihowilson/skillscope/internal/harness"
)

type h struct{}

func (h) ID() string            { return "cursor" }
func (h) Name() string          { return "Cursor" }
func (h) Color() lipgloss.Color { return lipgloss.Color("#F54E00") } // Cursor International Orange

func (h) Scopes(ctx harness.Context) []harness.Scope {
	scopes := []harness.Scope{
		{Harness: "cursor", Kind: harness.User, Path: filepath.Join(ctx.HomeDir, ".cursor", "skills")},
	}
	if ctx.RepoRoot != "" {
		scopes = append(scopes, harness.Scope{
			Harness: "cursor", Kind: harness.Project,
			Path: filepath.Join(ctx.RepoRoot, ".cursor", "skills"),
		})
	}
	return scopes
}

func init() { harness.Register(h{}) }
