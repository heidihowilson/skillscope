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

func (vv v) Update(m *app.Model, msg tea.Msg) (app.View, tea.Cmd) {
	return vv, nil
}

// Hints implements ActionHinter.
func (vv v) Hints(m *app.Model) []ui.ActionHint {
	return []ui.ActionHint{
		{Key: "↑↓", Label: "row"},
		{Key: "/", Label: "search"},
		{Key: "f", Label: "filter"},
		{Key: "p", Label: "preview"},
		{Key: "c", Label: "copy"},
		{Key: "m", Label: "move"},
	}
}

// scopeKindLabel returns a short label for the matrix header.
func scopeKindLabel(k harness.ScopeKind) string {
	switch k {
	case harness.User:
		return "user"
	case harness.Project:
		return "proj"
	case harness.ProjectLocal:
		return "local"
	case harness.Plugin:
		return "plug"
	}
	return "?"
}

type columnGroup struct {
	harness string
	cols    []harness.ScopeKind
}

func (vv v) Render(m *app.Model, width, height int) string {
	skills := m.FilteredSkills()
	if len(skills) == 0 {
		return ui.DimStyle.Width(width).Height(height).Render("  no skills found")
	}

	// Group columns by harness, in stable registry order.
	groups := buildColumnGroups(m)
	if len(groups) == 0 {
		return ui.DimStyle.Width(width).Height(height).Render("  no harnesses")
	}

	// Count total columns.
	totalCols := 0
	for _, g := range groups {
		totalCols += len(g.cols)
	}

	// Pick a name column width that grows with the longest visible name,
	// capped to a reasonable share of the screen.
	maxName := 6
	for _, sk := range skills {
		if len(sk.Name) > maxName {
			maxName = len(sk.Name)
		}
	}
	nameW := maxName + 2
	if nameW > width/3 {
		nameW = width / 3
	}
	if nameW < 14 {
		nameW = 14
	}

	// Distribute remaining width across columns; minimum 6 chars/col.
	avail := width - nameW - 2
	cellW := avail / totalCols
	if cellW < 6 {
		cellW = 6
	}
	if cellW > 14 {
		cellW = 14
	}

	// Build row 1 of header: harness names, each spanning its scope cells.
	var hdrTop strings.Builder
	hdrTop.WriteString(strings.Repeat(" ", nameW))
	for _, g := range groups {
		span := cellW * len(g.cols)
		label := g.harness
		// Truncate or center.
		if len(label) > span-1 {
			label = label[:span-1]
		}
		pad := span - len(label)
		left := pad / 2
		right := pad - left
		colored := lipgloss.NewStyle().
			Foreground(ui.HarnessColor(g.harness)).
			Bold(true).
			Render(label)
		hdrTop.WriteString(strings.Repeat(" ", left) + colored + strings.Repeat(" ", right))
	}

	// Underline row showing harness boundaries.
	var hdrRule strings.Builder
	hdrRule.WriteString(strings.Repeat(" ", nameW))
	for _, g := range groups {
		span := cellW * len(g.cols)
		rule := strings.Repeat("─", span-1) + " "
		hdrRule.WriteString(lipgloss.NewStyle().Foreground(ui.HarnessColor(g.harness)).Render(rule))
	}

	// Build row 2 of header: scope kind under each cell.
	var hdrBot strings.Builder
	hdrBot.WriteString(ui.BoldStyle.Render(fmt.Sprintf("%-*s", nameW, "skill")))
	for _, g := range groups {
		for _, k := range g.cols {
			label := scopeKindLabel(k)
			cell := lipgloss.NewStyle().
				Foreground(ui.ColorFgDim).
				Width(cellW).
				Render(label)
			hdrBot.WriteString(cell)
		}
	}

	// Skill name index (deduped, sorted).
	nameSet := map[string]bool{}
	for _, sk := range skills {
		nameSet[sk.Name] = true
	}
	names := make([]string, 0, len(nameSet))
	for n := range nameSet {
		names = append(names, n)
	}
	sort.Strings(names)

	// Lookup table: name -> harness -> kind -> record
	lookup := map[string]map[string]map[harness.ScopeKind]scan.SkillRecord{}
	for _, sk := range m.Skills {
		if lookup[sk.Name] == nil {
			lookup[sk.Name] = map[string]map[harness.ScopeKind]scan.SkillRecord{}
		}
		if lookup[sk.Name][sk.Scope.Harness] == nil {
			lookup[sk.Name][sk.Scope.Harness] = map[harness.ScopeKind]scan.SkillRecord{}
		}
		lookup[sk.Name][sk.Scope.Harness][sk.Scope.Kind] = sk
	}

	// Compose body rows.
	maxBodyRows := height - 5
	if maxBodyRows < 1 {
		maxBodyRows = 1
	}
	visStart := 0
	if m.Cursor >= maxBodyRows {
		visStart = m.Cursor - maxBodyRows + 1
	}

	var body []string
	for i, name := range names {
		if i < visStart {
			continue
		}
		if len(body) >= maxBodyRows {
			break
		}

		nameLabel := name
		if len(nameLabel) > nameW-1 {
			nameLabel = nameLabel[:nameW-2] + "…"
		}

		var row strings.Builder
		row.WriteString(fmt.Sprintf("%-*s", nameW, nameLabel))

		for _, g := range groups {
			for _, k := range g.cols {
				sk, ok := lookup[name][g.harness][k]
				var glyph string
				var color lipgloss.Color
				switch {
				case !ok:
					glyph = "·"
					color = ui.ColorAbsent
				case sk.ParseErr != nil:
					glyph = "!"
					color = ui.ColorError
				case m.IsShadowed(sk):
					glyph = "◐"
					color = ui.ColorShadow
				default:
					glyph = "●"
					color = ui.HarnessColor(g.harness)
				}
				cell := lipgloss.NewStyle().Foreground(color).Width(cellW).Render(glyph)
				row.WriteString(cell)
			}
		}

		line := row.String()
		if i == m.Cursor {
			line = ui.SelectedStyle.Render(line)
		}
		body = append(body, line)
	}

	legend := ui.DimStyle.Render("  ● present  ◐ shadowed (higher scope wins)  · absent  ! parse error")

	rows := []string{
		hdrTop.String(),
		hdrRule.String(),
		hdrBot.String(),
		ui.DimStyle.Render(strings.Repeat("─", width)),
	}
	rows = append(rows, body...)
	rows = append(rows, "")
	rows = append(rows, legend)

	return strings.Join(rows, "\n")
}

func buildColumnGroups(m *app.Model) []columnGroup {
	// Discover which (harness, kind) pairs actually exist in the scan.
	have := map[string]map[harness.ScopeKind]bool{}
	for _, sk := range m.Skills {
		if have[sk.Scope.Harness] == nil {
			have[sk.Scope.Harness] = map[harness.ScopeKind]bool{}
		}
		have[sk.Scope.Harness][sk.Scope.Kind] = true
	}

	order := []harness.ScopeKind{harness.User, harness.Project, harness.ProjectLocal, harness.Plugin}

	var groups []columnGroup
	for _, h := range m.Harnesses {
		hid := h.ID()
		if _, ok := have[hid]; !ok {
			continue
		}
		// Apply harness filter.
		if len(m.HarnessFilter) > 0 && !m.HarnessFilter[hid] {
			continue
		}
		g := columnGroup{harness: hid}
		for _, k := range order {
			if have[hid][k] {
				g.cols = append(g.cols, k)
			}
		}
		if len(g.cols) > 0 {
			groups = append(groups, g)
		}
	}
	return groups
}

func init() { app.RegisterView(v{}) }
