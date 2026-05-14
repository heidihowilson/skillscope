package diff

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sethgho/skillscope/internal/app"
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

func (vv v) Render(m *app.Model, width, height int) string {
	sel := m.SelectedSkill()
	if sel == nil {
		return ui.DimStyle.Width(width).Height(height).Render("  select a skill to diff")
	}

	// Find all records with same name in same harness.
	var siblings []scan.SkillRecord
	for _, sk := range m.Skills {
		if sk.Name == sel.Name && sk.Scope.Harness == sel.Scope.Harness {
			siblings = append(siblings, sk)
		}
	}
	if len(siblings) < 2 {
		return ui.DimStyle.Width(width).Height(height).Render(
			fmt.Sprintf("  %q only exists in one scope — nothing to diff", sel.Name))
	}

	// Sort by precedence: user first.
	precedence := func(sk scan.SkillRecord) int {
		switch sk.Scope.Kind {
		case 0: // User
			return 3
		case 1: // Project
			return 2
		case 2: // ProjectLocal
			return 1
		}
		return 0
	}
	sort.Slice(siblings, func(i, j int) bool {
		return precedence(siblings[i]) > precedence(siblings[j])
	})

	// Show side-by-side for first two.
	left := siblings[0]
	right := siblings[1]

	halfW := (width - 3) / 2

	leftHeader := lipgloss.NewStyle().
		Foreground(ui.HarnessColor(left.Scope.Harness)).Bold(true).
		Width(halfW).
		Render(fmt.Sprintf("%s [%s] wins", left.Scope.Kind, left.Scope.Harness))
	rightHeader := lipgloss.NewStyle().
		Foreground(ui.ColorFgDim).
		Width(halfW).
		Render(fmt.Sprintf("%s [%s] shadowed", right.Scope.Kind, right.Scope.Harness))

	leftBody := formatRecord(left, halfW)
	rightBody := formatRecord(right, halfW)

	// Align line counts.
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

	header := lipgloss.JoinHorizontal(lipgloss.Top,
		leftHeader, ui.DimStyle.Render(" │ "), rightHeader)
	sep := strings.Repeat("─", halfW) + "─┼─" + strings.Repeat("─", halfW)

	var rows []string
	rows = append(rows, header, sep)
	for i := range leftLines {
		if len(rows)-2 >= height-3 {
			break
		}
		lLine := leftLines[i]
		rLine := rightLines[i]
		// Highlight differing lines.
		if lLine != rLine {
			lLine = lipgloss.NewStyle().Foreground(ui.ColorPresent).Render(lLine)
			rLine = lipgloss.NewStyle().Foreground(ui.ColorShadow).Render(rLine)
		}
		rows = append(rows, fmt.Sprintf("%-*s │ %s", halfW, lLine, rLine))
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
	// Truncate lines to width.
	var lines []string
	for _, l := range strings.Split(sb.String(), "\n") {
		if len(l) > width {
			l = l[:width-1] + "…"
		}
		lines = append(lines, l)
	}
	return strings.Join(lines, "\n")
}

func init() { app.RegisterView(v{}) }
