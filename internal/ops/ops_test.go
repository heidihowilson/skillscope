package ops_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/heidihowilson/skillscope/internal/harness"
	"github.com/heidihowilson/skillscope/internal/ops"
	"github.com/heidihowilson/skillscope/internal/scan"
)

func TestCopyRefusesReadOnly(t *testing.T) {
	rec := scan.SkillRecord{Name: "test", Path: "/tmp/test/SKILL.md"}
	ro := harness.Scope{ReadOnly: true, Path: "/tmp/ro"}
	if err := ops.Copy(rec, ro); err == nil {
		t.Error("expected error copying to read-only scope, got nil")
	}
}

func TestMoveRefusesReadOnly(t *testing.T) {
	rec := scan.SkillRecord{Name: "test", Path: "/tmp/test/SKILL.md"}
	ro := harness.Scope{ReadOnly: true, Path: "/tmp/ro"}
	if err := ops.Move(rec, ro); err == nil {
		t.Error("expected error moving to read-only scope, got nil")
	}
}

func TestCopyAndDelete(t *testing.T) {
	// Set up source.
	srcDir := t.TempDir()
	skillDir := filepath.Join(srcDir, "my-skill")
	if err := os.Mkdir(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	srcPath := filepath.Join(skillDir, "SKILL.md")
	content := "---\nname: my-skill\ndescription: test\n---\n\nBody.\n"
	if err := os.WriteFile(srcPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// Destination scope.
	destDir := t.TempDir()
	dest := harness.Scope{Path: destDir, ReadOnly: false}
	rec := scan.SkillRecord{Name: "my-skill", Path: srcPath}

	if err := ops.Copy(rec, dest); err != nil {
		t.Fatalf("copy: %v", err)
	}
	destPath := filepath.Join(destDir, "my-skill", "SKILL.md")
	if _, err := os.Stat(destPath); err != nil {
		t.Fatalf("expected dest file to exist: %v", err)
	}

	// Source should still exist after copy.
	if _, err := os.Stat(srcPath); err != nil {
		t.Error("copy removed source file")
	}

	// Delete.
	if err := ops.Delete(rec); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := os.Stat(srcPath); !os.IsNotExist(err) {
		t.Error("delete did not remove source file")
	}
}

func TestMove(t *testing.T) {
	srcDir := t.TempDir()
	skillDir := filepath.Join(srcDir, "mv-skill")
	if err := os.Mkdir(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	srcPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(srcPath, []byte("---\nname: mv-skill\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	destDir := t.TempDir()
	dest := harness.Scope{Path: destDir}
	rec := scan.SkillRecord{Name: "mv-skill", Path: srcPath}

	if err := ops.Move(rec, dest); err != nil {
		t.Fatalf("move: %v", err)
	}

	destPath := filepath.Join(destDir, "mv-skill", "SKILL.md")
	if _, err := os.Stat(destPath); err != nil {
		t.Fatalf("dest not found after move: %v", err)
	}
	if _, err := os.Stat(srcPath); !os.IsNotExist(err) {
		t.Error("source still exists after move")
	}
}
