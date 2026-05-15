package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/heidihowilson/skillscope/internal/harness"
	"github.com/heidihowilson/skillscope/internal/ui"
)

func key(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func newKMTestModel() *Model {
	m := NewModel(harness.Context{})
	m.Width = 100
	m.Height = 30
	return m
}

func TestPreviewKeymap_Space_TogglesModal(t *testing.T) {
	m := newKMTestModel()
	if m.PreviewMode != ui.PreviewOff {
		t.Fatalf("initial PreviewMode = %v, want Off", m.PreviewMode)
	}
	m.Update(key(" "))
	if m.PreviewMode != ui.PreviewModal {
		t.Errorf("after first Space: PreviewMode = %v, want Modal", m.PreviewMode)
	}
	m.Update(key(" "))
	if m.PreviewMode != ui.PreviewOff {
		t.Errorf("after second Space: PreviewMode = %v, want Off", m.PreviewMode)
	}
}

func TestPreviewKeymap_LowercaseP_DoesNotTriggerPreview(t *testing.T) {
	m := newKMTestModel()
	m.Update(key("p"))
	if m.PreviewMode != ui.PreviewOff {
		t.Errorf("lowercase p should no longer open preview, got %v", m.PreviewMode)
	}
}

func TestPreviewKeymap_UppercaseP_TogglesSide(t *testing.T) {
	m := newKMTestModel()
	m.Update(key("P"))
	if m.PreviewMode != ui.PreviewSide {
		t.Errorf("after first P: PreviewMode = %v, want Side", m.PreviewMode)
	}
	m.Update(key("P"))
	if m.PreviewMode != ui.PreviewOff {
		t.Errorf("after second P: PreviewMode = %v, want Off", m.PreviewMode)
	}
}

func TestPreviewKeymap_CrossSwitching(t *testing.T) {
	m := newKMTestModel()
	m.Update(key(" "))
	if m.PreviewMode != ui.PreviewModal {
		t.Fatalf("setup: PreviewMode = %v, want Modal", m.PreviewMode)
	}
	m.Update(key("P"))
	if m.PreviewMode != ui.PreviewSide {
		t.Errorf("Modal + P should switch to Side, got %v", m.PreviewMode)
	}
	m.Update(key(" "))
	if m.PreviewMode != ui.PreviewModal {
		t.Errorf("Side + Space should switch to Modal, got %v", m.PreviewMode)
	}
}

func TestPreviewKeymap_EscClosesAnyPreview(t *testing.T) {
	for _, start := range []ui.PreviewMode{ui.PreviewSide, ui.PreviewModal} {
		m := newKMTestModel()
		m.PreviewMode = start
		m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		if m.PreviewMode != ui.PreviewOff {
			t.Errorf("from %v, Esc should close to Off, got %v", start, m.PreviewMode)
		}
	}
}
