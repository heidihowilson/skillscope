package app

import (
	"sort"
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/heidihowilson/skillscope/internal/harness"
	"github.com/heidihowilson/skillscope/internal/scan"
	"github.com/heidihowilson/skillscope/internal/ui"
)

// View is the plugin interface for a visualization mode. Each view owns
// its cursor semantics — Navigate moves through whatever the view
// considers "rows," and Selected returns the record that maps to the
// current cursor position.
type View interface {
	ID() string
	Name() string
	KeyHint() string
	Init(m *Model) tea.Cmd
	Update(m *Model, msg tea.Msg) (View, tea.Cmd)
	Render(m *Model, width, height int) string

	// Navigate moves the model's cursor by dir (+1 down, -1 up). The view
	// is responsible for clamping and for translating the cursor index
	// into a meaningful row in its own display order.
	Navigate(m *Model, dir int)

	// Selected returns the SkillRecord that should be acted on (preview,
	// copy, move, delete) for the current cursor position. Returning nil
	// means "no selection."
	Selected(m *Model) *scan.SkillRecord
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

// AllViews returns registered views sorted by their KeyHint (so "1" comes
// before "2" regardless of init() order — goimports likes to alphabetize
// blank imports, which used to scramble the tab bar).
func AllViews() []View {
	sort.SliceStable(viewRegistry, func(i, j int) bool {
		a, _ := strconv.Atoi(viewRegistry[i].KeyHint())
		b, _ := strconv.Atoi(viewRegistry[j].KeyHint())
		return a < b
	})
	return viewRegistry
}

// OpMode describes the current overlay state for operations.
type OpMode int

const (
	OpNone OpMode = iota
	OpCopyPicker
	OpMovePicker
	OpDeleteConfirm
)

// FocusPanel identifies which panel has keyboard focus.
type FocusPanel int

const (
	FocusMain FocusPanel = iota
	FocusPreview
)

// ScanDoneMsg carries scan results back to the event loop.
type ScanDoneMsg struct {
	Skills []scan.SkillRecord
	Err    error
}

// PreviewRenderedMsg carries a finished glamour render back to the event
// loop. Dispatched by a tea.Cmd running in a goroutine — never block
// View() or Update() on the actual render.
type PreviewRenderedMsg struct {
	Key   string
	Lines []string
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

	// Preview state. PreviewScroll is the current vertical scroll offset
	// within the preview body. PreviewRendered toggles between raw file
	// content (default — fast, no glamour pass) and the glamour-rendered
	// markdown (R toggles). Preview caches the expensive rendered output.
	PreviewScroll   int
	PreviewRendered bool
	Preview         ui.Preview

	// Operation overlay
	OpMode   OpMode
	OpSkill  *scan.SkillRecord
	OpScopes []harness.Scope
	OpCursor int

	// Terminal size
	Width, Height int

	// Overlays
	ShowHelp       bool
	SearchInput    textinput.Model
	DeleteInput    textinput.Model // type the skill name to confirm delete
	DeleteMismatch bool            // last attempt didn't match

	// Status message (transient)
	StatusMsg string
}

// NewModel constructs a Model with defaults.
func NewModel(ctx harness.Context) *Model {
	ti := textinput.New()
	ti.Placeholder = "fuzzy search…"
	ti.CharLimit = 120

	di := textinput.New()
	di.Placeholder = "type skill name…"
	di.CharLimit = 200

	return &Model{
		HCtx:        ctx,
		ScopeFilter: harness.User, // not used unless ScopeFilterOn
		SearchInput: ti,
		DeleteInput: di,
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

// SelectedSkill returns the currently highlighted skill, or nil. It
// delegates to the active view so each view can interpret the cursor in
// its own coordinate system.
func (m *Model) SelectedSkill() *scan.SkillRecord {
	if len(m.Views) == 0 || m.ActiveView < 0 || m.ActiveView >= len(m.Views) {
		return nil
	}
	return m.Views[m.ActiveView].Selected(m)
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

// ClampCursor asks the active view to re-clamp (via a no-op Navigate).
func (m *Model) ClampCursor() {
	if len(m.Views) == 0 || m.ActiveView < 0 || m.ActiveView >= len(m.Views) {
		m.Cursor = 0
		return
	}
	m.Views[m.ActiveView].Navigate(m, 0)
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
