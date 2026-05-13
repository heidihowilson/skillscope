package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// StatusBar renders the bottom status bar.
type StatusBar struct {
	HarnessFilter []string
	ScopeFilter   string
	Query         string
	Visible       int
	Total         int
	ActiveView    string
	Width         int
	Msg           string // transient operation result
}

func (s StatusBar) Render() string {
	var parts []string

	if len(s.HarnessFilter) > 0 {
		parts = append(parts, fmt.Sprintf("harnesses:[%s]", strings.Join(s.HarnessFilter, ",")))
	}
	if s.ScopeFilter != "" {
		parts = append(parts, fmt.Sprintf("scope:%s", s.ScopeFilter))
	}
	if s.Query != "" {
		parts = append(parts, fmt.Sprintf("q:%q", s.Query))
	}

	counter := fmt.Sprintf("%d/%d skills", s.Visible, s.Total)

	var msg string
	if s.Msg != "" {
		msg = ErrorStyle.Render(s.Msg)
	}

	left := strings.Join(parts, "  ")
	right := counter
	if msg != "" {
		right = msg + "  " + right
	}

	gap := s.Width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}
	line := left + strings.Repeat(" ", gap) + right
	return StatusBarStyle.Width(s.Width).Render(line)
}
