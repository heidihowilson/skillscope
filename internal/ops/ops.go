package ops

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sethgho/skillscope/internal/harness"
	"github.com/sethgho/skillscope/internal/scan"
)

// Copy copies a skill directory into destScope.
func Copy(rec scan.SkillRecord, destScope harness.Scope) error {
	if destScope.ReadOnly {
		return fmt.Errorf("scope %q is read-only", destScope.Path)
	}
	destDir := filepath.Join(destScope.Path, rec.Name)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	destPath := filepath.Join(destDir, "SKILL.md")
	return copyFile(rec.Path, destPath)
}

// Move moves a skill to destScope. Uses os.Rename on same filesystem,
// falls back to copy+delete across filesystems.
func Move(rec scan.SkillRecord, destScope harness.Scope) error {
	if destScope.ReadOnly {
		return fmt.Errorf("scope %q is read-only", destScope.Path)
	}
	destDir := filepath.Join(destScope.Path, rec.Name)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	destPath := filepath.Join(destDir, "SKILL.md")

	if err := os.Rename(rec.Path, destPath); err == nil {
		_ = os.Remove(filepath.Dir(rec.Path)) // best-effort remove empty dir
		return nil
	}
	// Cross-device: copy then delete.
	if err := copyFile(rec.Path, destPath); err != nil {
		return err
	}
	if err := os.Remove(rec.Path); err != nil {
		return fmt.Errorf("remove original: %w", err)
	}
	_ = os.Remove(filepath.Dir(rec.Path))
	return nil
}

// Delete removes a skill file. Does not touch sibling files.
func Delete(rec scan.SkillRecord) error {
	if err := os.Remove(rec.Path); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	_ = os.Remove(filepath.Dir(rec.Path)) // best-effort remove empty dir
	return nil
}

func copyFile(src, dst string) error {
	// Validate src is a regular file (no symlink chasing).
	srcInfo, err := os.Lstat(src)
	if err != nil {
		return fmt.Errorf("stat src: %w", err)
	}
	if srcInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to copy symlink %s", src)
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
