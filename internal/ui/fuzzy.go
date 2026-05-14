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
//   2. fd → fdfind → find. fd appends "/" to directory entries by
//      default, so the picker shows a clear visual distinction
//      between files and dirs. fdfind is the Debian/Ubuntu binary
//      name for the same tool. find is the portable last resort.
//
// rg is intentionally NOT in the chain: rg is a content search tool,
// not a path enumerator — `rg --files` only lists files and has no
// equivalent --type d mode, which surprised users who expected the X
// picker to include directories. If you specifically want rg, set
// $LAZYSVN_FZF_CMD.
//
// --no-ignore-vcs keeps an upstream .gitignore from hiding everything
// in an SVN WC. .ignore / .fdignore files remain respected.
//
// The shell's $FZF_DEFAULT_COMMAND is INTENTIONALLY ignored. Users
// commonly tune it for git workflows; inheriting it inside lazysvn
// has shipped zero-file surprises in the past.
func fzfDefaultCommand() (cmd, source string) {
	if v := os.Getenv("LAZYSVN_FZF_CMD"); v != "" {
		return v, "LAZYSVN_FZF_CMD"
	}
	if _, err := exec.LookPath("fd"); err == nil {
		return "fd --hidden --no-ignore-vcs --exclude .svn", "fd (default)"
	}
	if _, err := exec.LookPath("fdfind"); err == nil {
		return "fdfind --hidden --no-ignore-vcs --exclude .svn", "fdfind (default)"
	}
	return `find . -mindepth 1 -not -path './.svn/*' -not -path './.svn'`, "find (default)"
}

// pickPathFuzzy suspends the tview app and runs fzf. fzf itself runs
// FZF_DEFAULT_COMMAND concurrently so its UI appears immediately and
// candidates stream in while you type — no synchronous walk on the
// lazysvn side.
//
// When multi is true, fzf is started with --multi and the user may
// Tab-toggle multiple selections; the returned slice has one entry
// per selection. When multi is false, the slice has zero or one entry.
//
// Returns (selected, picked, err). picked=false means the user
// cancelled fzf (Esc/Ctrl-C/Ctrl-G, exit 130) — not an error. Returns
// an error when fzf is missing or exits non-zero for other reasons.
func pickPathFuzzy(app *tview.Application, wcRoot string, multi bool) ([]string, bool, error) {
	if !fzfAvailable() {
		return nil, false, fmt.Errorf("fzf not found on PATH")
	}

	defaultCmd, source := fzfDefaultCommand()
	logfile.Append(fmt.Sprintf("fzf: FZF_DEFAULT_COMMAND=%q source=%s cwd=%s multi=%v", defaultCmd, source, wcRoot, multi))

	var rawOut string
	var cancelled bool
	var runErr error
	var stderrBuf bytes.Buffer

	app.Suspend(func() {
		args := []string{
			"--prompt=path> ",
			"--reverse",
			"--height=60%",
			"--tiebreak=begin,length",
			"--info=inline",
		}
		if multi {
			args = append(args, "--multi")
		}
		cmd := exec.Command("fzf", args...)
		cmd.Dir = wcRoot
		// fzf only runs FZF_DEFAULT_COMMAND when its stdin is a TTY.
		// Go's exec connects an unset Stdin to /dev/null, which fzf
		// treats as "user piped candidates in" — it then reads stdin
		// (gets EOF immediately) and ignores FZF_DEFAULT_COMMAND
		// entirely, producing zero candidates. During app.Suspend our
		// os.Stdin IS the terminal, so wiring it through fixes that.
		cmd.Stdin = os.Stdin
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
		rawOut = out.String()
	})

	if runErr != nil {
		tail := strings.TrimSpace(stderrBuf.String())
		if len(tail) > 400 {
			tail = tail[:400] + "..."
		}
		logfile.Append(fmt.Sprintf("fzf: FAILED %v stderr=%q", runErr, tail))
		return nil, false, runErr
	}
	if cancelled {
		logfile.Append("fzf: cancelled by user")
		return nil, false, nil
	}
	var picked []string
	for _, line := range strings.Split(rawOut, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "./")
		if line != "" {
			picked = append(picked, line)
		}
	}
	if len(picked) == 0 {
		logfile.Append("fzf: no selection (empty result — candidate command may have produced zero lines)")
		return nil, false, nil
	}
	logfile.Append(fmt.Sprintf("fzf: picked %d path(s) — first=%q", len(picked), picked[0]))
	return picked, true, nil
}

// pickFromList runs fzf with the given candidates piped on stdin (no
// FZF_DEFAULT_COMMAND involved). Returns (selected, picked, err);
// picked=false means fzf was cancelled (not an error).
func pickFromList(app *tview.Application, candidates []string, prompt string) (string, bool, error) {
	if !fzfAvailable() {
		return "", false, fmt.Errorf("fzf not found on PATH")
	}
	if len(candidates) == 0 {
		return "", false, fmt.Errorf("no candidates")
	}

	var picked string
	var cancelled bool
	var runErr error
	var stderrBuf bytes.Buffer

	app.Suspend(func() {
		cmd := exec.Command("fzf",
			"--prompt="+prompt,
			"--reverse",
			"--height=60%",
			"--tiebreak=begin,length",
			"--info=inline",
		)
		// Candidates are explicit — pipe via stdin. fzf detects the
		// non-TTY stdin and reads candidates from it instead of running
		// FZF_DEFAULT_COMMAND.
		cmd.Stdin = strings.NewReader(strings.Join(candidates, "\n"))
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
	})

	if runErr != nil {
		tail := strings.TrimSpace(stderrBuf.String())
		if len(tail) > 400 {
			tail = tail[:400] + "..."
		}
		logfile.Append(fmt.Sprintf("fzf(list): FAILED %v stderr=%q", runErr, tail))
		return "", false, runErr
	}
	if cancelled || picked == "" {
		return "", false, nil
	}
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
