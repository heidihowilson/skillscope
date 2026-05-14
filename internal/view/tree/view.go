package tree

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sethgho/skillscope/internal/app"
	"github.com/sethgho/skillscope/internal/harness"
	"github.com/sethgho/skillscope/internal/scan"
	"github.com/sethgho/skillscope/internal/ui"
)

type v struct{}

func (v) ID() string      { return "tree" }
func (v) Name() string    { return "Tree" }
func (v) KeyHint() string { return "2" }

func (v) Init(m *app.Model) tea.Cmd { return nil }

func (vv v) Update(m *app.Model, msg tea.Msg) (app.View, tea.Cmd) {
	return vv, nil
}

// leaf is one cursor-addressable row in the tree (one (scope, name) pair).
type leaf struct {
	kind harness.ScopeKind
	name string
}

// leaves returns all visible (scope, name) pairs in display order:
// scope precedence (user→project→local→plugin), then name alphabetical.
func leaves(m *app.Model) []leaf {
	skills := m.FilteredSkills()
	set := map[leaf]bool{}
	for _, sk := range skills {
		set[leaf{sk.Scope.Kind, sk.Name}] = true
	}
	out := make([]leaf, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	scopeOrder := func(k harness.ScopeKind) int {
		switch k {
		case harness.User:
			return 0
		case harness.Project:
			return 1
		case harness.ProjectLocal:
			return 2
		case harness.Plugin:
			return 3
		}
		return 4
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].kind != out[j].kind {
			return scopeOrder(out[i].kind) < scopeOrder(out[j].kind)
		}
		return out[i].name < out[j].name
	})
	return out
}

// recordsFor returns the SkillRecords matching a (scope, name) pair, in
// registry-harness order.
func recordsFor(m *app.Model, l leaf) []scan.SkillRecord {
	harnessIdx := map[string]int{}
	for i, h := range m.Harnesses {
		harnessIdx[h.ID()] = i
	}
	var recs []scan.SkillRecord
	for _, sk := range m.Skills {
		if sk.Name == l.name && sk.Scope.Kind == l.kind {
			recs = append(recs, sk)
		}
	}
	sort.Slice(recs, func(i, j int) bool {
		return harnessIdx[recs[i].Scope.Harness] < harnessIdx[recs[j].Scope.Harness]
	})
	return recs
}

func (vv v) Navigate(m *app.Model, dir int) {
	ls := leaves(m)
	if len(ls) == 0 {
		m.Cursor = 0
		return
	}
	m.Cursor += dir
	if m.Cursor < 0 {
		m.Cursor = 0
	}
	if m.Cursor >= len(ls) {
		m.Cursor = len(ls) - 1
	}
}

// Selected returns the first (by harness registry order) record at the
// cursor's (scope, name) cell. For multi-harness cells, the matrix view
// is better for picking a specific harness.
func (vv v) Selected(m *app.Model) *scan.SkillRecord {
	ls := leaves(m)
	if len(ls) == 0 || m.Cursor < 0 || m.Cursor >= len(ls) {
		return nil
	}
	recs := recordsFor(m, ls[m.Cursor])
	if len(recs) == 0 {
		return nil
	}
	r := recs[0]
	return &r
}

func scopeHeader(k harness.ScopeKind) string {
	label := ""
	switch k {
	case harness.User:
		label = "user"
	case harness.Project:
		label = "project"
	case harness.ProjectLocal:
		label = "local"
	case harness.Plugin:
		label = "plugin"
	}
	return lipgloss.NewStyle().Bold(true).Foreground(ui.ColorAccent).Render("▼ " + label)
}

// renderDots returns colored dots for the harness records at this leaf.
func renderDots(m *app.Model, recs []scan.SkillRecord) string {
	if len(recs) == 0 {
		return ""
	}
	var glyphs []string
	for _, r := range recs {
		glyph := "●"
		color := ui.HarnessColor(r.Scope.Harness)
		switch {
		case r.ParseErr != nil:
			glyph = "!"
			color = ui.ColorError
		case m.IsShadowed(r):
			glyph = "◐"
		}
		glyphs = append(glyphs, lipgloss.NewStyle().Foreground(color).Render(glyph))
	}
	return strings.Join(glyphs, " ")
}

func (vv v) Render(m *app.Model, width, height int) string {
	ls := leaves(m)
	if len(ls) == 0 {
		return ui.DimStyle.Width(width).Height(height).Render("  no skills found")
	}

	// Clamp.
	if m.Cursor >= len(ls) {
		m.Cursor = len(ls) - 1
	}
	if m.Cursor < 0 {
		m.Cursor = 0
	}

	// Build display rows alongside a parallel `leafAt` slice mapping each
	// row to its leaf index (or -1 for headers/blanks).
	type row struct {
		text    string
		leafIdx int
	}
	var rows []row

	var curKind harness.ScopeKind = -1
	for i, l := range ls {
		if l.kind != curKind {
			if curKind != -1 {
				rows = append(rows, row{text: "", leafIdx: -1})
			}
			curKind = l.kind
			rows = append(rows, row{text: scopeHeader(l.kind), leafIdx: -1})
		}
		recs := recordsFor(m, l)
		dots := renderDots(m, recs)

		// Build the row: name padded, then dots aligned right.
		nameW := width - 16
		if nameW < 12 {
			nameW = 12
		}
		name := l.name
		if len(name) > nameW-1 {
			name = name[:nameW-2] + "…"
		}
		line := fmt.Sprintf("  %-*s %s", nameW, name, dots)
		rows = append(rows, row{text: line, leafIdx: i})
	}

	// Top legend so users know what colors mean.
	legend := buildLegend(m.Harnesses)

	// Find the row index of the selected leaf for scroll math.
	selRow := -1
	for i, r := range rows {
		if r.leafIdx == m.Cursor {
			selRow = i
			break
		}
	}

	// Available scroll area = height minus legend (2 lines including blank).
	maxRows := height - 3 // legend(1) + blank(1) + footer space(1)
	if maxRows < 1 {
		maxRows = 1
	}
	start := 0
	if selRow >= maxRows {
		start = selRow - maxRows + 1
	}
	end := start + maxRows
	if end > len(rows) {
		end = len(rows)
	}

	var rendered []string
	rendered = append(rendered, legend, "")
	for i := start; i < end; i++ {
		r := rows[i]
		text := r.text
		if r.leafIdx == m.Cursor && m.Cursor >= 0 {
			text = ui.SelectedStyle.Render(fmt.Sprintf("%-*s", width, strings.TrimRight(text, "\n")))
		}
		rendered = append(rendered, text)
	}
	return strings.Join(rendered, "\n")
}

func buildLegend(harnesses []harness.Harness) string {
	parts := []string{ui.DimStyle.Render("Harnesses:")}
	for _, h := range harnesses {
		dot := lipgloss.NewStyle().Foreground(h.Color()).Render("●")
		parts = append(parts, dot+" "+h.ID())
	}
	return strings.Join(parts, "  ")
}

func init() { app.RegisterView(v{}) }
