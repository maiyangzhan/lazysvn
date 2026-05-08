# lazysvn M4 — Long Operations + Error Reporting

**Goal:** Add `u` (svn update) with spinner modal + goroutine, error toast on hint bar (3s auto-dismiss), log file at `~/.cache/lazysvn/log`.

**Parent directory:** `/Users/myz/claude_work/svn_tui/lazysvn/`

---

## Task 1: Error logging (`internal/logfile/logfile.go`)

Write svn stderr to `~/.cache/lazysvn/log` (append, create dirs if needed).

## Task 2: Error toast on hint bar

Replace hint bar text with red error message for 3 seconds, then restore.

## Task 3: Update with spinner (`u` key)

Show spinner modal → run `svn update` in goroutine → close modal → show result toast → refresh panels.

## Task 4: Wire into app.go

Global `u` key → update flow. Error reporting through toast instead of preview panel.

## Task 5: Build + test
