# skillscope — design

skillscope has two plugin surfaces and one core. New harnesses and new
visualizations both drop in as single files; the core never has to learn
about them.

## The two registries

Both live behind package-level globals populated by `init()`:

- `internal/harness` — `harness.Register(h Harness)` / `harness.All()`
- `internal/app`     — `app.RegisterView(v View)` / `app.AllViews()`

A plugin file in `internal/harness/<id>/` or `internal/view/<id>/` calls
its registry from `init()`. The plugin is wired into the binary by a single
blank-import line in `cmd/skillscope/main.go`:

```go
_ "github.com/sethgho/skillscope/internal/harness/<id>"
_ "github.com/sethgho/skillscope/internal/view/<id>"
```

No central switch statement, no factory map, no config file.

## Harness interface

```go
type Harness interface {
    ID() string
    Name() string
    Color() lipgloss.Color
    Scopes(ctx Context) []Scope
}
```

Each harness encodes its own filesystem conventions in `Scopes`. The core
hands every harness a `harness.Context{CWD, RepoRoot, HomeDir}` and gets
back a `[]Scope`. Scope kinds (`User`, `Project`, `ProjectLocal`, `Plugin`)
are core concepts but their *paths* are entirely the harness's call —
Claude Code is the only one that has plugin scopes today; adding new ones
later doesn't touch the core.

## View interface

```go
type View interface {
    ID() string
    Name() string
    KeyHint() string
    Init(m *Model) tea.Cmd
    Update(m *Model, msg tea.Msg) (View, tea.Cmd)
    Render(m *Model, width, height int) string
}
```

Views are passive — they take a `*Model`, ask it what to draw, and return
a string. They don't own state outside their own struct; the global model
is the single source of truth. That lets every view share the cursor,
filters, search query, preview state, etc. without coordinating.

## Model

`internal/app/model.go` holds one `Model` struct that owns everything:

- the registry slices (`Harnesses`, `Views`)
- the scanner's output (`Skills []scan.SkillRecord`)
- filters (harness set, scope kind, query, shadowed-only)
- navigation (`ActiveView`, `Cursor`, `Focus`)
- preview state and op-overlay state

Views call `m.FilteredSkills()` / `m.SelectedSkill()` / `m.IsShadowed()`
to get a consistent view of the world. The `update.go` file owns the
keymap; it dispatches global keys, then delegates the rest to the active
view's `Update`.

## Scanner

`internal/scan` walks one level deep under each `Scope.Path` looking for
`*/SKILL.md`. Each scope walk runs in its own goroutine; results fan in
through `sync.Mutex`-protected slices. Frontmatter parsing is tolerant —
a broken YAML block doesn't kill the scan, it records `ParseErr` on the
record and the preview pane shows the error in red.

## Ops

`internal/ops` is the only package that writes to disk. Every entry point
(`Copy`, `Move`, `Delete`) refuses `ReadOnly` scopes up front. `Move`
prefers `os.Rename` and falls back to copy+delete across filesystems.
Symlinks are stat'd with `Lstat` and refused, so a malicious symlink in a
scope directory can't trick us into writing outside the scope root.

The Bubble Tea event loop re-scans after every successful op, so the UI
reflects the change without a manual `R`.

## Layout

`cmd/skillscope/main.go` is intentionally thin: parse flags, build the
`Context`, instantiate the model, hand it to Bubble Tea. The same flags
support a no-TUI `--json` mode that reuses the scanner directly. That mode
exists so future tooling (CI checks, completion scripts, agentic skills
about skills) doesn't need to scrape a TUI.
