package matrix

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/heidihowilson/skillscope/internal/app"
	"github.com/heidihowilson/skillscope/internal/harness"
	"github.com/heidihowilson/skillscope/internal/scan"
	"github.com/heidihowilson/skillscope/internal/ui"
)

type v struct{}

func (v) ID() string      { return "matrix" }
func (v) Name() string    { return "Matrix" }
func (v) KeyHint() string { return "1" }

func (v) Init(m *app.Model) tea.Cmd { return nil }

func (vv v) Update(m *app.Model, msg tea.Msg) (app.View, tea.Cmd) {
	return vv, nil
}

// uniqueNames returns the alphabetically sorted unique skill names after
// applying the current filters. This is the matrix view's row order.
func uniqueNames(m *app.Model) []string {
	set := map[string]bool{}
	for _, sk := range m.FilteredSkills() {
		set[sk.Name] = true
	}
	out := make([]string, 0, len(set))
	for n := range set {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// Navigate moves the cursor through unique skill names.
func (vv v) Navigate(m *app.Model, dir int) {
	names := uniqueNames(m)
	if len(names) == 0 {
		m.Cursor = 0
		return
	}
	m.Cursor += dir
	if m.Cursor < 0 {
		m.Cursor = 0
	}
	if m.Cursor >= len(names) {
		m.Cursor = len(names) - 1
	}
}

// Selected returns the highest-precedence record for the cursored name.
func (vv v) Selected(m *app.Model) *scan.SkillRecord {
	names := uniqueNames(m)
	if len(names) == 0 || m.Cursor < 0 || m.Cursor >= len(names) {
		return nil
	}
	name := names[m.Cursor]

	prec := func(k harness.ScopeKind) int {
		switch k {
		case harness.User:
			return 4
		case harness.Project:
			return 3
		case harness.ProjectLocal:
			return 2
		case harness.Plugin:
			return 1
		}
		return 0
	}

	var best *scan.SkillRecord
	for i := range m.Skills {
		sk := &m.Skills[i]
		if sk.Name != name {
			continue
		}
		if best == nil || prec(sk.Scope.Kind) > prec(best.Scope.Kind) {
			best = sk
		}
	}
	if best == nil {
		return nil
	}
	out := *best
	return &out
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

func scopeKindLabel(k harness.ScopeKind) string {
	switch k {
	case harness.User:
		return "user"
	case harness.Project:
		return "project"
	case harness.ProjectLocal:
		return "local"
	case harness.Plugin:
		return "plugin"
	}
	return "?"
}

// cell describes everything we need to render the dots for one
// (skill_name, scope_kind) slot.
type cell struct {
	dots []cellDot
}

type cellDot struct {
	harness  string
	shadowed bool
	parseErr bool
}

func (vv v) Render(m *app.Model, width, height int) string {
	skills := m.FilteredSkills()
	if len(skills) == 0 {
		return ui.DimStyle.Width(width).Height(height).Render("  no skills found")
	}

	// Determine which scope kinds actually have any data, in fixed order.
	kindOrder := []harness.ScopeKind{harness.User, harness.Project, harness.ProjectLocal, harness.Plugin}
	kindsPresent := map[harness.ScopeKind]bool{}
	for _, sk := range m.Skills {
		kindsPresent[sk.Scope.Kind] = true
	}
	var visibleKinds []harness.ScopeKind
	for _, k := range kindOrder {
		if kindsPresent[k] {
			visibleKinds = append(visibleKinds, k)
		}
	}
	if len(visibleKinds) == 0 {
		return ui.DimStyle.Width(width).Height(height).Render("  no scopes")
	}

	// Build the harness presence table:
	//   table[name][kind] = cell
	// Walking m.Skills (not filtered) so dots reflect total presence; the
	// filter only controls which *rows* show up.
	table := map[string]map[harness.ScopeKind]*cell{}
	for _, sk := range m.Skills {
		if table[sk.Name] == nil {
			table[sk.Name] = map[harness.ScopeKind]*cell{}
		}
		c := table[sk.Name][sk.Scope.Kind]
		if c == nil {
			c = &cell{}
			table[sk.Name][sk.Scope.Kind] = c
		}
		c.dots = append(c.dots, cellDot{
			harness:  sk.Scope.Harness,
			shadowed: m.IsShadowed(sk),
			parseErr: sk.ParseErr != nil,
		})
	}
	// Stable dot order within a cell: registry order.
	harnessOrder := map[string]int{}
	for i, h := range m.Harnesses {
		harnessOrder[h.ID()] = i
	}
	for _, byKind := range table {
		for _, c := range byKind {
			sort.Slice(c.dots, func(i, j int) bool {
				return harnessOrder[c.dots[i].harness] < harnessOrder[c.dots[j].harness]
			})
		}
	}

	// Names visible after filtering, deduped, sorted — same order as Navigate uses.
	names := uniqueNames(m)

	// Column widths.
	maxName := 8
	for _, n := range names {
		if len(n) > maxName {
			maxName = len(n)
		}
	}
	nameW := maxName + 2
	if nameW > width/2 {
		nameW = width / 2
	}
	if nameW < 16 {
		nameW = 16
	}

	avail := width - nameW
	cellW := avail / len(visibleKinds)
	if cellW < 10 {
		cellW = 10
	}

	// Top legend: one colored dot per registered harness.
	legend := buildLegend(m.Harnesses)

	// Header.
	hdr := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%-*s", nameW, "skill"))
	for _, k := range visibleKinds {
		hdr += lipgloss.NewStyle().
			Foreground(ui.ColorFg).Bold(true).
			Width(cellW).Render(scopeKindLabel(k))
	}
	rule := ui.DimStyle.Render(strings.Repeat("─", width))

	// Clamp cursor to the visible name list.
	if m.Cursor >= len(names) {
		m.Cursor = len(names) - 1
	}
	if m.Cursor < 0 {
		m.Cursor = 0
	}
	selName := ""
	if m.Cursor < len(names) {
		selName = names[m.Cursor]
	}

	// Body rows.
	maxBody := height - 5
	if maxBody < 1 {
		maxBody = 1
	}
	visStart := 0
	if m.Cursor >= maxBody {
		visStart = m.Cursor - maxBody + 1
	}

	var body []string
	for i := visStart; i < len(names) && len(body) < maxBody; i++ {
		name := names[i]

		nameLabel := name
		if len(nameLabel) > nameW-1 {
			nameLabel = nameLabel[:nameW-2] + "…"
		}

		var row strings.Builder
		row.WriteString(fmt.Sprintf("%-*s", nameW, nameLabel))

		for _, k := range visibleKinds {
			row.WriteString(renderCell(table[name][k], cellW))
		}

		line := row.String()
		if name == selName {
			line = ui.SelectedStyle.Render(line)
		}
		body = append(body, line)
	}

	hint := ui.DimStyle.Render("  ● present  ◐ shadowed (higher scope wins in same harness)  · absent  ! parse error")

	out := []string{legend, "", hdr, rule}
	out = append(out, body...)

	// Pad so `hint` is bottom-anchored within the height we were given.
	// Reserve one row above the hint for breathing room.
	used := len(out) + 2 // body + blank + hint
	if used < height {
		out = append(out, strings.Repeat("\n", height-used-1))
	}
	out = append(out, "", hint)
	return strings.Join(out, "\n")
}

// renderCell turns a cell into a width-padded string of colored dots.
// Each dot is the harness color; shadowed = ◐, parse error = ! in red.
func renderCell(c *cell, width int) string {
	if c == nil || len(c.dots) == 0 {
		return lipgloss.NewStyle().
			Foreground(ui.ColorAbsent).
			Width(width).
			Render("·")
	}
	var dots []string
	for _, d := range c.dots {
		glyph := "●"
		color := ui.HarnessColor(d.harness)
		switch {
		case d.parseErr:
			glyph = "!"
			color = ui.ColorError
		case d.shadowed:
			glyph = "◐"
		}
		dots = append(dots, lipgloss.NewStyle().Foreground(color).Render(glyph))
	}
	content := strings.Join(dots, " ")
	// Pad to cell width.
	pad := width - lipgloss.Width(content)
	if pad > 0 {
		content += strings.Repeat(" ", pad)
	}
	return content
}

// buildLegend renders the harness color key.
func buildLegend(harnesses []harness.Harness) string {
	var parts []string
	parts = append(parts, ui.DimStyle.Render("Harnesses:"))
	for _, h := range harnesses {
		dot := lipgloss.NewStyle().Foreground(h.Color()).Render("●")
		parts = append(parts, dot+" "+h.ID())
	}
	return strings.Join(parts, "  ")
}

func init() { app.RegisterView(v{}) }
