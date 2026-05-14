package matrix_test

import (
	"testing"

	"github.com/heidihowilson/skillscope/internal/app"
	"github.com/heidihowilson/skillscope/internal/harness"
	"github.com/heidihowilson/skillscope/internal/scan"
	_ "github.com/heidihowilson/skillscope/internal/view/matrix"
)

func makeTestModel() *app.Model {
	m := app.NewModel(harness.Context{})
	m.Width = 80
	m.Height = 24
	m.Skills = []scan.SkillRecord{
		{Name: "skill-a", Description: "desc a", Scope: harness.Scope{Harness: "claude-code", Kind: harness.User}},
		{Name: "skill-b", Description: "desc b", Scope: harness.Scope{Harness: "codex", Kind: harness.Project}},
	}
	m.Views = app.AllViews()
	return m
}

func TestMatrixRenderNonempty(t *testing.T) {
	m := makeTestModel()
	views := app.AllViews()
	var mv app.View
	for _, v := range views {
		if v.ID() == "matrix" {
			mv = v
			break
		}
	}
	if mv == nil {
		t.Fatal("matrix view not registered")
	}
	out := mv.Render(m, 80, 24)
	if out == "" {
		t.Error("matrix render returned empty string")
	}
	if len(out) < 10 {
		t.Errorf("matrix render suspiciously short: %q", out)
	}
}
