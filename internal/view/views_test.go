package view_test

import (
	"strings"
	"testing"

	"github.com/sethgho/skillscope/internal/app"
	"github.com/sethgho/skillscope/internal/harness"
	"github.com/sethgho/skillscope/internal/scan"

	// Register everything.
	_ "github.com/sethgho/skillscope/internal/harness/antigravity"
	_ "github.com/sethgho/skillscope/internal/harness/claudecode"
	_ "github.com/sethgho/skillscope/internal/harness/codex"
	_ "github.com/sethgho/skillscope/internal/harness/cursor"
	_ "github.com/sethgho/skillscope/internal/harness/opencode"

	_ "github.com/sethgho/skillscope/internal/view/diff"
	_ "github.com/sethgho/skillscope/internal/view/gallery"
	_ "github.com/sethgho/skillscope/internal/view/heatmap"
	_ "github.com/sethgho/skillscope/internal/view/matrix"
	_ "github.com/sethgho/skillscope/internal/view/tree"
	_ "github.com/sethgho/skillscope/internal/view/venn"
)

func newTestModel(width, height int) *app.Model {
	m := app.NewModel(harness.Context{})
	m.Width = width
	m.Height = height
	m.Harnesses = harness.All()
	m.Views = app.AllViews()
	m.Skills = []scan.SkillRecord{
		{Name: "alpha", Description: "first", Scope: harness.Scope{Harness: "claude-code", Kind: harness.User, Path: "/tmp/u"}},
		{Name: "alpha", Description: "first (proj)", Scope: harness.Scope{Harness: "claude-code", Kind: harness.Project, Path: "/tmp/p"}},
		{Name: "beta", Description: "second", Scope: harness.Scope{Harness: "codex", Kind: harness.User, Path: "/tmp/c"}},
		{Name: "gamma", Description: "third", Scope: harness.Scope{Harness: "cursor", Kind: harness.Project, Path: "/tmp/cur"}},
		{Name: "delta", Description: "fourth", Scope: harness.Scope{Harness: "opencode", Kind: harness.User, Path: "/tmp/oc"}},
		{Name: "epsilon", Description: "fifth", Scope: harness.Scope{Harness: "antigravity", Kind: harness.User, Path: "/tmp/ag"}},
	}
	return m
}

// Each view renders without panicking and produces non-empty output
// across the three required terminal sizes.
func TestAllViewsRenderAtAllSizes(t *testing.T) {
	sizes := [][2]int{{80, 24}, {120, 40}, {200, 60}}

	for _, size := range sizes {
		m := newTestModel(size[0], size[1])

		for _, v := range app.AllViews() {
			v := v
			name := v.ID()
			t.Run(name+"_"+sizeName(size), func(t *testing.T) {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("view %q panicked at %dx%d: %v", name, size[0], size[1], r)
					}
				}()
				out := v.Render(m, size[0], size[1])
				if strings.TrimSpace(out) == "" {
					t.Errorf("view %q returned empty output at %dx%d", name, size[0], size[1])
				}
			})
		}
	}
}

// Golden-render-ish snapshot: matrix should mention skill names.
func TestMatrixContainsSkillNames(t *testing.T) {
	m := newTestModel(120, 30)
	for _, v := range app.AllViews() {
		if v.ID() != "matrix" {
			continue
		}
		out := v.Render(m, 120, 30)
		for _, name := range []string{"alpha", "beta", "gamma", "delta", "epsilon"} {
			if !strings.Contains(out, name) {
				t.Errorf("matrix output missing skill %q", name)
			}
		}
	}
}

func TestGalleryListsAllViews(t *testing.T) {
	m := newTestModel(100, 30)
	for _, v := range app.AllViews() {
		if v.ID() != "gallery" {
			continue
		}
		out := v.Render(m, 100, 30)
		for _, id := range []string{"matrix", "tree", "venn", "diff", "heatmap", "gallery"} {
			if !strings.Contains(out, id) {
				t.Errorf("gallery missing view id %q", id)
			}
		}
	}
}

func TestVennShowsCounts(t *testing.T) {
	m := newTestModel(100, 30)
	m.HarnessFilter = map[string]bool{"claude-code": true}
	for _, v := range app.AllViews() {
		if v.ID() != "venn" {
			continue
		}
		out := v.Render(m, 100, 30)
		if !strings.Contains(out, "claude-code") {
			t.Errorf("venn missing harness id: %q", out)
		}
	}
}

func TestHeatmapRendersHarnesses(t *testing.T) {
	m := newTestModel(100, 30)
	for _, v := range app.AllViews() {
		if v.ID() != "heatmap" {
			continue
		}
		out := v.Render(m, 100, 30)
		for _, hid := range []string{"claude-code", "codex", "cursor"} {
			if !strings.Contains(out, hid) {
				t.Errorf("heatmap missing harness %q", hid)
			}
		}
	}
}

func TestDiffOnMultiScope(t *testing.T) {
	m := newTestModel(120, 30)
	// alpha is present in both user and project scopes of claude-code.
	m.Cursor = 0
	for _, v := range app.AllViews() {
		if v.ID() != "diff" {
			continue
		}
		out := v.Render(m, 120, 30)
		if strings.TrimSpace(out) == "" {
			t.Error("diff produced empty output for multi-scope skill")
		}
	}
}

func TestTreeRendersHarnesses(t *testing.T) {
	m := newTestModel(100, 30)
	for _, v := range app.AllViews() {
		if v.ID() != "tree" {
			continue
		}
		out := v.Render(m, 100, 30)
		if !strings.Contains(out, "claude-code") {
			t.Error("tree view missing harness label")
		}
	}
}

func sizeName(s [2]int) string {
	return strings_itoa(s[0]) + "x" + strings_itoa(s[1])
}

// Tiny local itoa to avoid pulling strconv into a test helper.
func strings_itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
