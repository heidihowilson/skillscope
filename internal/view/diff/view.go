package diff

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
	"github.com/muesli/reflow/wordwrap"
	"gopkg.in/yaml.v3"
)

type v struct{}

func (v) ID() string      { return "diff" }
func (v) Name() string    { return "Diff" }
func (v) KeyHint() string { return "3" }

func (v) Init(m *app.Model) tea.Cmd { return nil }

func (vv v) Update(m *app.Model, msg tea.Msg) (app.View, tea.Cmd) {
	return vv, nil
}

// uniqueNames returns the visible names alphabetically.
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

func (vv v) Selected(m *app.Model) *scan.SkillRecord {
	names := uniqueNames(m)
	if len(names) == 0 || m.Cursor < 0 || m.Cursor >= len(names) {
		return nil
	}
	name := names[m.Cursor]
	for i := range m.Skills {
		if m.Skills[i].Name == name {
			out := m.Skills[i]
			return &out
		}
	}
	return nil
}

func (vv v) Render(m *app.Model, width, height int) string {
	names := uniqueNames(m)
	if len(names) == 0 {
		return ui.DimStyle.Width(width).Height(height).Render("  no skills found")
	}
	if m.Cursor >= len(names) {
		m.Cursor = len(names) - 1
	}
	if m.Cursor < 0 {
		m.Cursor = 0
	}

	// Index skills by name to count multi-scope availability.
	byName := map[string][]scan.SkillRecord{}
	for _, sk := range m.Skills {
		byName[sk.Name] = append(byName[sk.Name], sk)
	}

	// Layout: 30% master, 70% detail.
	masterW := width / 3
	if masterW < 20 {
		masterW = 20
	}
	if masterW > 40 {
		masterW = 40
	}
	detailW := width - masterW - 1

	master := renderMaster(names, byName, m.Cursor, masterW, height)
	detail := renderDetail(names[m.Cursor], byName[names[m.Cursor]], detailW, height)

	divider := strings.Repeat("│\n", height-1) + "│"
	dv := lipgloss.NewStyle().Foreground(ui.ColorBorder).Render(divider)
	return lipgloss.JoinHorizontal(lipgloss.Top, master, dv, detail)
}

// renderMaster renders the skill list on the left.
func renderMaster(names []string, byName map[string][]scan.SkillRecord,
	cursor, width, height int,
) string {
	var rows []string
	rows = append(rows, ui.BoldStyle.Render("skills"))
	rows = append(rows, ui.DimStyle.Render(strings.Repeat("─", width)))

	// Scroll window.
	maxRows := height - 2
	if maxRows < 1 {
		maxRows = 1
	}
	start := 0
	if cursor >= maxRows {
		start = cursor - maxRows + 1
	}
	end := start + maxRows
	if end > len(names) {
		end = len(names)
	}

	for i := start; i < end; i++ {
		name := names[i]
		marker := " "
		recs := byName[name]
		// Skills with multiple records in the same harness are diffable.
		diffable := false
		seen := map[string]int{}
		for _, r := range recs {
			seen[r.Scope.Harness]++
			if seen[r.Scope.Harness] > 1 {
				diffable = true
				break
			}
		}
		if diffable {
			marker = lipgloss.NewStyle().Foreground(ui.ColorShadow).Render("⇄")
		}

		label := name
		if len(label) > width-4 {
			label = label[:width-5] + "…"
		}
		line := fmt.Sprintf(" %s %-*s", marker, width-3, label)
		if i == cursor {
			line = ui.SelectedStyle.Render(line)
		} else if !diffable {
			line = ui.DimStyle.Render(line)
		}
		rows = append(rows, line)
	}

	// Pad to height so the divider extends.
	for len(rows) < height {
		rows = append(rows, strings.Repeat(" ", width))
	}
	return strings.Join(rows, "\n")
}

