package opencode

import (
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
	"github.com/sethgho/skillscope/internal/harness"
)

type h struct{}

func (h) ID() string            { return "opencode" }
func (h) Name() string          { return "OpenCode" }
func (h) Color() lipgloss.Color { return lipgloss.Color("#FFCC00") } // OpenCode's brand is grayscale-only; gold stands in for TUI use

func (h) Scopes(ctx harness.Context) []harness.Scope {
	scopes := []harness.Scope{
		{Harness: "opencode", Kind: harness.User, Path: filepath.Join(ctx.HomeDir, ".config", "opencode", "skills")},
	}
	if ctx.RepoRoot != "" {
		scopes = append(scopes, harness.Scope{
			Harness: "opencode", Kind: harness.Project,
			Path: filepath.Join(ctx.RepoRoot, ".opencode", "skills"),
		})
	}
	return scopes
}

func init() { harness.Register(h{}) }
