package app

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/heidihowilson/skillscope/internal/harness"
	"github.com/heidihowilson/skillscope/internal/ops"
	"github.com/heidihowilson/skillscope/internal/scan"
	"github.com/heidihowilson/skillscope/internal/ui"
)

// isScrollingPreview reports whether j/k should scroll the preview body
// instead of moving the active view's cursor.
//
//	Modal: always — preview owns the screen.
//	Side + Focus on preview pane: yes.
//	Otherwise: no — j/k navigates skills.
func (m *Model) isScrollingPreview() bool {
	if m.PreviewMode == ui.PreviewModal {
		return true
	}
	if m.PreviewMode == ui.PreviewSide && m.Focus == FocusPreview {
		return true
	}
	return false
}

// previewWidth is the width passed to glamour when rendering the preview
// for the current mode. Has to match what render.go uses to display it.
func (m *Model) previewWidth() int {
	switch m.PreviewMode {
	case ui.PreviewModal:
		// Floating modal: ~80% of screen, minus border + padding.
		w := (m.Width*80)/100 - 4
		if w < 20 {
			w = 20
		}
		return w
	case ui.PreviewSide:
		mainW := (m.Width * 60) / 100
		w := m.Width - mainW - 1
		if w < 20 {
			w = 20
		}
		return w
	}
	return m.Width
}

// ensurePreviewRendered returns a tea.Cmd to render the current preview
// in a goroutine when (and only when) it's open and the cache doesn't
// already have the result. Returns nil if no work is needed.
//
// The glamour pass is too slow to do inline — calling it from Update or
// View blocks the input loop and produces visible lag on every keystroke.
// Bubble Tea's pattern: dispatch a Cmd, get a Msg back, store the result.
func (m *Model) ensurePreviewRendered() tea.Cmd {
	if m.PreviewMode == ui.PreviewOff {
		return nil
	}
	sk := m.SelectedSkill()
	if sk == nil {
		return nil
	}
	width := m.previewWidth()
	if _, ok := m.Preview.Get(sk, width); ok {
		return nil
	}
	skCopy := *sk
	return func() tea.Msg {
		return PreviewRenderedMsg{
			Key:   ui.Key(&skCopy, width),
			Lines: ui.Render(&skCopy, width),
		}
	}
}

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
		// Resize changes preview width — kick a re-render if needed.
		return m, m.ensurePreviewRendered()

	case ScanDoneMsg:
		m.Loading = false
		m.LoadErr = msg.Err
		m.Skills = msg.Skills
		m.Cursor = 0
		return m, m.ensurePreviewRendered()

	case PreviewRenderedMsg:
		// A goroutine finished rendering markdown — store it. View() will
		// pick it up on the next frame.
		m.Preview.Set(msg.Key, msg.Lines)
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
		if m.PreviewMode != ui.PreviewOff {
			m.PreviewMode = ui.PreviewOff
			m.PreviewScroll = 0
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
		// Only meaningful when the side panel is open. Modal preview
		// already owns the input; off has nothing to switch to.
		if m.PreviewMode == ui.PreviewSide {
			if m.Focus == FocusMain {
				m.Focus = FocusPreview
			} else {
				m.Focus = FocusMain
			}
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

	case " ", "space":
		// Toggle modal preview.
		if m.PreviewMode == ui.PreviewModal {
			m.PreviewMode = ui.PreviewOff
			m.PreviewScroll = 0
			return m, nil
		}
		m.PreviewMode = ui.PreviewModal
		m.PreviewScroll = 0
		m.Focus = FocusMain
		return m, m.ensurePreviewRendered()

	case "P":
		// Toggle side panel.
		if m.PreviewMode == ui.PreviewSide {
			m.PreviewMode = ui.PreviewOff
			m.PreviewScroll = 0
			m.Focus = FocusMain
			return m, nil
		}
		m.PreviewMode = ui.PreviewSide
		m.PreviewScroll = 0
		return m, m.ensurePreviewRendered()

	case "g":
		m.ShadowedOnly = !m.ShadowedOnly
		m.ClampCursor()
		return m, nil

	case "r":
		// Inside a preview, r toggles raw <-> rendered. Otherwise no-op
		// (uppercase R handles re-scan, which is the action shells expect
		// for "reload" anyway).
		if m.PreviewMode != ui.PreviewOff {
			m.PreviewRendered = !m.PreviewRendered
			m.PreviewScroll = 0
			// Rendered cache should already be warm — kicked off when the
			// preview opened. ensure covers the case where the user toggles
			// before the goroutine finished.
			return m, m.ensurePreviewRendered()
		}
		return m, nil

	case "R":
		// Re-scan the filesystem (works in any mode).
		m.Loading = true
		m.StatusMsg = ""
		return m, m.Init()

	// Navigation. j/k normally moves the cursor in the active view; in
	// modal preview mode (and in side mode when the preview pane is
	// focused) it scrolls the preview body instead.
	case "up", "k":
		if m.isScrollingPreview() {
			m.PreviewScroll--
			return m, nil
		}
		if len(m.Views) > 0 && m.ActiveView < len(m.Views) {
			m.Views[m.ActiveView].Navigate(m, -1)
		}
		// Selected skill probably changed → start preview at the top
		// and (if preview is open) kick a render for the new selection.
		m.PreviewScroll = 0
		return m, m.ensurePreviewRendered()

	case "down", "j":
		if m.isScrollingPreview() {
			m.PreviewScroll++
			return m, nil
		}
		if len(m.Views) > 0 && m.ActiveView < len(m.Views) {
			m.Views[m.ActiveView].Navigate(m, +1)
		}
		m.PreviewScroll = 0
		return m, m.ensurePreviewRendered()

	case "pgup":
		if m.isScrollingPreview() {
			m.PreviewScroll -= 10
			return m, nil
		}
		return m, nil

	case "pgdown", "pgdn":
		if m.isScrollingPreview() {
			m.PreviewScroll += 10
			return m, nil
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
