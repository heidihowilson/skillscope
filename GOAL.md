/goal Build skillscope — a fast, pluggable TUI for cross-scope skill-file inspection

## Objective
Deliver a working POC of `skillscope`, a terminal UI that shows the user, at a
glance, every SKILL.md across every agentic-CLI harness on their machine, broken
down by scope (global user / project committed / project local) and by harness
(Claude Code, Codex CLI, Cursor, OpenCode, Antigravity). It must let the user
preview, filter, and move skills between scopes in 1–2 keystrokes, and it must
ship with multiple visualization modes that I can compare side-by-side. The
architecture must be extensible so new harnesses and new view modes drop in as
pure plugins without touching the core.

## Working directory
The repo already exists and is initialized:
- Path: `/home/wilson/clawd/skillscope`
- Branch: `main` (one commit: `ca9dfd4 initial scaffold`)
- Existing files: `README.md` (placeholder), `.gitignore`
- Do NOT `git init` again or move the directory. Work inside it. Replace the
  placeholder README as part of the build. Commit incrementally with clear
  messages; do not force-push or rewrite the root commit.

## Background — Skill discovery (don't re-derive this)
SKILL.md is a `directory/SKILL.md` unit with YAML frontmatter. The frontmatter
fields you should parse and surface:
- `name` (required — must match parent dir)
- `description` (required — primary trigger text)
- `allowed-tools` (optional, list — Claude Code CLI only)
- `disable-model-invocation` (optional, bool)
- `model` (optional)
- Anything else: preserve and display as "extra" metadata.

Harness scope roots to scan (each entry is one Scope record in the model):
- **Claude Code**
  - user: `~/.claude/skills/` and `~/.config/claude/skills/`
  - project: `<repo>/.claude/skills/`
  - project-local: `<repo>/.claude/skills.local/` (convention — gitignored)
  - plugin (read-only): `~/.claude/plugins/*/skills/`
- **Codex CLI (OpenAI)**
  - user: `~/.codex/skills/`
  - project: `<repo>/.codex/skills/`
- **Cursor**
  - user: `~/.cursor/skills/`
  - project: `<repo>/.cursor/skills/`
- **OpenCode**
  - user: `~/.config/opencode/skills/`
  - project: `<repo>/.opencode/skills/`
- **Google Antigravity**
  - user: `~/.antigravity/skills/`
  - project: `<repo>/.antigravity/skills/`

Each harness implementation must declare its own scope list — don't hardcode the
above in the core. Project root is detected by walking up from `cwd` looking for
`.git`. If no repo, project + project-local scopes are simply empty.

## Stack
- Language: **Go**
- TUI: **Bubble Tea + Bubbles + Lipgloss** (Charmbracelet)
- Markdown rendering: **Glamour**
- YAML: `gopkg.in/yaml.v3`
- Fuzzy search: `sahilm/fuzzy`
- Single static binary; `go install`-able.
- Target Go 1.22+. No CGO.

## Core architecture (extensibility is the point)

Two plugin surfaces. Both registered at init() time via global registries.

### 1. Harness plugin

```go
type Harness interface {
    ID() string         // "claude-code"
    Name() string       // "Claude Code"
    Color() lipgloss.Color
    Scopes(ctx Context) []Scope  // resolves user/project/local/plugin roots
}

type Scope struct {
    Harness    string
    Kind       ScopeKind // User | Project | ProjectLocal | Plugin
    Path       string    // absolute
    ReadOnly   bool      // plugin scopes
}
```

Each harness lives in `internal/harness/<id>/harness.go`, registers itself in
`init()`. Adding a new harness = one file, no edits to core.

### 2. View plugin

```go
type View interface {
    ID() string                // "matrix"
    Name() string              // "Matrix"
    KeyHint() string           // "1"
    Init(m *Model) tea.Cmd
    Update(m *Model, msg tea.Msg) (View, tea.Cmd)
    Render(m *Model, width, height int) string
}
```

Each view lives in `internal/view/<id>/view.go`, registers itself in `init()`.
Adding a new visualization = one file, no edits to core.

## Required visualizations (ship all five)

1. **Matrix** — rows = skills (deduped by name), columns = (harness × scope)
   cells. Cell glyphs: `●` present, `◐` shadowed (also exists at higher
   precedence), `·` absent. Color cells by harness. This is the default view.

2. **Tree** — collapsible: Harness → Scope → Skill. Standard left-pane file-tree
   feel. Right pane shows preview.

3. **Venn** — for a selected harness, an ASCII three-set diagram (User /
   Project / Project-Local) with counts and the currently-highlighted skill's
   membership lit up.

4. **Diff** — for skills present in 2+ scopes within one harness, side-by-side
   frontmatter + body diff with shadowing precedence indicated (which scope
   wins).

5. **Heatmap** — harnesses × scopes grid, each cell shaded by skill count
   (Lipgloss bg ramps). Quick "where's all my stuff" overview.

User cycles views with `1`–`5` or `v` (next) / `V` (prev). Add a sixth slot
labeled "Gallery" that just lists all registered views — proves the registry
works end-to-end.

## Keymap (this is firm — 1–2 keystrokes per the requirement)
Global:
- `q` quit, `?` help overlay, `/` fuzzy search, `Esc` clear filter
- `1`–`9` jump to view by index, `v`/`V` cycle views
- `f` filter by harness (cycles or opens picker), `F` clear harness filter
- `s` filter by scope kind (cycles)
- `Tab` move focus between panels

