package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/heidihowilson/skillscope/internal/harness"
)

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

// HarnessColor returns the brand color the harness registered itself with.
// Each harness package is the single source of truth for its own color;
// this helper just looks it up by ID.
func HarnessColor(hid string) lipgloss.Color {
	for _, h := range harness.All() {
		if h.ID() == hid {
			return h.Color()
		}
	}
	return lipgloss.Color("#718096") // fallback gray
}
