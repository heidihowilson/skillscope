package app

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/heidihowilson/skillscope/internal/harness"
	"github.com/heidihowilson/skillscope/internal/ops"
	"github.com/heidihowilson/skillscope/internal/scan"
)

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd {
	m.Loading = true
	harnesses := m.Harnesses
	ctx := m.HCtx
	return func() tea.Msg {
		s := scan.Scanner{Harnesses: harnesses}
		skills := s.Scan(ctx)
		return ScanDoneMsg{Skills: skills}
	}
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case ScanDoneMsg:
		m.Loading = false
		m.LoadErr = msg.Err
		m.Skills = msg.Skills
		m.Cursor = 0
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Delegate to active view.
	if len(m.Views) > 0 && m.ActiveView < len(m.Views) {
		v, cmd := m.Views[m.ActiveView].Update(m, msg)
		m.Views[m.ActiveView] = v
		return m, cmd
	}
	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Search input mode.
	if m.SearchActive {
		return m.handleSearchKey(msg)
	}

	// Operation overlays.
	if m.OpMode != OpNone {
		return m.handleOpKey(msg)
	}

	// Help overlay.
	if m.ShowHelp {
		m.ShowHelp = false
		return m, nil
	}

	k := msg.String()

	switch k {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "?":
		m.ShowHelp = true
		return m, nil

	case "/":
		m.SearchActive = true
		m.SearchInput.SetValue("")
		m.SearchInput.Focus()
		return m, textinput.Blink

	case "esc":
		// Priority: preview > search filter > shadowed-only.
		if m.PreviewMode != 0 {
			m.PreviewMode = 0
			m.Focus = FocusMain
			return m, nil
		}
		m.SearchQuery = ""
		m.ShadowedOnly = false
		return m, nil

	case "f":
		return m.cycleHarnessFilter()

	case "F":
		m.HarnessFilter = nil
		m.StatusMsg = "harness filter cleared"
		return m, nil

	case "s":
		return m.cycleScopeFilter()

	case "tab":
		if m.Focus == FocusMain {
			m.Focus = FocusPreview
		} else {
			m.Focus = FocusMain
		}
		return m, nil

	case "v":
		m.setActiveView((m.ActiveView + 1) % len(m.Views))
		return m, nil

	case "V":
		m.setActiveView((m.ActiveView - 1 + len(m.Views)) % len(m.Views))
		return m, nil

	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		idx := int(k[0] - '1')
		if idx < len(m.Views) {
			m.setActiveView(idx)
		}
		return m, nil

	case "p":
		m.PreviewMode = (m.PreviewMode + 1) % 3
		return m, nil

	case "g":
		m.ShadowedOnly = !m.ShadowedOnly
		m.ClampCursor()
		return m, nil

	case "R":
		m.Loading = true
		m.StatusMsg = ""
		return m, m.Init()

	// Navigation — let the active view decide what "row" means.
	case "up", "k":
		if len(m.Views) > 0 && m.ActiveView < len(m.Views) {
			m.Views[m.ActiveView].Navigate(m, -1)
		}
		return m, nil

	case "down", "j":
		if len(m.Views) > 0 && m.ActiveView < len(m.Views) {
			m.Views[m.ActiveView].Navigate(m, +1)
		}
		return m, nil

	case "e":
		return m.openEditor()

	case "y":
		return m.yankPath()

	case "c":
		return m.beginCopy()

	case "m":
		return m.beginMove()

	case "d":
		return m.beginDelete()
	}

	// Delegate unhandled keys to active view.
	if len(m.Views) > 0 && m.ActiveView < len(m.Views) {
		v, cmd := m.Views[m.ActiveView].Update(m, msg)
		m.Views[m.ActiveView] = v
		return m, cmd
	}
	return m, nil
}

func (m *Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.SearchActive = false
		m.SearchQuery = m.SearchInput.Value()
		m.SearchInput.Blur()
		m.ClampCursor()
		return m, nil
	}
	var cmd tea.Cmd
	m.SearchInput, cmd = m.SearchInput.Update(msg)
	m.SearchQuery = m.SearchInput.Value()
	m.ClampCursor()
	return m, cmd
}

