package ui

import "github.com/charmbracelet/lipgloss"

var (
	ColorFg      = lipgloss.Color("#E2E8F0")
	ColorFgDim   = lipgloss.Color("#718096")
	ColorBg      = lipgloss.Color("#1A202C")
	ColorBorder  = lipgloss.Color("#4A5568")
	ColorAccent  = lipgloss.Color("#63B3ED")
	ColorPresent = lipgloss.Color("#68D391")
	ColorShadow  = lipgloss.Color("#F6AD55")
	ColorAbsent  = lipgloss.Color("#4A5568")
	ColorError   = lipgloss.Color("#FC8181")
	ColorSelect  = lipgloss.Color("#2D3748")

	TabBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1A202C")).
			Foreground(ColorFgDim).
			PaddingLeft(1).PaddingRight(1)

	TabActiveStyle = lipgloss.NewStyle().
			Background(ColorAccent).
			Foreground(lipgloss.Color("#1A202C")).
			Bold(true).
			PaddingLeft(1).PaddingRight(1)

	StatusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#2D3748")).
			Foreground(ColorFg).
			PaddingLeft(1).PaddingRight(1)

	PanelBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(ColorBorder)

	SelectedStyle = lipgloss.NewStyle().
			Background(ColorSelect).
			Foreground(ColorFg)

	DimStyle = lipgloss.NewStyle().
			Foreground(ColorFgDim)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError)

	BoldStyle = lipgloss.NewStyle().Bold(true)

	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorFgDim).
			PaddingLeft(2)

	TinyTermStyle = lipgloss.NewStyle().
			Foreground(ColorFgDim).
			Padding(1, 2).
			Align(lipgloss.Center)
)

// HarnessColor returns the harness-specific color string for use in lipgloss.
func HarnessColor(hid string) lipgloss.Color {
	switch hid {
	case "claude-code":
		return lipgloss.Color("#CC785C")
	case "codex":
		return lipgloss.Color("#10A37F")
	case "cursor":
		return lipgloss.Color("#1C94F4")
	case "opencode":
		return lipgloss.Color("#A855F7")
	case "antigravity":
		return lipgloss.Color("#F59E0B")
	}
	return lipgloss.Color("#718096")
}
