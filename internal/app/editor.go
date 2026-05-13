package app

import (
	"os"
	"os/exec"
)

func editorCmd(path string) *exec.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}
	return exec.Command(editor, path)
}
