package tree

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sethgho/skillscope/internal/app"
	"github.com/sethgho/skillscope/internal/ui"
)

type v struct {
	// flat index -> (harness, scope, skill) for cursor tracking
	flatItems []flatItem
}

type flatItemKind int

const (
	itemHarness flatItemKind = iota
	itemScope
	itemSkill
)

type flatItem struct {
	kind    flatItemKind
	label   string
	hid     string
	indent  int
	skillIdx int // index into FilteredSkills()
}

func (v) ID() string      { return "tree" }
func (v) Name() string    { return "Tree" }
func (v) KeyHint() string { return "2" }

func (v) Init(m *app.Model) tea.Cmd { return nil }

func (vv v) Update(m *app.Model, msg tea.Msg) (app.View, tea.Cmd) {
	return vv, nil
}

func (vv v) Render(m *app.Model, width, height int) string {
	skills := m.FilteredSkills()
	if len(skills) == 0 {
		return ui.DimStyle.Width(width).Height(height).Render("  no skills found")
	}

	// Group: harness -> scope path -> []skill
	type scopeKey struct {
		harness string
		path    string
		kind    string
	}
	type node struct {
		sk   scopeKey
		idxs []int
	}

	harnessOrder := []string{}
	hSet := map[string]bool{}
	scopeMap := map[string]map[string][]int{} // harness -> scopePath -> []index

	for i, sk := range skills {
		if !hSet[sk.Scope.Harness] {
			hSet[sk.Scope.Harness] = true
			harnessOrder = append(harnessOrder, sk.Scope.Harness)
		}
		if scopeMap[sk.Scope.Harness] == nil {
			scopeMap[sk.Scope.Harness] = map[string][]int{}
		}
		scopeMap[sk.Scope.Harness][sk.Scope.Path] = append(
			scopeMap[sk.Scope.Harness][sk.Scope.Path], i)
	}
	sort.Strings(harnessOrder)

	var rows []string
	skillFlatIdx := 0 // tracks which flat skill row the cursor matches

	for _, hid := range harnessOrder {
		hLabel := lipgloss.NewStyle().Foreground(ui.HarnessColor(hid)).Bold(true).Render("▼ " + hid)
		rows = append(rows, hLabel)

		scopePaths := make([]string, 0)
		for p := range scopeMap[hid] {
			scopePaths = append(scopePaths, p)
		}
		sort.Strings(scopePaths)

		for _, sp := range scopePaths {
			idxs := scopeMap[hid][sp]
			shortPath := sp
			if len(shortPath) > width-8 {
				shortPath = "…" + shortPath[len(shortPath)-(width-9):]
			}
			rows = append(rows, ui.DimStyle.Render("  ├─ "+shortPath))

			sort.Ints(idxs)
			for i, idx := range idxs {
				sk := skills[idx]
				connector := "│  ├─ "
				if i == len(idxs)-1 {
					connector = "│  └─ "
				}
				name := sk.Name
				if sk.ParseErr != nil {
					name += ui.ErrorStyle.Render(" !")
				}
				line := "  " + connector + name
				if idx == m.Cursor {
					line = ui.SelectedStyle.Render(fmt.Sprintf("%-*s", width, line))
				}
				rows = append(rows, line)
				skillFlatIdx++
			}
		}
	}

	// Scroll to keep cursor visible.
	maxRows := height - 1
	if maxRows < 1 {
		maxRows = 1
	}
	if len(rows) > maxRows {
		// Find the selected row.
		selRow := -1
		skillCount := 0
		for i, r := range rows {
			if strings.Contains(r, ui.SelectedStyle.Render("")) {
				selRow = i
				break
			}
			_ = skillCount
		}
		start := 0
		if selRow >= 0 && selRow >= maxRows {
			start = selRow - maxRows + 1
		}
		end := start + maxRows
		if end > len(rows) {
			end = len(rows)
		}
		rows = rows[start:end]
	}

	return strings.Join(rows, "\n")
}

func init() { app.RegisterView(v{}) }
