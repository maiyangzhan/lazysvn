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

// fzfDefaultCommand returns the candidate-producing command lazysvn
// will hand to fzf. Resolution order:
//
//   1. $LAZYSVN_FZF_CMD — lazysvn-specific override.
//   2. Our default: fd > rg > find, with --no-ignore-vcs so an upstream
//      .gitignore doesn't silently hide every file in an SVN WC, but
//      .ignore / .fdignore files remain respected.
//
// The shell's $FZF_DEFAULT_COMMAND is INTENTIONALLY ignored here.
// That var is often tuned for git workflows (e.g. `git ls-files`) and
// inheriting it inside lazysvn has shipped zero-file surprises to
// users whose SVN WC lives under a git-tracked parent. Users who want
// a custom command for lazysvn should set $LAZYSVN_FZF_CMD.
func fzfDefaultCommand() (cmd, source string) {
	if v := os.Getenv("LAZYSVN_FZF_CMD"); v != "" {
		return v, "LAZYSVN_FZF_CMD"
	}
	if _, err := exec.LookPath("fd"); err == nil {
		return "fd --type f --hidden --no-ignore-vcs --exclude .svn", "fd (default)"
	}
	if _, err := exec.LookPath("rg"); err == nil {
		return "rg --files --hidden --no-ignore-vcs --glob '!.svn'", "rg (default)"
	}
	return `find . -type f -not -path './.svn/*'`, "find (default)"
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

	defaultCmd, source := fzfDefaultCommand()
	logfile.Append(fmt.Sprintf("fzf: FZF_DEFAULT_COMMAND=%q source=%s cwd=%s", defaultCmd, source, wcRoot))

	var picked string
	var cancelled bool
	var runErr error
	var stderrBuf bytes.Buffer

	app.Suspend(func() {
		cmd := exec.Command("fzf",
			"--prompt=path> ",
			"--reverse",
			"--height=60%",
			"--tiebreak=begin,length",
			"--info=inline",
		)
		cmd.Dir = wcRoot
		// Force FZF_DEFAULT_COMMAND to our chosen value so a shell-level
		// setting (commonly git-oriented) doesn't leak in and hide files.
		cmd.Env = replaceOrAppendEnv(os.Environ(), "FZF_DEFAULT_COMMAND", defaultCmd)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderrBuf
		if err := cmd.Run(); err != nil {
			if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 130 {
				cancelled = true
				return
			}
			runErr = err
			return
		}
		picked = strings.TrimSpace(out.String())
		picked = strings.TrimPrefix(picked, "./")
	})

	if runErr != nil {
		tail := strings.TrimSpace(stderrBuf.String())
		if len(tail) > 400 {
			tail = tail[:400] + "..."
		}
		logfile.Append(fmt.Sprintf("fzf: FAILED %v stderr=%q", runErr, tail))
		return "", false, runErr
	}
	if cancelled {
		logfile.Append("fzf: cancelled by user")
		return "", false, nil
	}
	if picked == "" {
		logfile.Append("fzf: no selection (empty result — candidate command may have produced zero lines)")
		return "", false, nil
	}
	logfile.Append(fmt.Sprintf("fzf: picked %q", picked))
	return picked, true, nil
}

// replaceOrAppendEnv returns env with key set to val — replacing an
// existing entry for key if present, otherwise appending.
func replaceOrAppendEnv(env []string, key, val string) []string {
	prefix := key + "="
	out := make([]string, 0, len(env)+1)
	for _, e := range env {
		if !strings.HasPrefix(e, prefix) {
			out = append(out, e)
		}
	}
	out = append(out, prefix+val)
	return out
}