func (m *Model) handleOpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	switch m.OpMode {
	case OpDeleteConfirm:
		switch k {
		case "esc", "ctrl+c":
			m.OpMode = OpNone
			m.OpSkill = nil
			m.DeleteInput.Blur()
			m.DeleteMismatch = false
			return m, nil
		case "enter":
			if m.OpSkill == nil {
				m.OpMode = OpNone
				return m, nil
			}
			typed := m.DeleteInput.Value()
			if typed != m.OpSkill.Name {
				m.DeleteMismatch = true
				return m, nil
			}
			if err := ops.Delete(*m.OpSkill); err != nil {
				m.StatusMsg = "delete failed: " + err.Error()
			} else {
				m.StatusMsg = fmt.Sprintf("deleted %s", m.OpSkill.Name)
				m.rescan()
			}
			m.OpMode = OpNone
			m.OpSkill = nil
			m.DeleteInput.Blur()
			m.DeleteMismatch = false
			return m, nil
		}
		// Forward all other keys to the textinput.
		var cmd tea.Cmd
		m.DeleteInput, cmd = m.DeleteInput.Update(msg)
		m.DeleteMismatch = false
		return m, cmd

	case OpCopyPicker, OpMovePicker:
		switch k {
		case "esc", "q":
			m.OpMode = OpNone
			m.OpSkill = nil
		case "up", "k":
			if m.OpCursor > 0 {
				m.OpCursor--
			}
		case "down", "j":
			if m.OpCursor < len(m.OpScopes)-1 {
				m.OpCursor++
			}
		case "enter":
			if m.OpCursor < len(m.OpScopes) && m.OpSkill != nil {
				dest := m.OpScopes[m.OpCursor]
				var err error
				if m.OpMode == OpCopyPicker {
					err = ops.Copy(*m.OpSkill, dest)
				} else {
					err = ops.Move(*m.OpSkill, dest)
				}
				if err != nil {
					m.StatusMsg = "op failed: " + err.Error()
				} else {
					m.StatusMsg = fmt.Sprintf("done → %s", dest.Path)
					m.rescan()
				}
			}
			m.OpMode = OpNone
			m.OpSkill = nil
		default:
			// Number shortcut: 1-9 selects scope.
			if len(k) == 1 && k[0] >= '1' && k[0] <= '9' {
				idx := int(k[0] - '1')
				if idx < len(m.OpScopes) && m.OpSkill != nil {
					dest := m.OpScopes[idx]
					var err error
					if m.OpMode == OpCopyPicker {
						err = ops.Copy(*m.OpSkill, dest)
					} else {
						err = ops.Move(*m.OpSkill, dest)
					}
					if err != nil {
						m.StatusMsg = "op failed: " + err.Error()
					} else {
						m.StatusMsg = fmt.Sprintf("done → %s", dest.Path)
						m.rescan()
					}
					m.OpMode = OpNone
					m.OpSkill = nil
				}
			}
		}
	}
	return m, nil
}

func (m *Model) openEditor() (tea.Model, tea.Cmd) {
	sk := m.SelectedSkill()
	if sk == nil {
		return m, nil
	}
	return m, tea.ExecProcess(editorCmd(sk.Path), func(err error) tea.Msg {
		if err != nil {
			return ScanDoneMsg{Err: err}
		}
		// Re-scan after editing.
		s := scan.Scanner{Harnesses: m.Harnesses}
		return ScanDoneMsg{Skills: s.Scan(m.HCtx)}
	})
}

func (m *Model) yankPath() (tea.Model, tea.Cmd) {
	sk := m.SelectedSkill()
	if sk == nil {
		return m, nil
	}
	m.StatusMsg = "yanked: " + sk.Path
	return m, nil
}

func (m *Model) beginCopy() (tea.Model, tea.Cmd) {
	sk := m.SelectedSkill()
	if sk == nil {
		return m, nil
	}
	m.OpSkill = sk
	m.OpScopes = writableScopes(m.AllScopes())
	m.OpCursor = 0
	m.OpMode = OpCopyPicker
	return m, nil
}

func (m *Model) beginMove() (tea.Model, tea.Cmd) {
	sk := m.SelectedSkill()
	if sk == nil {
		return m, nil
	}
	m.OpSkill = sk
	m.OpScopes = writableScopes(m.AllScopes())
	m.OpCursor = 0
	m.OpMode = OpMovePicker
	return m, nil
}

func (m *Model) beginDelete() (tea.Model, tea.Cmd) {
	sk := m.SelectedSkill()
	if sk == nil {
		return m, nil
	}
	m.OpSkill = sk
	m.OpMode = OpDeleteConfirm
	m.DeleteInput.SetValue("")
	m.DeleteInput.Placeholder = "type " + sk.Name
	m.DeleteInput.Focus()
	m.DeleteMismatch = false
	return m, textinput.Blink
}

func (m *Model) cycleHarnessFilter() (tea.Model, tea.Cmd) {
	ids := make([]string, 0, len(m.Harnesses))
	for _, h := range m.Harnesses {
		ids = append(ids, h.ID())
	}
	if len(ids) == 0 {
		return m, nil
	}
	if len(m.HarnessFilter) == 0 {
		m.HarnessFilter = map[string]bool{ids[0]: true}
	} else {
		// Find current and advance.
		var cur string
		for k := range m.HarnessFilter {
			cur = k
			break
		}
		found := false
		for i, id := range ids {
			if id == cur {
				next := ids[(i+1)%len(ids)]
				if next == ids[0] && i == len(ids)-1 {
					m.HarnessFilter = nil
				} else {
					m.HarnessFilter = map[string]bool{next: true}
				}
				found = true
				break
			}
		}
		if !found {
			m.HarnessFilter = nil
		}
	}
	m.ClampCursor()
	return m, nil
}

func (m *Model) cycleScopeFilter() (tea.Model, tea.Cmd) {
	kinds := []harness.ScopeKind{harness.User, harness.Project, harness.ProjectLocal, harness.Plugin}
	if !m.ScopeFilterOn {
		m.ScopeFilterOn = true
		m.ScopeFilter = kinds[0]
	} else {
		next := -1
		for i, k := range kinds {
			if k == m.ScopeFilter {
				next = i + 1
				break
			}
		}
		if next >= len(kinds) {
			m.ScopeFilterOn = false
		} else {
			m.ScopeFilter = kinds[next]
		}
	}
	m.ClampCursor()
	return m, nil
}

func (m *Model) rescan() {
	s := scan.Scanner{Harnesses: m.Harnesses}
	m.Skills = s.Scan(m.HCtx)
	m.ClampCursor()
}

// setActiveView switches view and resets cursor — cursor coords are
// view-relative.
func (m *Model) setActiveView(idx int) {
	if idx == m.ActiveView {
		return
	}
	m.ActiveView = idx
	m.Cursor = 0
}

func writableScopes(scopes []harness.Scope) []harness.Scope {
	var out []harness.Scope
	for _, s := range scopes {
		if !s.ReadOnly {
			out = append(out, s)
		}
	}
	return out
}
