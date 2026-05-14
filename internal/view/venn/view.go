package venn

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

func (v) ID() string      { return "venn" }
func (v) Name() string    { return "Venn" }
func (v) KeyHint() string { return "3" }

func (v) Init(m *app.Model) tea.Cmd { return nil }

func (vv v) Update(m *app.Model, msg tea.Msg) (app.View, tea.Cmd) {
	return vv, nil
}

// Hints implements the ActionHinter optional interface.
func (vv v) Hints(m *app.Model) []ui.ActionHint {
	return []ui.ActionHint{
		{Key: "f", Label: "change harness"},
		{Key: "1-5", Label: "view"},
		{Key: "p", Label: "preview"},
		{Key: "?", Label: "help"},
	}
}

func (vv v) Render(m *app.Model, width, height int) string {
	// Pick the focus harness: first in filter, else first registered.
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
	if hid == "" {
		return ui.DimStyle.Width(width).Height(height).Render("  no harness available")
	}

	// Index this harness's skills by name -> set of scope kinds.
	byName := map[string]map[harness.ScopeKind]scan.SkillRecord{}
	for _, sk := range m.Skills {
		if sk.Scope.Harness != hid {
			continue
		}
		if byName[sk.Name] == nil {
			byName[sk.Name] = map[harness.ScopeKind]scan.SkillRecord{}
		}
		byName[sk.Name][sk.Scope.Kind] = sk
	}

	// Classify into regions.
	var (
		userOnly    []string
		projectOnly []string
		localOnly   []string
		userProj    []string
		userLocal   []string
		projLocal   []string
		allThree    []string
	)
	for name, m := range byName {
		hasU := false
		hasP := false
		hasL := false
		if _, ok := m[harness.User]; ok {
			hasU = true
		}
		if _, ok := m[harness.Project]; ok {
			hasP = true
		}
		if _, ok := m[harness.ProjectLocal]; ok {
			hasL = true
		}
		switch {
		case hasU && hasP && hasL:
			allThree = append(allThree, name)
		case hasU && hasP:
			userProj = append(userProj, name)
		case hasU && hasL:
			userLocal = append(userLocal, name)
		case hasP && hasL:
			projLocal = append(projLocal, name)
		case hasU:
			userOnly = append(userOnly, name)
		case hasP:
			projectOnly = append(projectOnly, name)
		case hasL:
			localOnly = append(localOnly, name)
		}
	}
	sort.Strings(userOnly)
	sort.Strings(projectOnly)
	sort.Strings(localOnly)
	sort.Strings(userProj)
	sort.Strings(userLocal)
	sort.Strings(projLocal)
	sort.Strings(allThree)

	uniqueCount := len(byName)
	instanceCount := 0
	for _, sk := range m.Skills {
		if sk.Scope.Harness == hid {
			instanceCount++
		}
	}

	hColor := ui.HarnessColor(hid)
	title := lipgloss.NewStyle().
		Foreground(hColor).
		Bold(true).
		Render(fmt.Sprintf("%s · scope membership", hid))
	subtitle := ui.DimStyle.Render(fmt.Sprintf(
		"%d unique skills across %d instances    (f to change harness)",
		uniqueCount, instanceCount))

	// Compact 3-circle Venn-ish header. Counts shown in each region.
	venn := renderVennArt(
		len(userOnly), len(projectOnly), len(localOnly),
		len(userProj), len(userLocal), len(projLocal),
		len(allThree),
		hColor,
	)

	// Region listings — two-column layout.
	colW := (width - 4) / 2
	if colW < 20 {
		colW = 20
	}

	col1 := []string{
		regionHeader("user only", len(userOnly), ui.ColorPresent),
		listBody(userOnly, colW),
		"",
		regionHeader("project only", len(projectOnly), ui.ColorAccent),
		listBody(projectOnly, colW),
		"",
		regionHeader("local only", len(localOnly), ui.ColorShadow),
		listBody(localOnly, colW),
	}
	col2 := []string{
		regionHeader("user ∩ project", len(userProj), ui.ColorShadow),
		listBody(userProj, colW),
		"",
		regionHeader("user ∩ local", len(userLocal), ui.ColorShadow),
		listBody(userLocal, colW),
		"",
		regionHeader("project ∩ local", len(projLocal), ui.ColorShadow),
		listBody(projLocal, colW),
		"",
		regionHeader("all three", len(allThree), ui.ColorAccent),
		listBody(allThree, colW),
	}

	leftStyle := lipgloss.NewStyle().Width(colW).MaxHeight(height - 8)
	rightStyle := lipgloss.NewStyle().Width(colW).MaxHeight(height - 8)
	leftBlock := leftStyle.Render(strings.Join(col1, "\n"))
	rightBlock := rightStyle.Render(strings.Join(col2, "\n"))

	regions := lipgloss.JoinHorizontal(lipgloss.Top, leftBlock, "  ", rightBlock)

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		subtitle,
		"",
		venn,
		"",
		regions,
	)
}

func regionHeader(label string, count int, color lipgloss.Color) string {
	return lipgloss.NewStyle().Foreground(color).Bold(true).Render(
		fmt.Sprintf("%s (%d)", label, count),
	)
}

func listBody(items []string, width int) string {
	if len(items) == 0 {
		return ui.DimStyle.Render("  —")
	}
	var lines []string
	for _, it := range items {
		lines = append(lines, "  • "+it)
	}
	return strings.Join(lines, "\n")
}

// renderVennArt draws a compact 3-circle Venn with counts in each region.
// Layout (fixed-width ASCII):
//
//        ╭─────── user ────────╮
//        │  uO=3               │
//        │       ╭─────── project ────╮
//        │       │  uP=1     pO=2     │
//        │       │  all=0             │
//        ╰───────┤   uL=0    pL=0     │
//                │                    │
//        ╭───── local ────────╮       │
//        │  lO=1              │       │
//        ╰────────────────────┴───────╯
//
// We keep it text-only and color-coded by harness.
func renderVennArt(uO, pO, lO, uP, uL, pL, all int, harnessColor lipgloss.Color) string {
	col := lipgloss.NewStyle().Foreground(harnessColor).Bold(true)
	dim := ui.DimStyle

	// Three pseudo-circles, side-by-side, with counts.
	circle := func(label string, count int) string {
		head := col.Render(label)
		body := lipgloss.NewStyle().
			Width(14).
			Align(lipgloss.Center).
			Render(fmt.Sprintf("%d", count))
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(harnessColor).
			Width(14).
			Render(fmt.Sprintf("%s\n%s", head, body))
	}

	intersection := func(label string, count int) string {
		return dim.Render(fmt.Sprintf("%-16s %d", label, count))
	}

	sets := lipgloss.JoinHorizontal(lipgloss.Top,
		circle("user", uO+uP+uL+all),
		"  ",
		circle("project", pO+uP+pL+all),
		"  ",
		circle("local", lO+uL+pL+all),
	)

	intersections := []string{
		intersection("user ∩ project", uP+all),
		intersection("user ∩ local", uL+all),
		intersection("project ∩ local", pL+all),
		intersection("all three", all),
	}
	intersectBlock := strings.Join(intersections, "    ")

	return sets + "\n\n" + intersectBlock
}

func init() { app.RegisterView(v{}) }
