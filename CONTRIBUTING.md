# Contributing to skillscope

Thanks for stopping by. The project is intentionally small and the
plugin surfaces are the whole point — most contributions are one file.

## Dev setup

```sh
git clone https://github.com/heidihowilson/skillscope
cd skillscope
make test
make demo
```

Go 1.22+, no CGO. No other tooling required.

## Adding a harness

A harness is one self-registering Go file in `internal/harness/<id>/`
that implements four methods. The Anthropic Claude Code harness is the
canonical example. Quick sketch:

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

Add a blank-import line in `cmd/skillscope/main.go` and you're done. No
core changes.

When opening the PR:
- Add a fixture under `testdata/home/.myharness/skills/<name>/SKILL.md`
  so the scanner test covers it.
- Use the harness's real brand color when one exists; cite the source
  in a code comment.

## Adding a view

Same pattern but in `internal/view/<id>/`. Implement the `app.View`
interface — `ID`, `Name`, `KeyHint`, `Init`, `Update`, `Render`,
`Navigate`, `Selected`. See `internal/view/matrix/view.go` for a
worked example. Cursor coords are view-relative: each view decides
what "a row" means.

## Code style

- `gofumpt -w .` and `go vet ./...` should pass.
- Tests: scanner edge cases go in `internal/scan`, ops in
  `internal/ops`, view rendering in `internal/view/views_test.go`.
- Prefer editing existing files. New files are fine for new plugins.

## Reporting bugs

Open an issue with the version (`skillscope --version`), Go version,
OS, and the output of `skillscope --json` reproducing the case. For
parser bugs, attach the offending SKILL.md.

## License

By contributing you agree your code will be released under the MIT
license that covers the rest of the repo.
