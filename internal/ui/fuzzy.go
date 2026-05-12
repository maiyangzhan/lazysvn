package ui

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/rivo/tview"

	"github.com/maiyangzhan/lazysvn/internal/logfile"
)

// fzfAvailable reports whether fzf is on PATH.
func fzfAvailable() bool {
	_, err := exec.LookPath("fzf")
	return err == nil
}

// fzfDefaultCommand returns the candidate-producing command fzf should
// run. Respects the user's own FZF_DEFAULT_COMMAND if they've set one
// (they likely tuned it). Otherwise picks the fastest available of
// fd → rg → find.
//
// The fd/rg defaults pass --no-ignore-vcs on purpose: without it, both
// tools respect an upstream .gitignore (e.g. one in a parent directory
// that happens to be a git repo), which for an SVN working copy often
// hides every file. With --no-ignore-vcs the .gitignore is ignored but
// .ignore / .fdignore files are still respected — so users who want
// fuzzy-exclude rules can drop a `.ignore` in their WC.
func fzfDefaultCommand() string {
	if v := os.Getenv("FZF_DEFAULT_COMMAND"); v != "" {
		return v
	}
	if _, err := exec.LookPath("fd"); err == nil {
		return "fd --type f --hidden --no-ignore-vcs --exclude .svn"
	}
	if _, err := exec.LookPath("rg"); err == nil {
		return "rg --files --hidden --no-ignore-vcs --glob '!.svn'"
	}
	return `find . -type f -not -path './.svn/*'`
}

// pickPathFuzzy suspends the tview app and runs fzf. fzf itself runs
// FZF_DEFAULT_COMMAND concurrently so its UI appears immediately and
// candidates stream in while you type — no synchronous walk on the
// lazysvn side.
//
// Returns (selected, picked, err). picked=false means the user
// cancelled fzf (Esc/Ctrl-C/Ctrl-G, exit 130) — not an error. Returns
// an error when fzf is missing or exits non-zero for other reasons.
func pickPathFuzzy(app *tview.Application, wcRoot string) (string, bool, error) {
	if !fzfAvailable() {
		return "", false, fmt.Errorf("fzf not found on PATH")
	}

	defaultCmd := fzfDefaultCommand()
	logfile.Append(fmt.Sprintf("fzf: FZF_DEFAULT_COMMAND=%q cwd=%s", defaultCmd, wcRoot))

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
		cmd.Dir = wcRoot
		// Seed env with FZF_DEFAULT_COMMAND so fzf drives the file
		// enumeration itself (streaming, concurrent with its UI startup).
		env := os.Environ()
		if os.Getenv("FZF_DEFAULT_COMMAND") == "" {
			env = append(env, "FZF_DEFAULT_COMMAND="+defaultCmd)
		}
		cmd.Env = env
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 130 {
				cancelled = true
				return
			}
			runErr = err
			return
		}
		picked = strings.TrimSpace(out.String())
		// Some candidate generators (notably `find .`) emit a leading
		// "./" on each path. svn accepts it but the UI title looks
		// nicer without it.
		picked = strings.TrimPrefix(picked, "./")
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
