package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ActionHint is a single keyboard-shortcut suggestion rendered in the toolbar.
type ActionHint struct {
	Key   string
	Label string
}

var (
	toolbarBg = lipgloss.Color("#1A202C")

	toolbarKeyStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Background(toolbarBg).
			Bold(true)

	toolbarLabelStyle = lipgloss.NewStyle().
				Foreground(ColorFg).
				Background(toolbarBg)

	toolbarBaseStyle = lipgloss.NewStyle().
				Background(toolbarBg).
				PaddingLeft(1).PaddingRight(1)
)

// RenderToolbar renders a single-line action-hint bar. Always includes a
// trailing `?` help suggestion regardless of the hints slice.
func RenderToolbar(hints []ActionHint, width int) string {
	// Ensure ? help is always present.
	hasHelp := false
	for _, h := range hints {
		if h.Key == "?" {
			hasHelp = true
			break
		}
	}
	if !hasHelp {
		hints = append(hints, ActionHint{Key: "?", Label: "help"})
	}

	var parts []string
	for _, h := range hints {
		key := toolbarKeyStyle.Render("[" + h.Key + "]")
		label := toolbarLabelStyle.Render(" " + h.Label)
		parts = append(parts, key+label)
	}

	joined := strings.Join(parts, toolbarLabelStyle.Render("  "))

	// Truncate if it overflows width.
	if lipgloss.Width(joined) > width-2 {
		// Drop trailing parts until it fits, but always keep help.
		for len(parts) > 1 {
			parts = parts[:len(parts)-1]
			joined = strings.Join(parts, toolbarLabelStyle.Render("  "))
			helpKey := toolbarKeyStyle.Render("[?]")
			helpLab := toolbarLabelStyle.Render(" help")
			withHelp := joined + toolbarLabelStyle.Render("  ") + helpKey + helpLab
			if lipgloss.Width(withHelp) <= width-2 {
				joined = withHelp
				break
			}
		}
	}

	return toolbarBaseStyle.Width(width).Render(joined)
}
