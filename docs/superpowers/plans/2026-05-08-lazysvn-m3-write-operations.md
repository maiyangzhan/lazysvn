# lazysvn M3 — Write Operations Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add write operations to the TUI: `c` commit, `r` revert, `a` add, `x` delete, `m` resolved, `Space` multi-mark. Commit via `$EDITOR` with tview suspend/resume. Confirm modals for destructive ops.

**Architecture:** New `internal/editor/` package for editor spawning. Multi-mark state added to `FilesPanel`. Confirm modal added to `internal/ui/`. All write ops wired through `app.go` callbacks — panels remain decoupled.

**Reference:** Design spec sections "Commit message input", "Editor launcher", "Interaction details".

**Parent directory:** `/Users/myz/claude_work/svn_tui/lazysvn/`

---

## Task 1: Multi-mark in FilesPanel

Add mark toggling with Space. Marked files shown with `◆` indicator. `MarkedOrCurrent()` returns marked entries or falls back to current entry.

**Files:**
- Modified: `internal/ui/files.go`

- [ ] **Step 1: Add marks set and methods**

Add to `FilesPanel` struct:
```go
marks map[string]bool  // keyed by path
```

Add methods:
```go
func (p *FilesPanel) ToggleMark()
func (p *FilesPanel) ClearMarks()
func (p *FilesPanel) MarkedOrCurrent() []svn.FileEntry
func (p *FilesPanel) HasMarks() bool
```

- [ ] **Step 2: Update SetEntries to render marks**

Marked items show `◆` before the status letter. Unmarked show a space.

- [ ] **Step 3: Handle Space in InputCapture**

Space toggles mark on current item, then moves cursor down (like lazygit).

---

## Task 2: Editor launcher (`internal/editor/launch.go`)

Spawn `$EDITOR` (or `vi`) on a temp file for commit messages. Handle `$VIM_SERVERNAME` for nested-vim case.

**Files:**
- Create: `internal/editor/launch.go`

- [ ] **Step 1: Create launch.go**

```go
func Launch(app *tview.Application, path string) error
```

Logic:
- If `$VIM_SERVERNAME` set → `vim --servername $name --remote-wait-silent <path>` (no suspend)
- Else → `app.Suspend()` + run `$EDITOR <path>` (or `vi`) + resume

- [ ] **Step 2: Create ForCommit helper**

```go
func ForCommit(app *tview.Application) (string, error)
```

Creates temp file, launches editor, reads content, cleans up. Returns empty string if user saved empty file (abort commit).

---

## Task 3: Confirm modal (`internal/ui/modal.go`)

A tview.Modal for confirming destructive operations (revert, delete).

**Files:**
- Create: `internal/ui/modal.go`

- [ ] **Step 1: Create modal.go**

```go
func Confirm(app *tview.Application, root tview.Primitive, message string, onYes func())
```

Shows a modal with Yes/No buttons. On Yes: calls `onYes` callback. On No or Escape: closes modal. Restores previous root after dismissal.

---

## Task 4: Add CommitFromFile to svn client

The existing `Commit(paths, msg)` uses `-m`. For multi-line editor messages, we need `-F <file>`.

**Files:**
- Modified: `internal/svn/write.go`

- [ ] **Step 1: Add CommitFromFile method**

```go
func (c *Client) CommitFromFile(paths []string, msgFile string) error {
    args := append([]string{"commit", "-F", msgFile}, paths...)
    _, err := c.run(args...)
    return err
}
```

---

## Task 5: Wire write operations in app.go

Connect all keys to svn operations. Only active when files panel is focused.

**Files:**
- Modified: `internal/ui/app.go`

- [ ] **Step 1: Add filesKeys handler**

Keys handled in files panel InputCapture (not global):
- `Space` → toggle mark (handled in FilesPanel itself)
- `c` → commit flow (editor → CommitFromFile → refresh)
- `r` → confirm → revert → refresh
- `a` → add → refresh
- `x` → confirm → remove → refresh
- `m` → resolved → refresh

- [ ] **Step 2: Add refresh after write ops**

Each write op refreshes files + log panels after completion.

- [ ] **Step 3: Store root primitive for modal restore**

The confirm modal needs a reference to the root Flex to restore it after dismissal.

---

## Task 6: Update hint bar

Show context-sensitive hints: different keys when files panel is focused vs log panel.

**Files:**
- Modified: `internal/ui/app.go`

- [ ] **Step 1: Update setHints to be context-aware**

Files panel hints: `j/k:nav  Space:mark  c:commit  r:revert  a:add  x:delete  m:resolved  R:refresh  q:quit`
Log panel hints: `j/k:nav  Tab:switch  R:refresh  q:quit`

Call `updateHints()` on focus change.

---

## Task 7: Build, test, iterate

- [ ] **Step 1:** `go build ./...`
- [ ] **Step 2:** `make test` (M1 tests still pass)
- [ ] **Step 3:** `make linux`
- [ ] **Step 4:** Manual smoke test against real SVN working copy
