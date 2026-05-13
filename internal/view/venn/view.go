package venn

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sethgho/skillscope/internal/app"
	"github.com/sethgho/skillscope/internal/harness"
	"github.com/sethgho/skillscope/internal/ui"
)

type v struct{}

func (v) ID() string      { return "venn" }
func (v) Name() string    { return "Venn" }
func (v) KeyHint() string { return "3" }

func (v) Init(m *app.Model) tea.Cmd { return nil }

func (vv v) Update(m *app.Model, msg tea.Msg) (app.View, tea.Cmd) {
	return vv, nil
}

func (vv v) Render(m *app.Model, width, height int) string {
	skills := m.FilteredSkills()

	// Determine which harness to show (first in filter, or first available).
	hid := ""
	if len(m.HarnessFilter) > 0 {
		for k := range m.HarnessFilter {
			hid = k
			break
		}
	}
	if hid == "" && len(m.Harnesses) > 0 {
		hid = m.Harnesses[0].ID()
	}

	// Collect skills for this harness by scope kind.
	byKind := map[harness.ScopeKind][]string{}
	for _, sk := range m.Skills {
		if sk.Scope.Harness != hid {
			continue
		}
		byKind[sk.Scope.Kind] = append(byKind[sk.Scope.Kind], sk.Name)
	}
	for k := range byKind {
		sort.Strings(byKind[k])
	}

	sel := m.SelectedSkill()
	selName := ""
	if sel != nil {
		selName = sel.Name
	}

	var sb strings.Builder

	// Header
	hColor := ui.HarnessColor(hid)
	sb.WriteString(lipgloss.NewStyle().Foreground(hColor).Bold(true).Render("Venn: "+hid) + "\n")
	if len(m.HarnessFilter) == 0 {
		sb.WriteString(ui.DimStyle.Render("  (press f to filter by harness)") + "\n")
	}
	sb.WriteString("\n")

	// Three-circle ASCII diagram.
	kinds := []harness.ScopeKind{harness.User, harness.Project, harness.ProjectLocal}
	labels := []string{"User", "Project", "Local"}
	colors := []lipgloss.Color{
		lipgloss.Color("#68D391"),
		lipgloss.Color("#63B3ED"),
		lipgloss.Color("#F6AD55"),
	}

	const circleW = 24
	for i, k := range kinds {
		names := byKind[k]
		active := false
		for _, n := range names {
			if n == selName {
				active = true
				break
			}
		}

		borderColor := colors[i]
		if !active {
			borderColor = ui.ColorBorder
		}
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Width(circleW).
			Padding(0, 1)

		var lines []string
		label := labels[i]
		if active {
			label = lipgloss.NewStyle().Foreground(colors[i]).Bold(true).Render(label)
		} else {
			label = ui.DimStyle.Render(label)
		}
		lines = append(lines, label+" ("+fmt.Sprintf("%d", len(names))+")")
		for _, n := range names {
			entry := "  " + n
			if n == selName {
				entry = lipgloss.NewStyle().Foreground(colors[i]).Bold(true).Render("▶ " + n)
			}
			lines = append(lines, entry)
		}

		sb.WriteString(style.Render(strings.Join(lines, "\n")) + "\n")
	}

	_ = skills
	_ = width
	_ = height
	return sb.String()
}

func init() { app.RegisterView(v{}) }
