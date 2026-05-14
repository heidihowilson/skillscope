package diff

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

	leftHeader := lipgloss.NewStyle().Foreground(hColor).Bold(true).
		Width(halfW).
		Render(fmt.Sprintf("%s · wins", left.Scope.Kind))
	rightHeader := lipgloss.NewStyle().Foreground(ui.ColorFgDim).
		Width(halfW).
		Render(fmt.Sprintf("%s · shadowed", right.Scope.Kind))

	leftBody := formatRecord(left, halfW)
	rightBody := formatRecord(right, halfW)

	leftLines := strings.Split(leftBody, "\n")
	rightLines := strings.Split(rightBody, "\n")
	maxL := len(leftLines)
	if len(rightLines) > maxL {
		maxL = len(rightLines)
	}
	for len(leftLines) < maxL {
		leftLines = append(leftLines, "")
	}
	for len(rightLines) < maxL {
		rightLines = append(rightLines, "")
	}

	var rows []string
	rows = append(rows, title)
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
		leftHeader, ui.DimStyle.Render(" │ "), rightHeader))
	rows = append(rows, strings.Repeat("─", halfW)+"─┼─"+strings.Repeat("─", halfW))
	for i := range leftLines {
		if len(rows)-3 >= height-3 {
			break
		}
		l := leftLines[i]
		r := rightLines[i]
		if l != r {
			l = lipgloss.NewStyle().Foreground(ui.ColorPresent).Render(l)
			r = lipgloss.NewStyle().Foreground(ui.ColorShadow).Render(r)
		}
		rows = append(rows, fmt.Sprintf("%-*s │ %s", halfW, l, r))
	}
	for len(rows) < height {
		rows = append(rows, "")
	}
	return strings.Join(rows, "\n")
}

func formatRecord(sk scan.SkillRecord, width int) string {
	var sb strings.Builder
	if sk.Frontmatter != nil {
		out, _ := yaml.Marshal(sk.Frontmatter)
		sb.WriteString(string(out))
	}
	if sk.Body != "" {
		sb.WriteString("\n")
		sb.WriteString(sk.Body)
	}
	var lines []string
	for _, l := range strings.Split(sb.String(), "\n") {
		if len(l) > width {
			l = l[:width-1] + "…"
		}
		lines = append(lines, l)
	}
	return strings.Join(lines, "\n")
}

// Hints implements ActionHinter.
func (vv v) Hints(m *app.Model) []ui.ActionHint {
	return []ui.ActionHint{
		{Key: "↑↓", Label: "row"},
		{Key: "/", Label: "search"},
		{Key: "p", Label: "preview"},
		{Key: "c", Label: "copy"},
		{Key: "m", Label: "move"},
	}
}

func init() { app.RegisterView(v{}) }
