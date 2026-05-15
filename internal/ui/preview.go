package ui

import (
	"fmt"
	"strings"
	"sync"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/heidihowilson/skillscope/internal/scan"
	"gopkg.in/yaml.v3"
)

// PreviewMode controls how the preview is shown.
//
//	PreviewOff   — no preview rendered.
//	PreviewSide  — preview opens in the right ~40% of the screen alongside
//	               the active view.
//	PreviewModal — preview opens as a floating overlay centered on top of
//	               the underlying view.
type PreviewMode int

const (
	PreviewOff PreviewMode = iota
	PreviewSide
	PreviewModal
)

// Preview is a thread-safe, key-indexed cache of pre-rendered skill body
// lines. Rendering is expensive (glamour parses markdown synchronously);
// the caller does the render in a goroutine via `Render` and stores the
// result with `Set`. View() should only ever call `Get` — never block on
// `Render`.
type Preview struct {
	mu    sync.Mutex
	cache map[string][]string
}

// Key returns the cache key for (rec, width). Empty when rec is nil.
func Key(rec *scan.SkillRecord, width int) string {
	if rec == nil {
		return ""
	}
	return fmt.Sprintf("%x|%d", rec.Hash, width)
}

// Get returns cached lines for (rec, width). ok=false means the render
// hasn't completed yet — the caller should show a placeholder and the
// state machine should be dispatching a Render goroutine to fill it in.
func (p *Preview) Get(rec *scan.SkillRecord, width int) (lines []string, ok bool) {
	if rec == nil {
		return nil, false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	lines, ok = p.cache[Key(rec, width)]
	return
}

// Set stores rendered lines under `key`.
func (p *Preview) Set(key string, lines []string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cache == nil {
		p.cache = map[string][]string{}
	}
	p.cache[key] = lines
}

// RawLines returns the original file content split on newlines. Cheap —
// no parsing, no rendering. This is what the preview shows by default;
// the glamour-rendered version is only generated when the user asks
// for it (R) and is pre-warmed in the background.
func RawLines(rec *scan.SkillRecord) []string {
	if rec == nil {
		return nil
	}
	return strings.Split(strings.TrimRight(rec.Raw, "\n"), "\n")
}

// Render is the heavyweight glamour pass. Call from a goroutine (via
// tea.Cmd) — never inline from Update or View.
func Render(rec *scan.SkillRecord, width int) []string {
	if rec == nil {
		return nil
	}
	if width < 10 {
		width = 10
	}
	full := buildFullMarkdown(rec)
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return strings.Split(strings.TrimRight(full, "\n"), "\n")
	}
	out, err := r.Render(full)
	if err != nil {
		return strings.Split(strings.TrimRight(full, "\n"), "\n")
	}
	return strings.Split(strings.TrimRight(out, "\n"), "\n")
}

// Window returns the slice of lines visible at scroll offset, padded
// with full-width blank lines up to `height`.
func Window(lines []string, scroll, height, width int) string {
	if height < 1 {
		height = 1
	}
	n := len(lines)
	if n == 0 {
		blank := strings.Repeat(" ", width)
		out := make([]string, height)
		for i := range out {
			out[i] = blank
		}
		return strings.Join(out, "\n")
	}
	if scroll < 0 {
		scroll = 0
	}
	if scroll > n {
		scroll = n
	}
	end := scroll + height
	if end > n {
		end = n
	}
	visible := append([]string{}, lines[scroll:end]...)
	for len(visible) < height {
		visible = append(visible, strings.Repeat(" ", width))
	}
	return strings.Join(visible, "\n")
}

// ClampScroll bounds scroll to [0, max(0, len(lines)-height)].
func ClampScroll(lines []string, scroll, height int) int {
	maxS := len(lines) - height
	if maxS < 0 {
		maxS = 0
	}
	if scroll < 0 {
		scroll = 0
	}
	if scroll > maxS {
		scroll = maxS
	}
	return scroll
}

// RenderHeader returns the always-visible info strip above the preview
// body.
func RenderHeader(rec *scan.SkillRecord, width int) string {
	if rec == nil {
		return DimStyle.Width(width).Render("  no skill selected")
	}
	lines := []string{
		BoldStyle.Render(rec.Name),
		DimStyle.Render(fmt.Sprintf("%s  %s  %s", rec.Scope.Harness, rec.Scope.Kind, rec.Scope.Path)),
		DimStyle.Render(fmt.Sprintf("%s  %d bytes", rec.Mtime.Format("2006-01-02 15:04"), fileSize(rec))),
	}
	if rec.ParseErr != nil {
		lines = append(lines, ErrorStyle.Render("parse error: "+rec.ParseErr.Error()))
	}
	return lipgloss.NewStyle().
		Width(width).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(ColorBorder).
		Render(strings.Join(lines, "\n"))
}

// RenderScrollIndicator returns a "12–34 / 56"-style indicator. Empty when
// the body fits in the available height.
func RenderScrollIndicator(scroll, height, total int) string {
	if total <= height {
		return ""
	}
	first := scroll + 1
	last := scroll + height
	if last > total {
		last = total
	}
	return DimStyle.Render(fmt.Sprintf("↕ %d–%d / %d", first, last, total))
}

// Placeholder is what we show in the body area while a render is in
// flight. Centered "rendering…" so the user sees something is happening
// rather than a frozen empty box.
func Placeholder(width, height int) string {
	if height < 1 {
		height = 1
	}
	style := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center).
		Foreground(ColorFgDim).
		Italic(true)
	return style.Render("rendering…")
}

func buildFullMarkdown(rec *scan.SkillRecord) string {
	if rec.Frontmatter == nil {
		return rec.Body
	}
	out, _ := yaml.Marshal(rec.Frontmatter)
	return "```yaml\n" + string(out) + "```\n\n" + rec.Body
}

func fileSize(rec *scan.SkillRecord) int64 {
	return int64(len(rec.Body))
}
