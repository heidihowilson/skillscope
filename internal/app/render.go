package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/heidihowilson/skillscope/internal/scan"
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

	sk := m.SelectedSkill()

	var content string
	switch m.PreviewMode {
	case ui.PreviewModal:
		content = m.renderModalPreview(sk, m.Width, contentH)
	case ui.PreviewSide:
		content = m.renderSidePreview(sk, m.Width, contentH)
	default:
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

// padBlock returns `s` with every line right-padded to exactly `width`
// visible cells AND total line count brought up to `height`. This is the
// normalization the modal overlay needs from its background — `ansi.Cut`
// can't construct a "left strip" if the row is shorter than the cut
// position, which manifested as modal rows drifting back to column 0
// over short matrix rows.
func padBlock(s string, width, height int) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		w := lipgloss.Width(l)
		if w < width {
			lines[i] = l + strings.Repeat(" ", width-w)
		}
	}
	blank := strings.Repeat(" ", width)
	for len(lines) < height {
		lines = append(lines, blank)
	}
	return strings.Join(lines, "\n")
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

	if m.PreviewMode == ui.PreviewModal {
		hints = []ui.ActionHint{
			{Key: "j/k", Label: "scroll"},
			{Key: "r", Label: m.previewRenderHint()},
			{Key: "Esc/Space", Label: "close"},
			{Key: "e", Label: "edit"},
		}
		return ui.RenderToolbar(hints, m.Width)
	}

	if m.PreviewMode == ui.PreviewSide {
		if m.Focus == FocusPreview {
			hints = []ui.ActionHint{
				{Key: "j/k", Label: "scroll"},
				{Key: "r", Label: m.previewRenderHint()},
				{Key: "Tab", Label: "back to skills"},
				{Key: "P", Label: "close"},
			}
		} else {
			hints = []ui.ActionHint{
				{Key: "j/k", Label: "row"},
				{Key: "Tab", Label: "focus preview"},
				{Key: "r", Label: m.previewRenderHint()},
				{Key: "P", Label: "close"},
				{Key: "c", Label: "copy"},
				{Key: "m", Label: "move"},
			}
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
		{Key: "Space", Label: "preview"},
		{Key: "P", Label: "side panel"},
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

// previewLines returns the body lines to show and whether they're ready.
// Raw mode is always ready (it's just a string split). Rendered mode is
// ready only when the glamour goroutine has finished and stored the
// result in m.Preview.
func (m *Model) previewLines(sk *scan.SkillRecord) (lines []string, ready bool) {
	if sk == nil {
		return nil, true
	}
	if !m.PreviewRendered {
		return ui.RawLines(sk), true
	}
	lines, ok := m.Preview.Get(sk, m.previewWidth())
	return lines, ok
}

// previewRenderHint is the dynamic label for the R toolbar key — it
// tells the user exactly what mode R will switch them into next, which
// doubles as a "you are currently in X mode" indicator.
func (m *Model) previewRenderHint() string {
	if m.PreviewRendered {
		return "→ raw"
	}
	return "→ rendered"
}

// renderModalPreview overlays a floating modal box on top of the
// underlying view. The view is still visible at the screen edges so it
// feels like a true modal, not a takeover. All glamour work has already
// happened in a goroutine — this function only reads from cache.
func (m *Model) renderModalPreview(sk *scan.SkillRecord, screenW, screenH int) string {
	bg := m.renderMainView(screenW, screenH)
	bg = padBlock(bg, screenW, screenH)

	// Modal dimensions: roomy enough to read but with breathing room at
	// the edges so the underlying view shows through.
	modalW := (screenW * 80) / 100
	modalH := (screenH*80)/100 + 1
	if modalW < 40 {
		modalW = screenW - 4
	}
	if modalH < 10 {
		modalH = screenH - 2
	}

	// Inner content area = modal minus border (1 each side) and padding (1 each side).
	innerW := modalW - 4
	innerH := modalH - 2

	header := ui.RenderHeader(sk, innerW)
	bodyH := innerH - lipgloss.Height(header)
	if bodyH < 1 {
		bodyH = 1
	}

	lines, ready := m.previewLines(sk)
	var body string
	if ready {
		m.PreviewScroll = ui.ClampScroll(lines, m.PreviewScroll, bodyH)
		body = ui.Window(lines, m.PreviewScroll, bodyH, innerW)

		if indicator := ui.RenderScrollIndicator(m.PreviewScroll, bodyH, len(lines)); indicator != "" {
			header = overlayRight(header, indicator, innerW)
		}
	} else {
		body = ui.Placeholder(innerW, bodyH)
	}

	inner := lipgloss.JoinVertical(lipgloss.Left, header, body)

	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorAccent).
		Padding(0, 1).
		Width(innerW).
		Render(inner)

	// Place the modal centered over the background.
	x := (screenW - lipgloss.Width(modal)) / 2
	y := (screenH - lipgloss.Height(modal)) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	return overlay(bg, modal, x, y, screenW)
}

// renderSidePreview shows the active view on the left and a scrollable
// preview pane on the right. Cache-only: an async render is in flight if
// the cache is cold.
func (m *Model) renderSidePreview(sk *scan.SkillRecord, width, height int) string {
	mainW := (width * 60) / 100
	prevW := width - mainW - 1
	if prevW < 20 {
		prevW = 20
		mainW = width - prevW - 1
	}

	mainContent := m.renderMainView(mainW, height)

	var prevContent string
	if sk == nil {
		prevContent = ui.DimStyle.Width(prevW).Height(height).Render("\n  no skill selected")
	} else {
		header := ui.RenderHeader(sk, prevW)
		bodyH := height - lipgloss.Height(header)
		if bodyH < 1 {
			bodyH = 1
		}

		lines, ready := m.previewLines(sk)
		var body string
		if ready {
			m.PreviewScroll = ui.ClampScroll(lines, m.PreviewScroll, bodyH)
			body = ui.Window(lines, m.PreviewScroll, bodyH, prevW)

			if indicator := ui.RenderScrollIndicator(m.PreviewScroll, bodyH, len(lines)); indicator != "" {
				header = overlayRight(header, indicator, prevW)
			}
		} else {
			body = ui.Placeholder(prevW, bodyH)
		}

		prevContent = lipgloss.JoinVertical(lipgloss.Left, header, body)
	}

	focusColor := ui.ColorBorder
	if m.Focus == FocusPreview {
		focusColor = ui.ColorAccent
	}
	divider := lipgloss.NewStyle().
		Foreground(focusColor).
		Render(strings.Repeat("│\n", height-1) + "│")

	return lipgloss.JoinHorizontal(lipgloss.Top, mainContent, divider, prevContent)
}

// overlay places the multi-line `top` block on top of the multi-line
// `bg` block, starting at cell (x, y). Both blocks may contain ANSI
// escape sequences — cell positions are computed in visible cells,
// not byte offsets, via charmbracelet/x/ansi.
//
// This is what gives the modal preview its "floating window" feel:
// the underlying view stays rendered at the edges, the modal sits on
// top in the middle.
func overlay(bg, top string, x, y, screenW int) string {
	bgLines := strings.Split(bg, "\n")
	topLines := strings.Split(top, "\n")

	for i, topLine := range topLines {
		row := y + i
		if row < 0 || row >= len(bgLines) {
			continue
		}
		bgRow := bgLines[row]
		topW := lipgloss.Width(topLine)

		left := ansi.Cut(bgRow, 0, x)
		// The right portion starts after the top line ends.
		right := ansi.Cut(bgRow, x+topW, screenW)

		// Reset ANSI state at each boundary so the bg's escape codes
		// don't bleed into the top line and vice versa.
		bgLines[row] = left + ansiReset + topLine + ansiReset + right
	}
	return strings.Join(bgLines, "\n")
}

const ansiReset = "\x1b[0m"

// overlayRight prints `tag` at the right edge of `block`'s first line
// without changing the block's total width.
func overlayRight(block, tag string, width int) string {
	tagW := lipgloss.Width(tag)
	if tagW == 0 || tagW >= width {
		return block
	}
	lines := strings.Split(block, "\n")
	if len(lines) == 0 {
		return block
	}
	first := lines[0]
	firstW := lipgloss.Width(first)
	overlap := firstW + tagW - width
	if overlap > 0 {
		// Trim plain spaces off the end of the first line to make room
		// for the tag. lipgloss.Width handles ANSI safely; just operate
		// on the underlying string for the tail.
		trim := overlap
		for trim > 0 && len(first) > 0 && first[len(first)-1] == ' ' {
			first = first[:len(first)-1]
			trim--
		}
	}
	pad := width - lipgloss.Width(first) - tagW
	if pad < 0 {
		pad = 0
	}
	lines[0] = first + strings.Repeat(" ", pad) + tag
	return strings.Join(lines, "\n")
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
		"  Space           toggle modal preview (floating, j/k scrolls)",
		"  P               toggle side-panel preview (Tab focuses, j/k scrolls)",
		"  r               in preview: toggle raw / rendered markdown",
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
