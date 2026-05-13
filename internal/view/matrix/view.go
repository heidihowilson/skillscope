package matrix

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

func (v) ID() string      { return "matrix" }
func (v) Name() string    { return "Matrix" }
func (v) KeyHint() string { return "1" }

func (v) Init(m *app.Model) tea.Cmd { return nil }

func (v) Update(m *app.Model, msg tea.Msg) (app.View, tea.Cmd) {
	return v{}, nil
}

func (v) Render(m *app.Model, width, height int) string {
	skills := m.FilteredSkills()
	if len(skills) == 0 {
		return ui.DimStyle.Width(width).Height(height).Render("  no skills found")
	}

	// Collect all (harness, scope) column headers.
	type col struct {
		harness string
		kind    harness.ScopeKind
		path    string
	}
	colSet := map[string]col{}
	colOrder := []string{}
	for _, sk := range m.Skills {
		key := sk.Scope.Harness + "/" + sk.Scope.Kind.String()
		if _, ok := colSet[key]; !ok {
			colSet[key] = col{sk.Scope.Harness, sk.Scope.Kind, sk.Scope.Path}
			colOrder = append(colOrder, key)
		}
	}
	sort.Strings(colOrder)

	// Build skill name index.
	nameSet := map[string]bool{}
	for _, sk := range skills {
		nameSet[sk.Name] = true
	}
	names := make([]string, 0, len(nameSet))
	for n := range nameSet {
		names = append(names, n)
	}
	sort.Strings(names)

	// Build lookup: name -> colKey -> SkillRecord
	lookup := map[string]map[string]scan.SkillRecord{}
	for _, sk := range m.Skills {
		key := sk.Scope.Harness + "/" + sk.Scope.Kind.String()
		if lookup[sk.Name] == nil {
			lookup[sk.Name] = map[string]scan.SkillRecord{}
		}
		lookup[sk.Name][key] = sk
	}

	// Column widths: harness/scope label truncated.
	const cellW = 10
	nameW := 22

	// Build header row.
	var header strings.Builder
	header.WriteString(fmt.Sprintf("%-*s", nameW, "skill"))
	for _, ck := range colOrder {
		c := colSet[ck]
		label := c.harness + "/" + c.kind.String()
		if len(label) > cellW-1 {
			label = label[:cellW-1]
		}
		col := lipgloss.NewStyle().
			Foreground(ui.HarnessColor(c.harness)).
			Width(cellW).
			Render(label)
		header.WriteString(col)
	}

	hdrStr := ui.BoldStyle.Render(header.String())
	sep := ui.DimStyle.Render(strings.Repeat("─", width))

	// Build rows.
	var rows []string
	rows = append(rows, hdrStr, sep)

	visStart := 0
	maxRows := height - 4
	if maxRows < 1 {
		maxRows = 1
	}
	if m.Cursor >= maxRows {
		visStart = m.Cursor - maxRows + 1
	}

	for i, name := range names {
		if i < visStart {
			continue
		}
		if len(rows)-2 >= maxRows {
			break
		}

		var row strings.Builder
		nameLabel := name
		if len(nameLabel) > nameW-1 {
			nameLabel = nameLabel[:nameW-1] + "…"
		}
		row.WriteString(fmt.Sprintf("%-*s", nameW, nameLabel))

		for _, ck := range colOrder {
			c := colSet[ck]
			sk, ok := lookup[name][ck]
			var glyph string
			var color lipgloss.Color
			if !ok {
				glyph = "·"
				color = ui.ColorAbsent
			} else if m.IsShadowed(sk) {
				glyph = "◐"
				color = ui.ColorShadow
			} else {
				glyph = "●"
				color = ui.HarnessColor(c.harness)
			}
			cell := lipgloss.NewStyle().Foreground(color).Width(cellW).Render(glyph)
			row.WriteString(cell)
		}

		line := row.String()
		if i == m.Cursor {
			line = ui.SelectedStyle.Render(line)
		}
		rows = append(rows, line)
	}

	return strings.Join(rows, "\n")
}

func init() { app.RegisterView(v{}) }