Skill actions (operate on highlighted skill):
- `p` preview pane toggle (off / raw / rendered) — three-state
- `e` open in `$EDITOR`
- `c` copy to scope… (opens scope picker — one keystroke per target)
- `m` move to scope… (same picker)
- `d` delete (confirms with `y`)
- `y` yank skill path to clipboard
- `g` toggle "show shadowed only"

The copy/move scope picker shows numbered scopes so the full action is
e.g. `c 3` = "copy to scope 3."

## Operations — be safe
- All writes go through a single `internal/ops` package with explicit
  `Copy(skill, dest)`, `Move(skill, dest)`, `Delete(skill)` functions.
- Refuse to write to `ReadOnly` scopes.
- Before delete or overwrite, show a confirmation. No `--force` shortcuts.
- Use `os.Rename` for same-fs moves, fall back to copy+delete across fs.
- Never follow symlinks out of a known scope root.

## Preview pane
- Raw mode: syntax-highlighted YAML frontmatter + raw markdown body
  (use Chroma).
- Rendered mode: Glamour-rendered markdown.
- Header strip always shows: skill name, harness, scope, file path, byte size,
  mtime, and the shadow chain ("also lives in: <scope>, <scope>").

## State model (single source of truth)
One `Model` struct in `internal/app/model.go` holds:
- `harnesses []Harness` (from registry)
- `views []View` (from registry)
- `skills []SkillRecord` (loaded by scanner)
- filters (harness set, scope-kind set, search query, shadowed-only bool)
- focus (which view, which skill index, which pane)
- preview state

`SkillRecord` carries: `Name`, `Description`, `Frontmatter map[string]any`,
`Body string`, `Path`, `Scope` ref, `Hash` (content sha256 for dedup/diff),
`Mtime`.

## Scanner
- Concurrent: one goroutine per scope, fan-in to a channel.
- Each scope walks **one level deep** looking for `*/SKILL.md`.
- Parse frontmatter with a tolerant parser — invalid frontmatter doesn't kill
  the scan, it shows up flagged red with the parse error in the preview.
- Re-scan on `R`. Optional fsnotify-based live refresh behind a feature flag.

## Filtering & search
- `/` opens a single-line fuzzy input; matches `name` + `description`. Live.
- Harness filter `f` is a multi-select (Space to toggle, Enter to apply).
- Filters compose. Visible state always shown in the status bar:
  `harnesses: [claude, cursor]  scope: project  q: "react"  78/214 skills`

## Layout
- Default: left = view-specific main panel; right = preview pane (40%); bottom
  status bar (1 line); top tab bar listing views with active highlighted.
- Resize-aware (`tea.WindowSizeMsg`). Min size 80×24, show a friendly "resize
  your terminal" screen below that.

## Project layout
```
skillscope/
  cmd/skillscope/main.go
  internal/
    app/        # Model, root Bubble Tea program
    harness/    # one subpkg per harness, each self-registers
      claudecode/
      codex/
      cursor/
      opencode/
      antigravity/
    view/       # one subpkg per view, each self-registers
      matrix/
      tree/
      venn/
      diff/
      heatmap/
      gallery/
    scan/       # filesystem walk + frontmatter parsing
    ops/        # copy / move / delete with safety checks
    ui/         # shared lipgloss styles, status bar, preview pane
  testdata/     # fixture skills covering all five harnesses + edge cases
  README.md
  Makefile      # `make`, `make test`, `make demo` (uses testdata)
  go.mod
```

## Acceptance criteria (the bar for "complete")
1. `go install ./cmd/skillscope` builds clean on Linux + macOS from inside
   `/home/wilson/clawd/skillscope`.
2. Running `skillscope` in a repo with the testdata fixtures shows all five
   harnesses, all scopes, every skill, in every view.
3. `make demo` boots the TUI pointed at `testdata/` and looks good on a fresh
   80×24 terminal.
4. Each view renders without panics at sizes 80×24, 120×40, 200×60.
5. Copy/move between scopes works and is reflected after the operation without
   requiring a manual rescan.
6. README documents: install, keymap, how to add a harness (with a 20-line
   stub example), how to add a view (with a 30-line stub example).
7. Tests: scanner parses frontmatter correctly for every fixture; ops package
   refuses writes to read-only scopes; one Bubble Tea golden-render test per
   view at a fixed size (use `teatest`).
8. `skillscope --version` and `skillscope --json` (one-shot machine-readable
   dump, no TUI) both exist — JSON mode is for scripting + future integrations.

## Non-goals (don't do these)
- Editing skills inside the TUI. Open `$EDITOR` instead.
- Creating new skills from scratch. That's a different tool.
- Syncing skills to a remote / cloud / git host.
- Supporting non-skill rule files (CLAUDE.md, AGENTS.md, .cursorrules).
  Those are not skills; out of scope.
- Pretty animations. This is a power-user tool — instant > flashy.

## Deliverables
1. Code committed to the existing repo at `/home/wilson/clawd/skillscope` on
   `main`, building on the `initial scaffold` commit.
2. A 30-second-read README with screenshots (asciinema cast or three static
   pngs of different views) — replaces the current placeholder README.
3. A short `DESIGN.md` (≤ 1 page) explaining the harness + view registries so
   future contributors can extend without spelunking.

Stop. Build it.
