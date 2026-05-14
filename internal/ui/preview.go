package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/heidihowilson/skillscope/internal/scan"
	"gopkg.in/yaml.v3"
)

// PreviewMode controls how the preview pane renders.
type PreviewMode int

const (
	PreviewOff      PreviewMode = iota
	PreviewRaw
	PreviewRendered
)

// RenderPreview renders the preview pane for the given skill record.
func RenderPreview(rec *scan.SkillRecord, mode PreviewMode, width, height int) string {
	if rec == nil || mode == PreviewOff {
		return DimStyle.Width(width).Height(height).Render("  press p to preview")
	}

	header := renderPreviewHeader(rec, width)
	body := renderPreviewBody(rec, mode, width, height-lipgloss.Height(header)-1)

	return lipgloss.JoinVertical(lipgloss.Left, header, body)
}

func renderPreviewHeader(rec *scan.SkillRecord, width int) string {
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

func renderPreviewBody(rec *scan.SkillRecord, mode PreviewMode, width, height int) string {
	if height < 1 {
		height = 1
	}
	style := lipgloss.NewStyle().Width(width).Height(height).MaxHeight(height)

	switch mode {
	case PreviewRaw:
		raw := buildRawContent(rec)
		return style.Render(raw)
	case PreviewRendered:
		full := buildFullMarkdown(rec)
		r, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(width),
		)
		if err != nil {
			return style.Render(full)
		}
		rendered, err := r.Render(full)
		if err != nil {
			return style.Render(full)
		}
		return style.Render(rendered)
	}
	return ""
}

func buildRawContent(rec *scan.SkillRecord) string {
	var sb strings.Builder
	if rec.Frontmatter != nil {
		sb.WriteString("```yaml\n")
		out, _ := yaml.Marshal(rec.Frontmatter)
		sb.Write(out)
		sb.WriteString("```\n\n")
	}
	sb.WriteString(rec.Body)
	return sb.String()
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
