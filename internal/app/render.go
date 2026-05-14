package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/heidihowilson/skillscope/internal/ui"
)

const minWidth, minHeight = 80, 24

// View implements tea.Model.
func (m *Model) View() string {
	if m.Width < minWidth || m.Height < minHeight {
		msg := fmt.Sprintf("Terminal too small (%dx%d)\nMinimum: %dx%d\nPlease resize.", m.Width, m.Height, minWidth, minHeight)
		return ui.TinyTermStyle.Width(m.Width).Height(m.Height).Render(msg)
	}

	if m.Loading {
		return ui.DimStyle.Width(m.Width).Height(m.Height).Render("\n\n  scanning skills…")
	}

	if m.ShowHelp {
		return m.renderHelp()
	}

	if m.OpMode != OpNone {
		return m.renderOpOverlay()
	}

	tabBar := m.renderTabBar()
	toolbar := m.renderToolbar()
	statusBar := m.renderStatusBar()

	contentH := m.Height - lipgloss.Height(tabBar) - lipgloss.Height(toolbar) - lipgloss.Height(statusBar)
	if contentH < 1 {
		contentH = 1
	}

	var content string
	if m.PreviewMode != ui.PreviewOff && m.Focus != FocusPreview {
		// Split: 60% view, 40% preview.
		mainW := (m.Width * 60) / 100
		prevW := m.Width - mainW - 1
		mainContent := m.renderMainView(mainW, contentH)
		sk := m.SelectedSkill()
		prevContent := ui.RenderPreview(sk, m.PreviewMode, prevW, contentH)
		divider := lipgloss.NewStyle().
			Foreground(ui.ColorBorder).
			Width(1).
			Height(contentH).
			Render(strings.Repeat("│\n", contentH))
		content = lipgloss.JoinHorizontal(lipgloss.Top, mainContent, divider, prevContent)
	} else if m.PreviewMode != ui.PreviewOff && m.Focus == FocusPreview {
		sk := m.SelectedSkill()
		content = ui.RenderPreview(sk, m.PreviewMode, m.Width, contentH)
	} else {
		content = m.renderMainView(m.Width, contentH)
	}

	// Pad content to exactly contentH rows so the toolbar/status bar
	// anchor to the bottom of the terminal instead of floating wherever
	// the view's natural output ends.
	content = padToHeight(content, contentH, m.Width)

	if m.SearchActive {
		searchBar := m.renderSearchBar()
		return lipgloss.JoinVertical(lipgloss.Left, tabBar, content, toolbar, searchBar)
	}

	return lipgloss.JoinVertical(lipgloss.Left, tabBar, content, toolbar, statusBar)
}

// padToHeight appends blank lines to `s` until it has exactly `height`
// rows. Each blank line is `width` spaces so it doesn't collapse when
// joined vertically with other blocks.
func padToHeight(s string, height, width int) string {
	have := lipgloss.Height(s)
	if have >= height {
		return s
	}
	blank := strings.Repeat(" ", width)
	pad := strings.Repeat("\n"+blank, height-have)
	return s + pad
}

// renderToolbar builds the contextual action hint row. Views can implement
// ActionHinter to override the default hint set.
func (m *Model) renderToolbar() string {
	var hints []ui.ActionHint

	// Operation overlays take priority.
	switch m.OpMode {
	case OpCopyPicker, OpMovePicker:
		hints = []ui.ActionHint{
			{Key: "1-9", Label: "pick scope"},
			{Key: "↑↓", Label: "navigate"},
			{Key: "Enter", Label: "confirm"},
			{Key: "Esc", Label: "cancel"},
		}
		return ui.RenderToolbar(hints, m.Width)
	case OpDeleteConfirm:
		hints = []ui.ActionHint{
			{Key: "type name", Label: "to confirm"},
			{Key: "Enter", Label: "delete"},
			{Key: "Esc", Label: "cancel"},
		}
		return ui.RenderToolbar(hints, m.Width)
	}

	if m.SearchActive {
		hints = []ui.ActionHint{
			{Key: "Enter", Label: "apply"},
			{Key: "Esc", Label: "cancel"},
		}
		return ui.RenderToolbar(hints, m.Width)
	}

	if m.PreviewMode != ui.PreviewOff {
		hints = []ui.ActionHint{
			{Key: "p", Label: "cycle preview"},
			{Key: "Esc", Label: "close preview"},
			{Key: "Tab", Label: "focus pane"},
			{Key: "c", Label: "copy"},
			{Key: "m", Label: "move"},
		}
		return ui.RenderToolbar(hints, m.Width)
	}

	// View-specific hints when available.
	if len(m.Views) > 0 && m.ActiveView < len(m.Views) {
		if hinter, ok := m.Views[m.ActiveView].(ActionHinter); ok {
			return ui.RenderToolbar(hinter.Hints(m), m.Width)
		}
	}

	// Default hints.
	hints = []ui.ActionHint{
		{Key: "/", Label: "search"},
		{Key: "f", Label: "filter"},
		{Key: "p", Label: "preview"},
		{Key: "c", Label: "copy"},
		{Key: "m", Label: "move"},
		{Key: "d", Label: "delete"},
	}
	return ui.RenderToolbar(hints, m.Width)
}

