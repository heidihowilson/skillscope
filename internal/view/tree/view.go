package tree

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sethgho/skillscope/internal/app"
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

// flatLeaves returns the visible skill records in tree-display order:
// harness alphabetical, then scope path alphabetical, then skill name
// alphabetical. This is the order j/k steps through.
func flatLeaves(m *app.Model) []scan.SkillRecord {
	skills := m.FilteredSkills()
	sort.Slice(skills, func(i, j int) bool {
		if skills[i].Scope.Harness != skills[j].Scope.Harness {
			return skills[i].Scope.Harness < skills[j].Scope.Harness
		}
		if skills[i].Scope.Path != skills[j].Scope.Path {
			return skills[i].Scope.Path < skills[j].Scope.Path
		}
		return skills[i].Name < skills[j].Name
	})
	return skills
}

func (vv v) Navigate(m *app.Model, dir int) {
	leaves := flatLeaves(m)
	if len(leaves) == 0 {
		m.Cursor = 0
		return
	}
	m.Cursor += dir
	if m.Cursor < 0 {
		m.Cursor = 0
	}
	if m.Cursor >= len(leaves) {
		m.Cursor = len(leaves) - 1
	}
}

func (vv v) Selected(m *app.Model) *scan.SkillRecord {
	leaves := flatLeaves(m)
	if len(leaves) == 0 || m.Cursor < 0 || m.Cursor >= len(leaves) {
		return nil
	}
	sk := leaves[m.Cursor]
	return &sk
}

func (vv v) Render(m *app.Model, width, height int) string {
	leaves := flatLeaves(m)
	if len(leaves) == 0 {
		return ui.DimStyle.Width(width).Height(height).Render("  no skills found")
	}

	// Clamp cursor to visible leaves.
	if m.Cursor >= len(leaves) {
		m.Cursor = len(leaves) - 1
	}
	if m.Cursor < 0 {
		m.Cursor = 0
	}

	// Build rows in the same order as `leaves`.
	type row struct {
		text        string
		leafIdx     int // -1 if header/scope, else index in `leaves`
		fullWidthOK bool
	}
	var rows []row

	var curHarness, curScopePath string
	for i, sk := range leaves {
		if sk.Scope.Harness != curHarness {
			curHarness = sk.Scope.Harness
			curScopePath = ""
			label := lipgloss.NewStyle().
				Foreground(ui.HarnessColor(sk.Scope.Harness)).
				Bold(true).
				Render("▼ " + sk.Scope.Harness)
			rows = append(rows, row{text: label, leafIdx: -1})
		}
		if sk.Scope.Path != curScopePath {
			curScopePath = sk.Scope.Path
			short := sk.Scope.Path
			if width > 0 && len(short) > width-6 && width > 6 {
				short = "…" + short[len(short)-(width-7):]
			}
			rows = append(rows, row{
				text:    ui.DimStyle.Render("  ├─ " + short),
				leafIdx: -1,
			})
		}

		name := sk.Name
		if sk.ParseErr != nil {
			name += ui.ErrorStyle.Render(" !")
		}
		connector := "│  ├─ "
		// Look ahead to decide whether this is the last skill in this scope.
		isLast := true
		if i+1 < len(leaves) {
			next := leaves[i+1]
			if next.Scope.Harness == sk.Scope.Harness && next.Scope.Path == sk.Scope.Path {
				isLast = false
			}
		}
		if isLast {
			connector = "│  └─ "
		}
		rows = append(rows, row{
			text:        "  " + connector + name,
			leafIdx:     i,
			fullWidthOK: true,
		})
	}

	// Find the row index containing the cursor's leaf so we can scroll.
	selRowIdx := -1
	for i, r := range rows {
		if r.leafIdx == m.Cursor {
			selRowIdx = i
			break
		}
	}

	maxRows := height - 1
	if maxRows < 1 {
		maxRows = 1
	}
	start := 0
	if selRowIdx >= maxRows {
		start = selRowIdx - maxRows + 1
	}
	end := start + maxRows
	if end > len(rows) {
		end = len(rows)
	}

	var out []string
	for i := start; i < end; i++ {
		r := rows[i]
		text := r.text
		if r.leafIdx == m.Cursor {
			text = ui.SelectedStyle.Render(fmt.Sprintf("%-*s", width, stripTrailingNewlines(text)))
		}
		out = append(out, text)
	}
	return strings.Join(out, "\n")
}

func stripTrailingNewlines(s string) string {
	return strings.TrimRight(s, "\n")
}

func init() { app.RegisterView(v{}) }
