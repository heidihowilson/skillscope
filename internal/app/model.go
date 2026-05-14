package app

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sethgho/skillscope/internal/harness"
	"github.com/sethgho/skillscope/internal/scan"
	"github.com/sethgho/skillscope/internal/ui"
)

// View is the plugin interface for a visualization mode.
type View interface {
	ID() string
	Name() string
	KeyHint() string
	Init(m *Model) tea.Cmd
	Update(m *Model, msg tea.Msg) (View, tea.Cmd)
	Render(m *Model, width, height int) string
}

// ActionHinter is an optional interface views can implement to populate
// the contextual toolbar at the bottom of the screen. Views that don't
// implement it get the default keymap hints.
type ActionHinter interface {
	Hints(m *Model) []ui.ActionHint
}

var viewRegistry []View

// RegisterView adds a view to the global registry. Call from init().
func RegisterView(v View) {
	viewRegistry = append(viewRegistry, v)
}

// AllViews returns registered views in registration order.
func AllViews() []View {
	return viewRegistry
}

// OpMode describes the current overlay state for operations.
type OpMode int

const (
	OpNone        OpMode = iota
	OpCopyPicker
	OpMovePicker
	OpDeleteConfirm
)

// FocusPanel identifies which panel has keyboard focus.
type FocusPanel int

const (
	FocusMain    FocusPanel = iota
	FocusPreview
)

// ScanDoneMsg carries scan results back to the event loop.
type ScanDoneMsg struct {
	Skills []scan.SkillRecord
	Err    error
}

// Model is the single source of truth for all app state.
type Model struct {
	// Infrastructure
	Harnesses []harness.Harness
	Views     []View
	HCtx      harness.Context

	// Scan state
	Skills  []scan.SkillRecord
	Loading bool
	LoadErr error

	// Filters
	SearchQuery   string
	SearchActive  bool
	HarnessFilter map[string]bool // nil/empty = all
	ScopeFilter   harness.ScopeKind
	ScopeFilterOn bool
	ShadowedOnly  bool

	// Navigation
	ActiveView  int
	Cursor      int // index in FilteredSkills()
	Focus       FocusPanel
	PreviewMode ui.PreviewMode

	// Operation overlay
	OpMode     OpMode
	OpSkill    *scan.SkillRecord
	OpScopes   []harness.Scope
	OpCursor   int

	// Terminal size
	Width, Height int

	// Overlays
	ShowHelp    bool
	SearchInput textinput.Model

	// Status message (transient)
	StatusMsg string
}

// NewModel constructs a Model with defaults.
func NewModel(ctx harness.Context) *Model {
	ti := textinput.New()
	ti.Placeholder = "fuzzy search…"
	ti.CharLimit = 120

	return &Model{
		HCtx:        ctx,
		ScopeFilter: harness.User, // not used unless ScopeFilterOn
		SearchInput: ti,
	}
}

// FilteredSkills returns skills after applying all active filters.
func (m *Model) FilteredSkills() []scan.SkillRecord {
	var out []scan.SkillRecord
	for _, s := range m.Skills {
		if len(m.HarnessFilter) > 0 && !m.HarnessFilter[s.Scope.Harness] {
			continue
		}
		if m.ScopeFilterOn && s.Scope.Kind != m.ScopeFilter {
			continue
		}
		if m.SearchQuery != "" && !fuzzyMatch(m.SearchQuery, s.Name+" "+s.Description) {
			continue
		}
		if m.ShadowedOnly && !m.IsShadowed(s) {
			continue
		}
		out = append(out, s)
	}
	return out
}

// SelectedSkill returns the currently highlighted skill, or nil.
func (m *Model) SelectedSkill() *scan.SkillRecord {
	filtered := m.FilteredSkills()
	if m.Cursor < 0 || m.Cursor >= len(filtered) {
		return nil
	}
	sk := filtered[m.Cursor]
	return &sk
}

// IsShadowed returns true if another record with the same name exists at
// higher precedence in the same harness.
func (m *Model) IsShadowed(r scan.SkillRecord) bool {
	// Higher precedence: User > Project > ProjectLocal > Plugin
	precedence := func(k harness.ScopeKind) int {
		switch k {
		case harness.User:
			return 3
		case harness.Project:
			return 2
		case harness.ProjectLocal:
			return 1
		}
		return 0
	}
	for _, s := range m.Skills {
		if s.Path == r.Path {
			continue
		}
		if s.Name == r.Name && s.Scope.Harness == r.Scope.Harness &&
			precedence(s.Scope.Kind) > precedence(r.Scope.Kind) {
			return true
		}
	}
	return false
}

// AllScopes returns all resolved scopes across all harnesses.
func (m *Model) AllScopes() []harness.Scope {
	var scopes []harness.Scope
	seen := map[string]bool{}
	for _, h := range m.Harnesses {
		for _, s := range h.Scopes(m.HCtx) {
			if !seen[s.Path] {
				seen[s.Path] = true
				scopes = append(scopes, s)
			}
		}
	}
	return scopes
}

// ClampCursor ensures the cursor stays in bounds.
func (m *Model) ClampCursor() {
	n := len(m.FilteredSkills())
	if n == 0 {
		m.Cursor = 0
		return
	}
	if m.Cursor >= n {
		m.Cursor = n - 1
	}
	if m.Cursor < 0 {
		m.Cursor = 0
	}
}

// fuzzyMatch is a simple substring / character-sequence matcher.
func fuzzyMatch(query, text string) bool {
	if query == "" {
		return true
	}
	// Case-insensitive sequential character match.
	qi := 0
	for _, c := range text {
		if qi < len(query) && (c == rune(query[qi]) || c == rune(query[qi]-32) || c == rune(query[qi]+32)) {
			qi++
		}
	}
	return qi == len(query)
}
