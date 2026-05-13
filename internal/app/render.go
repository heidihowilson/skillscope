package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sethgho/skillscope/internal/ui"
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
	statusBar := m.renderStatusBar()

	contentH := m.Height - lipgloss.Height(tabBar) - lipgloss.Height(statusBar)
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

	if m.SearchActive {
		searchBar := m.renderSearchBar()
		return lipgloss.JoinVertical(lipgloss.Left, tabBar, content, searchBar)
	}

	return lipgloss.JoinVertical(lipgloss.Left, tabBar, content, statusBar)
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
		"  Esc             clear search",
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
				ui.ErrorStyle.Render(fmt.Sprintf("Delete %q?", m.OpSkill.Name)),
				ui.DimStyle.Render(m.OpSkill.Path),
				"",
				"  press y to confirm, any other key to cancel",
			)
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
