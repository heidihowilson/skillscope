package harness

import "github.com/charmbracelet/lipgloss"

// ScopeKind classifies where a skill scope lives.
type ScopeKind int

const (
	User         ScopeKind = iota
	Project
	ProjectLocal
	Plugin
)

func (k ScopeKind) String() string {
	switch k {
	case User:
		return "user"
	case Project:
		return "project"
	case ProjectLocal:
		return "local"
	case Plugin:
		return "plugin"
	}
	return "unknown"
}

// Scope is a single root directory that may contain skills.
type Scope struct {
	Harness  string
	Kind     ScopeKind
	Path     string
	ReadOnly bool
}

// Context carries runtime info harnesses use to resolve scope paths.
type Context struct {
	CWD      string // current working directory
	RepoRoot string // empty when not inside a git repo
	HomeDir  string // os.UserHomeDir()
}

// Harness represents one agentic-CLI tool's skill conventions.
type Harness interface {
	ID() string
	Name() string
	Color() lipgloss.Color
	Scopes(ctx Context) []Scope
}

var registry []Harness

// Register adds a harness to the global registry. Call from init().
func Register(h Harness) {
	registry = append(registry, h)
}

// All returns registered harnesses in registration order.
func All() []Harness {
	return registry
}