// renderDetail renders the diff panel for the given skill name.
func renderDetail(name string, recs []scan.SkillRecord, width, height int) string {
	if len(recs) == 0 {
		return ui.DimStyle.Width(width).Height(height).Render("\n  no records")
	}

	title := ui.BoldStyle.Render("diff: " + name)

	// Pick a harness that has multiple scopes; if none, show single record.
	groups := map[string][]scan.SkillRecord{}
	for _, r := range recs {
		groups[r.Scope.Harness] = append(groups[r.Scope.Harness], r)
	}

	var pair []scan.SkillRecord
	var harnessID string
	for hid, rs := range groups {
		if len(rs) >= 2 {
			pair = rs
			harnessID = hid
			break
		}
	}

	if pair == nil {
		// Only single-scope copies — show a helpful message.
		var sb strings.Builder
		sb.WriteString(title + "\n")
		sb.WriteString(ui.DimStyle.Render(strings.Repeat("─", width)) + "\n\n")
		sb.WriteString(ui.DimStyle.Render("  ⇄ no diff available — this skill only has one copy") + "\n")
		sb.WriteString(ui.DimStyle.Render("    per harness (no shadowing).") + "\n\n")
		sb.WriteString("  Existing copies:\n")
		for _, r := range recs {
			dot := lipgloss.NewStyle().Foreground(ui.HarnessColor(r.Scope.Harness)).Render("●")
			sb.WriteString(fmt.Sprintf("    %s %s · %s\n", dot, r.Scope.Harness, r.Scope.Kind))
		}
		out := sb.String()
		// Pad height.
		got := strings.Count(out, "\n")
		for got < height-1 {
			out += "\n"
			got++
		}
		return out
	}

	// Sort pair by scope precedence (winner first).
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
	sort.Slice(pair, func(i, j int) bool {
		return prec(pair[i].Scope.Kind) > prec(pair[j].Scope.Kind)
	})

	left := pair[0]
	right := pair[1]

	halfW := (width - 3) / 2
	if halfW < 10 {
		halfW = 10
	}

	hColor := ui.HarnessColor(harnessID)

	// Reserve 2 cols on each side for the gutter marker (" X "), plus 3
	// cells in the middle for the divider " │ ".
	gutter := 2
	colW := halfW - gutter
	if colW < 8 {
		colW = 8
	}

	leftHeader := lipgloss.NewStyle().Foreground(hColor).Bold(true).
		Width(halfW).
		Render(fmt.Sprintf("%s · wins", left.Scope.Kind))
	rightHeader := lipgloss.NewStyle().Foreground(ui.ColorFgDim).
		Width(halfW).
		Render(fmt.Sprintf("%s · shadowed", right.Scope.Kind))

	leftLines := strings.Split(recordToText(left), "\n")
	rightLines := strings.Split(recordToText(right), "\n")

	pairs := AlignLines(leftLines, rightLines)

	var rows []string
	rows = append(rows, title)
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
		leftHeader, ui.DimStyle.Render(" │ "), rightHeader))
	rows = append(rows, strings.Repeat("─", halfW)+"─┼─"+strings.Repeat("─", halfW))

	maxRows := height - 3
	for _, p := range pairs {
		if len(rows)-3 >= maxRows {
			rows = append(rows, ui.DimStyle.Render(fmt.Sprintf("  … %d more rows", countRemaining(pairs, len(rows)-3))))
			break
		}
		rows = append(rows, renderPair(p, colW)...)
		if len(rows)-3 >= maxRows {
			break
		}
	}
	for len(rows) < height {
		rows = append(rows, "")
	}
	return strings.Join(rows, "\n")
}