func (m *Model) renderTabBar() string {
	var tabs []string
	for i, v := range m.Views {
		label := fmt.Sprintf(" %s %s ", v.KeyHint(), v.Name())
		if i == m.ActiveView {
			tabs = append(tabs, ui.TabActiveStyle.Render(label))
		} else {
			tabs = append(tabs, ui.TabBarStyle.Render(label))
		}
	}
	bar := strings.Join(tabs, "")
	// Pad to full width.
	pad := m.Width - lipgloss.Width(bar)
	if pad > 0 {
		bar += ui.TabBarStyle.Width(pad).Render("")
	}
	return bar
}

func (m *Model) renderStatusBar() string {
	filtered := m.FilteredSkills()

	var hf []string
	for k := range m.HarnessFilter {
		hf = append(hf, k)
	}
	sf := ""
	if m.ScopeFilterOn {
		sf = m.ScopeFilter.String()
	}

	sb := ui.StatusBar{
		HarnessFilter: hf,
		ScopeFilter:   sf,
		Query:         m.SearchQuery,
		Visible:       len(filtered),
		Total:         len(m.Skills),
		Width:         m.Width,
		Msg:           m.StatusMsg,
	}
	return sb.Render()
}

func (m *Model) renderSearchBar() string {
	return ui.StatusBarStyle.Width(m.Width).Render("/ " + m.SearchInput.View())
}

func (m *Model) renderMainView(width, height int) string {
	if len(m.Views) == 0 {
		return ui.DimStyle.Width(width).Height(height).Render("  no views registered")
	}
	v := m.Views[m.ActiveView]
	return v.Render(m, width, height)
}

func (m *Model) renderHelp() string {
	lines := []string{
		ui.BoldStyle.Render("skillscope — help"),
		"",
		ui.BoldStyle.Render("Navigation"),
		"  j/k / ↑↓       move cursor",
		"  1-9             jump to view",
		"  v/V             cycle views",
		"  Tab             switch panel focus",
		"",
		ui.BoldStyle.Render("Filters"),
		"  /               fuzzy search",
		"  Esc             close preview / clear search",
		"  f               cycle harness filter",
		"  F               clear harness filter",
		"  s               cycle scope filter",
		"  g               toggle shadowed-only",
		"",
		ui.BoldStyle.Render("Actions"),
		"  p               preview (off / raw / rendered)",
		"  e               open in $EDITOR",
		"  c               copy to scope…",
		"  m               move to scope…",
		"  d               delete (confirm y)",
		"  y               yank path",
		"  R               re-scan",
		"",
		ui.BoldStyle.Render("Global"),
		"  q               quit",
		"  ?               this help",
		"",
		ui.DimStyle.Render("press any key to close"),
	}
	content := strings.Join(lines, "\n")
	return lipgloss.NewStyle().
		Width(m.Width).Height(m.Height).
		Padding(1, 3).
		Render(content)
}

func (m *Model) renderOpOverlay() string {
	var lines []string
	switch m.OpMode {
	case OpDeleteConfirm:
		if m.OpSkill != nil {
			lines = append(lines,
				ui.ErrorStyle.Bold(true).Render(fmt.Sprintf("DELETE %s", m.OpSkill.Name)),
				ui.DimStyle.Render(m.OpSkill.Path),
				"",
				ui.ErrorStyle.Render("This cannot be undone."),
				"",
				fmt.Sprintf("Type %s to confirm:", ui.BoldStyle.Render(m.OpSkill.Name)),
				m.DeleteInput.View(),
			)
			if m.DeleteMismatch {
				lines = append(lines, "", ui.ErrorStyle.Render("✗ name didn't match — type it exactly"))
			}
			lines = append(lines, "", ui.DimStyle.Render("  Enter to delete, Esc to cancel"))
		}
	case OpCopyPicker, OpMovePicker:
		action := "Copy"
		if m.OpMode == OpMovePicker {
			action = "Move"
		}
		if m.OpSkill != nil {
			lines = append(lines, ui.BoldStyle.Render(fmt.Sprintf("%s %q to:", action, m.OpSkill.Name)), "")
			for i, s := range m.OpScopes {
				prefix := fmt.Sprintf("  %d. ", i+1)
				label := fmt.Sprintf("%s  %s  %s", s.Harness, s.Kind, s.Path)
				if i == m.OpCursor {
					lines = append(lines, ui.SelectedStyle.Render(prefix+label))
				} else {
					lines = append(lines, prefix+label)
				}
			}
			lines = append(lines, "", ui.DimStyle.Render("  ↑↓/Enter to select, 1-9 shortcut, Esc to cancel"))
		}
	}

	content := strings.Join(lines, "\n")
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorAccent).
		Padding(1, 2).
		Render(content)

	// Center the box.
	bw := lipgloss.Width(box)
	bh := lipgloss.Height(box)
	padL := (m.Width - bw) / 2
	padT := (m.Height - bh) / 2
	if padL < 0 {
		padL = 0
	}
	if padT < 0 {
		padT = 0
	}
	top := strings.Repeat("\n", padT)
	left := strings.Repeat(" ", padL)

	var rows []string
	for _, line := range strings.Split(box, "\n") {
		rows = append(rows, left+line)
	}
	return top + strings.Join(rows, "\n")
}
