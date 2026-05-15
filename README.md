<p align="center">
  <img src="docs/assets/logo-wordmark.svg" alt="skillscope" width="540">
</p>

<p align="center">
  <em>Every <code>SKILL.md</code>, every harness.</em>
</p>

<p align="center">
  <a href="https://github.com/heidihowilson/skillscope/actions/workflows/ci.yml"><img src="https://github.com/heidihowilson/skillscope/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://goreportcard.com/report/github.com/heidihowilson/skillscope"><img src="https://goreportcard.com/badge/github.com/heidihowilson/skillscope" alt="Go Report"></a>
  <a href="https://github.com/heidihowilson/skillscope/releases"><img src="https://img.shields.io/github/v/release/heidihowilson/skillscope" alt="Release"></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/heidihowilson/skillscope" alt="License"></a>
  <a href="https://heidihowilson.github.io/skillscope/"><img src="https://img.shields.io/badge/docs-pages-blue" alt="Docs"></a>
</p>

A fast, pluggable TUI for inspecting `SKILL.md` files across every agentic-CLI
harness on your machine — Claude Code, Codex CLI, Cursor, OpenCode, Antigravity —
broken down by scope (user / project / project-local) and harness, with copy /
move / delete in 1–2 keystrokes.

```
┌ 1 Matrix ┐ 2 Tree   3 Diff
Harnesses:  ● claude-code  ● codex  ● cursor  ● opencode  ● antigravity

skill              user                       project                    local
────────────────────────────────────────────────────────────────────────────────
git-helper         ● ● ●                      ·                          ·
multi-scope-skill  ●                          ◐                          ·
local-skill        ·                          ·                          ●
…
  ● present   ◐ shadowed (higher scope wins)   · absent   ! parse error
[/] search  [f] filter  [p] preview  [c] copy  [m] move  [d] delete  [?] help
```

## Install

**Recommended — curl-pipe installer** (Linux + macOS, no Go required):

```sh
curl -sSL https://heidihowilson.github.io/skillscope/install.sh | sh
```

Pin a version with `SKILLSCOPE_VERSION=v0.1.0 …`. Override the install dir with `SKILLSCOPE_INSTALL_DIR=…`. The script verifies sha256 against the release's `checksums.txt` and refuses to run as root unless `SKILLSCOPE_ACCEPT_ROOT=1` is set.

**Scoop (Windows):**

```powershell
scoop bucket add heidihowilson https://github.com/heidihowilson/scoop-bucket
scoop install skillscope
```

**WinGet (Windows):** manifests are hand-curated in [`winget/`](winget/) pending submission to [`microsoft/winget-pkgs`](https://github.com/microsoft/winget-pkgs). Once accepted upstream, `winget install heidihowilson.skillscope` will work.

**Via `go install`:**

```sh
go install github.com/heidihowilson/skillscope/cmd/skillscope@latest
```

**From source:**

```sh
git clone https://github.com/heidihowilson/skillscope
cd skillscope
go install ./cmd/skillscope
```

Runs on Linux, macOS, and Windows. Go 1.22+. Single static binary, no CGO.

## Quick start

Run it from anywhere:

```sh
skillscope
```

It scans your home dir + the current git repo (if any) and shows every skill
across every harness. From inside this repo:

```sh
make demo
```

boots the TUI pointed at `testdata/`.

## Keymap

### Global

| Key | Action |
| --- | ------ |
| `q` / `Ctrl+C` | quit |
| `?` | help overlay |
| `/` | fuzzy search |
| `Esc` | close preview / clear filter |
| `1`–`9` | jump to view by index |
| `v` / `V` | cycle views forward / back |
| `f` | cycle harness filter |
| `F` | clear harness filter |
| `s` | cycle scope-kind filter |
| `g` | toggle "shadowed only" |
| `Tab` | move focus between panels |
| `R` | re-scan filesystem |

### Skill actions (on highlighted skill)

| Key | Action |
| --- | ------ |
| `p` | preview pane: off → raw → rendered |
| `e` | open in `$EDITOR` |
| `c` | copy to scope (numbered picker, then `1`–`9`) |
| `m` | move to scope (same picker) |
| `d` | delete (confirm with `y`) |
| `y` | yank skill path |

Picker shortcut: e.g. `c 3` = "copy to scope 3."

## Flags

| Flag | Meaning |
| ---- | ------- |
| `--version` | print version and exit |
| `--json` | dump all skills as JSON to stdout and exit |
| `--demo-home <dir>` | use `<dir>` as the simulated home dir |
| `--demo-repo <dir>` | use `<dir>` as the simulated repo root |

## Views

1. **Matrix** — rows = skills, columns = scope (user / project / local /
   plugin). Each cell shows one colored dot per harness present. `●`
   winning, `◐` shadowed by a higher-precedence scope in the same harness,
   `!` parse error, `·` absent.
2. **Tree** — collapsible Harness → Scope → Skill.
3. **Diff** — side-by-side frontmatter + body for skills in multiple scopes
   within one harness.

## Adding a harness

Drop a single file at `internal/harness/<id>/harness.go`. It self-registers
in `init()`; no edits to the core.

```go
package myharness

import (
    "path/filepath"
    "github.com/charmbracelet/lipgloss"
    "github.com/heidihowilson/skillscope/internal/harness"
)

type h struct{}

func (h) ID() string            { return "myharness" }
func (h) Name() string          { return "My Harness" }
func (h) Color() lipgloss.Color { return lipgloss.Color("#FF66CC") }

func (h) Scopes(ctx harness.Context) []harness.Scope {
    scopes := []harness.Scope{{
        Harness: "myharness", Kind: harness.User,
        Path: filepath.Join(ctx.HomeDir, ".myharness", "skills"),
    }}
    if ctx.RepoRoot != "" {
        scopes = append(scopes, harness.Scope{
            Harness: "myharness", Kind: harness.Project,
            Path: filepath.Join(ctx.RepoRoot, ".myharness", "skills"),
        })
    }
    return scopes
}

func init() { harness.Register(h{}) }
```

Then add a blank-import in `cmd/skillscope/main.go`:

```go
_ "github.com/heidihowilson/skillscope/internal/harness/myharness"
```

## Adding a view

Drop a single file at `internal/view/<id>/view.go`. Same `init()`-registration
pattern.

```go
package myview

import (
    "fmt"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/heidihowilson/skillscope/internal/app"
)

type v struct{}

func (v) ID() string                  { return "myview" }
func (v) Name() string                { return "My View" }
func (v) KeyHint() string             { return "7" }
func (v) Init(m *app.Model) tea.Cmd   { return nil }

func (vv v) Update(m *app.Model, msg tea.Msg) (app.View, tea.Cmd) {
    return vv, nil
}

func (vv v) Render(m *app.Model, width, height int) string {
    skills := m.FilteredSkills()
    out := fmt.Sprintf("My View — %d skills\n\n", len(skills))
    for i, sk := range skills {
        marker := "  "
        if i == m.Cursor {
            marker = "▶ "
        }
        out += fmt.Sprintf("%s%s (%s)\n", marker, sk.Name, sk.Scope.Harness)
        if i > height-4 {
            break
        }
    }
    _ = width
    return out
}

func init() { app.RegisterView(v{}) }
```

Blank-import it in `cmd/skillscope/main.go` and it'll appear in the tab bar.

## Non-goals

- Editing skills inside the TUI (use `$EDITOR` with `e`).
- Creating skills from scratch.
- Syncing skills to a remote.
- Supporting non-skill rules files (`CLAUDE.md`, `AGENTS.md`, `.cursorrules`).

## License

MIT.
