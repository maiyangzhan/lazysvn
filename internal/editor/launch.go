package editor

import (
	"os"
	"os/exec"
	"strings"

	"github.com/rivo/tview"
)

func Launch(app *tview.Application, path string) error {
	servername := os.Getenv("VIM_SERVERNAME")
	if servername != "" {
		cmd := exec.Command("vim", "--servername", servername, "--remote-wait-silent", path)
		return cmd.Run()
	}

	editorCmd := os.Getenv("EDITOR")
	if editorCmd == "" {
		editorCmd = "vi"
	}

	var cmdErr error
	app.Suspend(func() {
		parts := strings.Fields(editorCmd)
		cmd := exec.Command(parts[0], append(parts[1:], path)...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmdErr = cmd.Run()
	})
	return cmdErr
}

func ForCommit(app *tview.Application) (string, error) {
	f, err := os.CreateTemp("", "lazysvn-commit-*.txt")
	if err != nil {
		return "", err
	}
	tmpPath := f.Name()
	f.Close()
	defer os.Remove(tmpPath)

	if err := Launch(app, tmpPath); err != nil {
		return "", err
	}

	content, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", err
	}
	msg := strings.TrimSpace(string(content))
	return msg, nil
}
