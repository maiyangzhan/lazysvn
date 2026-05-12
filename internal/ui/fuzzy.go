package ui

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rivo/tview"

	"github.com/maiyangzhan/lazysvn/internal/logfile"
)

// fzfAvailable reports whether fzf is on PATH.
func fzfAvailable() bool {
	_, err := exec.LookPath("fzf")
	return err == nil
}

// collectWCPaths walks the working copy and returns all file paths
// relative to root, excluding SVN metadata directories. Paths are
// sorted for stable fzf display.
func collectWCPaths(root string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			// unreadable entry — skip silently rather than aborting the whole walk
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".svn" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return nil
		}
		paths = append(paths, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}

// pickPathFuzzy suspends the tview app, runs fzf over the list of
// working-copy files, and returns (selected, picked, err).
// picked=false means the user cancelled fzf (Esc/Ctrl-C, exit 130) —
// this is NOT an error and the caller should treat it as no-op.
// Returns err when fzf is missing, when the walk fails, or when fzf
// exits non-zero for reasons other than user-cancel.
func pickPathFuzzy(app *tview.Application, wcRoot string) (string, bool, error) {
	if !fzfAvailable() {
		return "", false, fmt.Errorf("fzf not found on PATH")
	}
	paths, err := collectWCPaths(wcRoot)
	if err != nil {
		return "", false, fmt.Errorf("walk working copy: %w", err)
	}
	if len(paths) == 0 {
		return "", false, fmt.Errorf("no files under %s", wcRoot)
	}

	logfile.Append(fmt.Sprintf("fzf: presenting %d paths under %s", len(paths), wcRoot))

	var picked string
	var cancelled bool
	var runErr error

	app.Suspend(func() {
		cmd := exec.Command("fzf",
			"--prompt=path> ",
			"--reverse",
			"--height=60%",
			"--tiebreak=begin,length",
			"--info=inline",
		)
		cmd.Stdin = strings.NewReader(strings.Join(paths, "\n"))
		cmd.Stderr = os.Stderr
		out, err := cmd.Output()
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 130 {
				// user cancelled via Esc/Ctrl-C/Ctrl-G
				cancelled = true
				return
			}
			runErr = err
			return
		}
		picked = strings.TrimSpace(string(out))
	})

	if runErr != nil {
		logfile.Append(fmt.Sprintf("fzf: FAILED %v", runErr))
		return "", false, runErr
	}
	if cancelled {
		logfile.Append("fzf: cancelled by user")
		return "", false, nil
	}
	if picked == "" {
		return "", false, nil
	}
	logfile.Append(fmt.Sprintf("fzf: picked %q", picked))
	return picked, true, nil
}