// renderPair wraps both sides of a pair to colW, then emits one or more
// aligned rows. Each row includes a gutter marker that indicates the op.
func renderPair(p DiffPair, colW int) []string {
	leftLines := wrapLines(p.Left, colW)
	rightLines := wrapLines(p.Right, colW)

	var leftMarker, rightMarker string
	var leftStyle, rightStyle lipgloss.Style

	base := lipgloss.NewStyle()
	addStyle := lipgloss.NewStyle().Foreground(ui.ColorPresent)
	delStyle := lipgloss.NewStyle().Foreground(ui.ColorError)
	chgStyle := lipgloss.NewStyle().Foreground(ui.ColorShadow)

	switch p.Op {
	case OpEqual:
		leftMarker, rightMarker = " ", " "
		leftStyle, rightStyle = base, base
	case OpChanged:
		leftMarker, rightMarker = "~", "~"
		leftStyle, rightStyle = chgStyle, chgStyle
	case OpDelete:
		leftMarker, rightMarker = "-", " "
		leftStyle = delStyle
		rightStyle = base
	case OpInsert:
		leftMarker, rightMarker = " ", "+"
		leftStyle = base
		rightStyle = addStyle
	}

	n := len(leftLines)
	if len(rightLines) > n {
		n = len(rightLines)
	}
	if n == 0 {
		n = 1
	}

	gutterStyleL := lipgloss.NewStyle().Foreground(ui.ColorFgDim)
	gutterStyleR := lipgloss.NewStyle().Foreground(ui.ColorFgDim)
	if p.Op == OpDelete {
		gutterStyleL = delStyle.Bold(true)
	}
	if p.Op == OpInsert {
		gutterStyleR = addStyle.Bold(true)
	}
	if p.Op == OpChanged {
		gutterStyleL = chgStyle.Bold(true)
		gutterStyleR = chgStyle.Bold(true)
	}

	var rows []string
	for i := 0; i < n; i++ {
		ll, rr := "", ""
		if i < len(leftLines) {
			ll = leftLines[i]
		}
		if i < len(rightLines) {
			rr = rightLines[i]
		}

		// Only show gutter marker on the first wrap line, subsequent
		// wrap lines get a continuation space.
		lm, rm := " ", " "
		if i == 0 {
			lm, rm = leftMarker, rightMarker
		}

		// Pad cells manually — lipgloss's Width() returns empty for
		// empty input, which collapses columns on continuation rows.
		leftCell := gutterStyleL.Render(lm+" ") + padCell(ll, leftStyle, colW)
		rightCell := gutterStyleR.Render(rm+" ") + padCell(rr, rightStyle, colW)

		rows = append(rows, leftCell+ui.DimStyle.Render(" │ ")+rightCell)
	}
	return rows
}

// padCell styles `content` and pads it (with raw spaces) so its visible
// width is exactly `width` cells.
func padCell(content string, style lipgloss.Style, width int) string {
	styled := style.Render(content)
	pad := width - lipgloss.Width(styled)
	if pad < 0 {
		pad = 0
	}
	return styled + strings.Repeat(" ", pad)
}

// wrapLines word-wraps s to width, returning the wrap-lines. Empty input
// returns a single empty string so callers can still emit one row.
func wrapLines(s string, width int) []string {
	if s == "" {
		return []string{""}
	}
	wrapped := wordwrap.String(s, width)
	return strings.Split(wrapped, "\n")
}

// countRemaining roughly estimates how many wrap-rows haven't fit yet.
func countRemaining(pairs []DiffPair, rendered int) int {
	if rendered >= len(pairs) {
		return 0
	}
	return len(pairs) - rendered
}

// recordToText returns the YAML frontmatter + body for a record as a
// single string. No truncation — the diff renderer wraps per column.
func recordToText(sk scan.SkillRecord) string {
	var sb strings.Builder
	if sk.Frontmatter != nil {
		out, _ := yaml.Marshal(sk.Frontmatter)
		sb.WriteString(strings.TrimRight(string(out), "\n"))
	}
	if sk.Body != "" {
		sb.WriteString("\n")
		sb.WriteString("---")
		sb.WriteString("\n")
		sb.WriteString(strings.TrimRight(sk.Body, "\n"))
	}
	return sb.String()
}

// Hints implements ActionHinter.
func (vv v) Hints(m *app.Model) []ui.ActionHint {
	return []ui.ActionHint{
		{Key: "↑↓", Label: "row"},
		{Key: "/", Label: "search"},
		{Key: "Space", Label: "preview"},
		{Key: "c", Label: "copy"},
		{Key: "m", Label: "move"},
	}
}

func init() { app.RegisterView(v{}) }
