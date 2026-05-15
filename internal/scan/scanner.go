package scan

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/heidihowilson/skillscope/internal/harness"
	"gopkg.in/yaml.v3"
)

// SkillRecord is the parsed representation of a single SKILL.md file.
type SkillRecord struct {
	Name        string
	Description string
	Frontmatter map[string]any
	Body        string
	Raw         string // original file contents, byte-exact
	Path        string
	Scope       harness.Scope
	Hash        [32]byte
	Mtime       time.Time
	ParseErr    error
}

// Scanner walks harness scopes and produces SkillRecords.
type Scanner struct {
	Harnesses []harness.Harness
}

// Scan resolves scopes for all harnesses and walks each one concurrently.
func (s *Scanner) Scan(ctx harness.Context) []SkillRecord {
	var (
		mu      sync.Mutex
		results []SkillRecord
		wg      sync.WaitGroup
	)

	for _, h := range s.Harnesses {
		for _, scope := range h.Scopes(ctx) {
			scope := scope
			wg.Add(1)
			go func() {
				defer wg.Done()
				recs := walkScope(scope)
				mu.Lock()
				results = append(results, recs...)
				mu.Unlock()
			}()
		}
	}
	wg.Wait()
	return results
}

// walkScope walks one level deep under scope.Path looking for */SKILL.md.
func walkScope(scope harness.Scope) []SkillRecord {
	entries, err := os.ReadDir(scope.Path)
	if err != nil {
		return nil
	}
	var recs []SkillRecord
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillPath := filepath.Join(scope.Path, e.Name(), "SKILL.md")
		info, err := os.Lstat(skillPath)
		if err != nil {
			continue
		}
		// Never follow symlinks out of scope root.
		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		recs = append(recs, parseSkill(skillPath, scope, info.ModTime()))
	}
	return recs
}

// parseSkill reads and parses a single SKILL.md file.
func parseSkill(path string, scope harness.Scope, mtime time.Time) SkillRecord {
	rec := SkillRecord{Path: path, Scope: scope, Mtime: mtime}

	data, err := os.ReadFile(path)
	if err != nil {
		rec.ParseErr = fmt.Errorf("read: %w", err)
		return rec
	}

	// Compute content hash.
	rec.Hash = sha256.Sum256(data)

	// Extract YAML frontmatter between --- delimiters.
	content := string(data)
	rec.Raw = content
	fm, body, parseErr := parseFrontmatter(content)
	rec.Body = body
	rec.ParseErr = parseErr

	if fm != nil {
		rec.Frontmatter = fm
		if v, ok := fm["name"]; ok {
			rec.Name, _ = v.(string)
		}
		if v, ok := fm["description"]; ok {
			rec.Description, _ = v.(string)
		}
	}

	// Fall back to directory name if name field missing.
	if rec.Name == "" {
		rec.Name = filepath.Base(filepath.Dir(path))
	}

	return rec
}

// parseFrontmatter splits YAML front matter from markdown body.
func parseFrontmatter(content string) (map[string]any, string, error) {
	const delim = "---"
	if !strings.HasPrefix(content, delim) {
		return nil, content, nil
	}
	rest := content[3:]
	// Skip optional newline after opening ---
	rest = strings.TrimPrefix(rest, "\n")

	end := strings.Index(rest, "\n---")
	if end == -1 {
		// Try without leading newline (e.g. "---\n...\n---\n")
		if idx := strings.Index(rest, "---"); idx == 0 {
			return nil, content, nil
		}
		return nil, content, fmt.Errorf("unclosed frontmatter")
	}

	fmRaw := rest[:end]
	body := strings.TrimPrefix(rest[end+4:], "\n")

	var fm map[string]any
	if err := yaml.Unmarshal([]byte(fmRaw), &fm); err != nil {
		return nil, body, fmt.Errorf("yaml: %w", err)
	}
	return fm, body, nil
}

// FindRepoRoot walks up from dir until it finds a .git directory.
func FindRepoRoot(dir string) string {
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// DumpJSON serialises all skills to a simple JSON format for --json mode.
func DumpJSON(recs []SkillRecord, w io.Writer) error {
	fmt.Fprintln(w, "[")
	for i, r := range recs {
		comma := ","
		if i == len(recs)-1 {
			comma = ""
		}
		fmt.Fprintf(w, "  {\"name\":%q,\"description\":%q,\"harness\":%q,\"scope\":%q,\"path\":%q,\"parse_error\":%q}%s\n",
			r.Name, r.Description, r.Scope.Harness, r.Scope.Kind.String(), r.Path, errStr(r.ParseErr), comma)
	}
	fmt.Fprintln(w, "]")
	return nil
}

func errStr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
