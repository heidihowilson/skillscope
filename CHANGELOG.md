# Changelog

All notable changes to this project will be documented in this file.

The format loosely follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and versioning follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] — 2026-05-15

### Added
- Floating preview modal with async Glamour rendering and a raw-first toggle (#40).
- Distribution: curl-pipe `install.sh`, Homebrew tap formula, Scoop manifest, and WinGet manifest set (#34, #35, #36, #37).

### Fixed
- Claude Code plugin scope now scans the real marketplace layout
  (`~/.claude/plugins/marketplaces/<mkt>/{plugins,external_plugins}/<plugin>/skills`),
  so plugin-installed skills actually show up (#41).
- `install.sh` picks a PATH-reachable install dir on macOS (#39).
- ASCII demo readable in light-mode README (#38).

## [0.1.0] — 2026-05-14

Initial public release.

### Added
- Plugin registry for harnesses (`internal/harness`) and views (`internal/app`),
  both populated at `init()` time.
- Five harness implementations: Claude Code, Codex CLI, Cursor, OpenCode,
  Antigravity. Each is one self-registering file using the harness's real
  brand color where one exists.
- Three views: Matrix (default, scope columns × harness-colored dots),
  Tree (scope-rooted with harness dots per leaf), Diff (master/detail with
  LCS-aligned side-by-side comparison).
- Concurrent scanner with tolerant YAML frontmatter parsing — broken
  frontmatter is recorded as a `ParseErr` on the record rather than
  killing the scan.
- Operations: `Copy` / `Move` / `Delete` that refuse `ReadOnly` scopes
  and refuse to follow symlinks. Delete demands typing the exact skill
  name to confirm.
- Preview pane with three-state cycle (off / raw / rendered).
- Contextual toolbar that adapts to the active overlay; `?` always
  appears as a help hint.
- `--version` and `--json` one-shot flags. `--demo-home` / `--demo-repo`
  scaffolding for screenshots.
