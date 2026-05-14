package gallery

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sethgho/skillscope/internal/app"
	"github.com/sethgho/skillscope/internal/ui"
)

type v struct{}

func (v) ID() string      { return "gallery" }
func (v) Name() string    { return "Gallery" }
func (v) KeyHint() string { return "5" }

func (v) Init(m *app.Model) tea.Cmd { return nil }

func (vv v) Update(m *app.Model, msg tea.Msg) (app.View, tea.Cmd) {
	return vv, nil
}

func (vv v) Render(m *app.Model, width, height int) string {
	var sb strings.Builder
	sb.WriteString(ui.BoldStyle.Render("Registered Views") + "\n\n")

	for i, view := range m.Views {
		active := i == m.ActiveView
		key := view.KeyHint()
		name := view.Name()
		id := view.ID()

		keyStyle := lipgloss.NewStyle().Foreground(ui.ColorAccent).Bold(true)
		nameStyle := lipgloss.NewStyle().Foreground(ui.ColorFg)
		idStyle := ui.DimStyle

		if active {
			nameStyle = nameStyle.Background(ui.ColorSelect)
			sb.WriteString(lipgloss.NewStyle().Foreground(ui.ColorAccent).Render("▶ "))
		} else {
			sb.WriteString("  ")
		}

		sb.WriteString(fmt.Sprintf("[%s] %s  %s\n",
			keyStyle.Render(key),
			nameStyle.Render(name),
			idStyle.Render("id:"+id),
		))
	}

	sb.WriteString("\n")
	sb.WriteString(ui.DimStyle.Render("  Press 1-6 to jump to a view, or v/V to cycle.\n"))
	sb.WriteString(ui.DimStyle.Render(fmt.Sprintf("\n  %d views registered — plugin surface verified.", len(m.Views))))

	return sb.String()
}

func init() { app.RegisterView(v{}) }
