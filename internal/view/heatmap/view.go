package heatmap

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

func (v) ID() string      { return "heatmap" }
func (v) Name() string    { return "Heatmap" }
func (v) KeyHint() string { return "5" }

func (v) Init(m *app.Model) tea.Cmd { return nil }

func (vv v) Update(m *app.Model, msg tea.Msg) (app.View, tea.Cmd) {
	return vv, nil
}

// heatColor returns a background color ramp from cold (few) to hot (many).
func heatColor(count, max int) lipgloss.Color {
	if max == 0 {
		return lipgloss.Color("#1A202C")
	}
	ratio := float64(count) / float64(max)
	switch {
	case ratio == 0:
		return lipgloss.Color("#1A202C")
	case ratio < 0.2:
		return lipgloss.Color("#1A365D")
	case ratio < 0.4:
		return lipgloss.Color("#2A4365")
	case ratio < 0.6:
		return lipgloss.Color("#2B6CB0")
	case ratio < 0.8:
		return lipgloss.Color("#3182CE")
	default:
		return lipgloss.Color("#63B3ED")
	}
}

func (vv v) Render(m *app.Model, width, height int) string {
	// Build counts: harness x scope kind
	type key struct {
		harness string
		kind    harness.ScopeKind
	}
	counts := map[key]int{}
	maxCount := 0

	for _, sk := range m.Skills {
		k := key{sk.Scope.Harness, sk.Scope.Kind}
		counts[k]++
		if counts[k] > maxCount {
			maxCount = counts[k]
		}
	}

	// Collect harness IDs in order.
	hids := []string{}
	seen := map[string]bool{}
	for _, h := range m.Harnesses {
		if !seen[h.ID()] {
			seen[h.ID()] = true
			hids = append(hids, h.ID())
		}
	}
	sort.Strings(hids)

	kinds := []harness.ScopeKind{harness.User, harness.Project, harness.ProjectLocal, harness.Plugin}
	kindLabels := []string{"user", "project", "local", "plugin"}

	const colW = 12
	nameW := 14

	var rows []string

	// Header row.
	header := fmt.Sprintf("%-*s", nameW, "harness")
	for _, kl := range kindLabels {
		header += fmt.Sprintf("%-*s", colW, kl)
	}
	rows = append(rows, ui.BoldStyle.Render(header))
	rows = append(rows, ui.DimStyle.Render(strings.Repeat("─", width)))

	for _, hid := range hids {
		name := hid
		if len(name) > nameW-1 {
			name = name[:nameW-1]
		}

		row := fmt.Sprintf("%-*s", nameW, name)
		for _, k := range kinds {
			cnt := counts[key{hid, k}]
			bg := heatColor(cnt, maxCount)
			fg := lipgloss.Color("#E2E8F0")
			if cnt == 0 {
				fg = lipgloss.Color("#4A5568")
			}
			cell := lipgloss.NewStyle().
				Background(bg).
				Foreground(fg).
				Width(colW).
				Render(fmt.Sprintf(" %2d skills", cnt))
			row += cell
		}
		rows = append(rows, row)
	}

	rows = append(rows, "")
	rows = append(rows, ui.DimStyle.Render("  shade intensity = skill count  (darker = fewer)"))

	return strings.Join(rows, "\n")
}

func init() { app.RegisterView(v{}) }
