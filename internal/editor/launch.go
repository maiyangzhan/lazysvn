package editor

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/rivo/tview"

	"github.com/maiyangzhan/lazysvn/internal/logfile"
)

// Launch opens `path` in the user's editor and blocks until they close it.
//
// If $VIM_SERVERNAME is set (the vim-plugin can export it when the user
// opts in with g:lazysvn_vim_remote=1), Launch first tries `vim
// --servername X --remote-wait-silent path` so the file opens in the
// already-running parent vim. This fails for vim builds without
// +clientserver, for Neovim, and for user configs whose autocmds
// misbehave with remote-open; when it fails lazysvn falls back to the
// default path.
//
// The default path suspends tview and runs $EDITOR (or `vi`) directly in
// the same terminal. It works everywhere at the cost of opening a nested
// editor when lazysvn itself is running inside a vim terminal popup.
func Launch(app *tview.Application, path string) error {
	if servername := os.Getenv("VIM_SERVERNAME"); servername != "" {
		logfile.Append(fmt.Sprintf("editor: trying vim --servername %q --remote-wait-silent %s", servername, path))
		cmd := exec.Command("vim", "--servername", servername, "--remote-wait-silent", path)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err == nil {
			logfile.Append("editor: remote-wait-silent ok")
			return nil
		} else {
			logfile.Append(fmt.Sprintf("editor: remote FAILED (%v); stderr=%q; falling back to $EDITOR", err, strings.TrimSpace(stderr.String())))
		}
	}

	editorCmd := os.Getenv("EDITOR")
	if editorCmd == "" {
		editorCmd = "vi"
	}
	logfile.Append(fmt.Sprintf("editor: launching %s %s (suspended tview)", editorCmd, path))

	var cmdErr error
	app.Suspend(func() {
		parts := strings.Fields(editorCmd)
		cmd := exec.Command(parts[0], append(parts[1:], path)...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmdErr = cmd.Run()
	})
	if cmdErr != nil {
		logfile.Append(fmt.Sprintf("editor: %s exited with %v", editorCmd, cmdErr))
	}
	return cmdErr
}

// ForCommit opens a temp file in the editor and returns the trimmed
// contents as the commit message. Returns empty string if user wrote
// nothing (treated by the caller as cancellation).
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
