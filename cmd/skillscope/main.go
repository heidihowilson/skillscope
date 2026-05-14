package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime/debug"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/heidihowilson/skillscope/internal/app"
	"github.com/heidihowilson/skillscope/internal/harness"
	"github.com/heidihowilson/skillscope/internal/scan"

	// Register harnesses.
	_ "github.com/heidihowilson/skillscope/internal/harness/antigravity"
	_ "github.com/heidihowilson/skillscope/internal/harness/claudecode"
	_ "github.com/heidihowilson/skillscope/internal/harness/codex"
	_ "github.com/heidihowilson/skillscope/internal/harness/cursor"
	_ "github.com/heidihowilson/skillscope/internal/harness/opencode"

	// Register views (order = key index).
	_ "github.com/heidihowilson/skillscope/internal/view/diff"
	_ "github.com/heidihowilson/skillscope/internal/view/matrix"
	_ "github.com/heidihowilson/skillscope/internal/view/tree"
)

var version = "dev"

func main() {
	var (
		showVersion = flag.Bool("version", false, "print version and exit")
		jsonMode    = flag.Bool("json", false, "dump skills as JSON and exit (no TUI)")
		demoHome    = flag.String("demo-home", "", "use this dir as the simulated home dir (for demos)")
		demoRepo    = flag.String("demo-repo", "", "use this dir as the simulated repo root (for demos)")
	)
	flag.Parse()

	if *showVersion {
		if bi, ok := debug.ReadBuildInfo(); ok {
			fmt.Printf("skillscope %s (%s)\n", version, bi.GoVersion)
		} else {
			fmt.Println("skillscope", version)
		}
		return
	}

	cwd, _ := os.Getwd()
	home, _ := os.UserHomeDir()
	repoRoot := scan.FindRepoRoot(cwd)

	if *demoHome != "" {
		home = *demoHome
	}
	if *demoRepo != "" {
		repoRoot = *demoRepo
		cwd = *demoRepo
	}

	ctx := harness.Context{
		CWD:      cwd,
		RepoRoot: repoRoot,
		HomeDir:  home,
	}

	if *jsonMode {
		s := scan.Scanner{Harnesses: harness.All()}
		skills := s.Scan(ctx)
		if err := jsonDump(skills, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, "json error:", err)
			os.Exit(1)
		}
		return
	}

	m := app.NewModel(ctx)
	m.Harnesses = harness.All()
	m.Views = app.AllViews()

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func jsonDump(skills []scan.SkillRecord, w *os.File) error {
	type record struct {
		Name        string         `json:"name"`
		Description string         `json:"description"`
		Harness     string         `json:"harness"`
		Scope       string         `json:"scope"`
		Path        string         `json:"path"`
		Frontmatter map[string]any `json:"frontmatter,omitempty"`
		ParseError  string         `json:"parse_error,omitempty"`
	}
	var out []record
	for _, s := range skills {
		r := record{
			Name:        s.Name,
			Description: s.Description,
			Harness:     s.Scope.Harness,
			Scope:       s.Scope.Kind.String(),
			Path:        s.Path,
			Frontmatter: s.Frontmatter,
		}
		if s.ParseErr != nil {
			r.ParseError = s.ParseErr.Error()
		}
		out = append(out, r)
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
