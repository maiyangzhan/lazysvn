# lazysvn M2 — UI Skeleton (Read-Only) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a read-only tview TUI with three panels (files, log, preview) + hint bar. Wire `Status()` and `Log()` from M1's data layer into the UI. Support cursor navigation, focus switching, diff preview with debounce. No write operations — those are M3.

**Architecture:** New `internal/ui/` package. Panels never call each other directly; all wiring goes through `app.go` callbacks. The existing `cmd/lazysvn/main.go` is replaced with the tview app entry point.

**Tech Stack:** Go 1.26, `github.com/rivo/tview` (first external dependency), `github.com/gdamore/tcell/v2` (transitive via tview).

**Reference:** Design spec at `docs/superpowers/specs/2026-05-08-lazysvn-design.md`, section "internal/ui/ — tview layer".

**Parent directory of this plan:** `/Users/myz/claude_work/svn_tui/lazysvn/`. All paths below are relative to this directory unless otherwise noted.

---

## Layout

```
+---------------------+----------------------------+
|   Files Panel       |                            |
|   (tview.List)      |     Preview Panel          |
+---------------------+     (tview.TextView)       |
|   Log Panel         |                            |
|   (tview.List)      |                            |
+---------------------+----------------------------+
|  d:diff  R:refresh  q:quit  Tab:switch panel     |
+--------------------------------------------------+
```

Root is a vertical `Flex`:
- Row 0: horizontal `Flex` (proportion 1)
  - Col 0: vertical `Flex` (proportion 2) → files panel + log panel (equal weight)
  - Col 1: preview panel (proportion 3)
- Row 1: hint bar (fixed height 1)

---

## File Map

| File | Purpose |
|---|---|
| `internal/ui/app.go` | `App` struct, `tview.Application`, `Flex` root, panel wiring, focus routing, debounce |
| `internal/ui/files.go` | `FilesPanel` — displays `svn.FileEntry` list with status coloring |
| `internal/ui/log.go` | `LogPanel` — displays `svn.LogEntry` list |
| `internal/ui/preview.go` | `PreviewPanel` — passive `TextView`, `SetContent(text)`, diff colorization |
| `internal/ui/hints.go` | `HintBar` — passive single-line `TextView` at bottom |
| `cmd/lazysvn/main.go` | Replace throwaway binary with tview app |

---

## Task 1: Add tview dependency

**Files:**
- Modified: `go.mod`, `go.sum`

- [ ] **Step 1: Fetch tview**

Run from `/Users/myz/claude_work/svn_tui/lazysvn/`:

```bash
go get github.com/rivo/tview@latest
```

- [ ] **Step 2: Tidy**

```bash
go mod tidy
```

- [ ] **Step 3: Verify**

`go.mod` should now list `github.com/rivo/tview` and `github.com/gdamore/tcell/v2` (transitive). Run `go build ./...` to confirm no import errors.

---

## Task 2: Preview panel (`internal/ui/preview.go`)

Simplest panel — passive text display. Build first so other panels can target it during wiring.

**Files:**
- Create: `internal/ui/preview.go`

- [ ] **Step 1: Create preview.go**

```go
package ui

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type PreviewPanel struct {
	view *tview.TextView
}

func NewPreviewPanel() *PreviewPanel {
	tv := tview.NewTextView()
	tv.SetDynamicColors(true)
	tv.SetBorder(true)
	tv.SetTitle(" Preview ")
	tv.SetTitleColor(tcell.ColorWhite)
	tv.SetScrollable(true)
	tv.SetWrap(false)
	return &PreviewPanel{view: tv}
}

func (p *PreviewPanel) View() tview.Primitive {
	return p.view
}

func (p *PreviewPanel) SetContent(text string) {
	p.view.Clear()
	p.view.SetText(colorizeDiff(text))
	p.view.ScrollToBeginning()
}

func colorizeDiff(text string) string {
	if text == "" {
		return ""
	}
	var buf strings.Builder
	for _, line := range strings.Split(text, "\n") {
		escaped := tview.Escape(line)
		switch {
		case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
			buf.WriteString("[white::b]")
			buf.WriteString(escaped)
			buf.WriteString("[-::-]")
		case strings.HasPrefix(line, "+"):
			buf.WriteString("[green]")
			buf.WriteString(escaped)
			buf.WriteString("[-]")
		case strings.HasPrefix(line, "-"):
			buf.WriteString("[red]")
			buf.WriteString(escaped)
			buf.WriteString("[-]")
		case strings.HasPrefix(line, "@@"):
			buf.WriteString("[cyan]")
			buf.WriteString(escaped)
			buf.WriteString("[-]")
		case strings.HasPrefix(line, "Index:") || strings.HasPrefix(line, "====="):
			buf.WriteString("[grey]")
			buf.WriteString(escaped)
			buf.WriteString("[-]")
		default:
			buf.WriteString(escaped)
		}
		buf.WriteByte('\n')
	}
	return buf.String()
}
```

---

## Task 3: Hint bar (`internal/ui/hints.go`)

Passive single-line bar at the bottom showing context-sensitive key hints.

**Files:**
- Create: `internal/ui/hints.go`

- [ ] **Step 1: Create hints.go**

```go
package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Hint struct {
	Key   string
	Label string
}

type HintBar struct {
	view *tview.TextView
}

func NewHintBar() *HintBar {
	tv := tview.NewTextView()
	tv.SetDynamicColors(true)
	tv.SetBackgroundColor(tcell.ColorDarkSlateGray)
	tv.SetTextAlign(tview.AlignLeft)
	return &HintBar{view: tv}
}

func (h *HintBar) View() tview.Primitive {
	return h.view
}

func (h *HintBar) Set(hints []Hint) {
	var parts []string
	for _, hint := range hints {
		parts = append(parts, fmt.Sprintf("[yellow]%s[white]: %s", hint.Key, hint.Label))
	}
	h.view.SetText(" " + strings.Join(parts, "  "))
}
```

---

## Task 4: Files panel (`internal/ui/files.go`)

Displays `svn.FileEntry` items with status letter + color. Supports j/k navigation. Fires `OnSelect` callback on cursor change for preview wiring.

**Files:**
- Create: `internal/ui/files.go`

- [ ] **Step 1: Create files.go**

```go
package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/maiyangzhan/lazysvn/internal/svn"
)

type FilesPanel struct {
	list     *tview.List
	entries  []svn.FileEntry
	OnSelect func(entry svn.FileEntry)
}

func NewFilesPanel() *FilesPanel {
	p := &FilesPanel{}
	l := tview.NewList()
	l.ShowSecondaryText(false)
	l.SetHighlightFullLine(true)
	l.SetBorder(true)
	l.SetTitle(" Files ")
	l.SetTitleColor(tcell.ColorWhite)
	l.SetChangedFunc(func(index int, _, _ string, _ rune) {
		if p.OnSelect != nil && index >= 0 && index < len(p.entries) {
			p.OnSelect(p.entries[index])
		}
	})
	l.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'j':
			return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
		case 'k':
			return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
		case 'g':
			return tcell.NewEventKey(tcell.KeyHome, 0, tcell.ModNone)
		case 'G':
			return tcell.NewEventKey(tcell.KeyEnd, 0, tcell.ModNone)
		}
		return event
	})
	p.list = l
	return p
}

func (p *FilesPanel) View() tview.Primitive {
	return p.list
}

func (p *FilesPanel) SetEntries(entries []svn.FileEntry) {
	p.entries = entries
	cur := p.list.GetCurrentItem()
	p.list.Clear()
	for _, e := range entries {
		label := fmt.Sprintf("[%s]%s[-]  %s", statusColor(e.Status), statusLetter(e.Status), e.Path)
		p.list.AddItem(label, "", 0, nil)
	}
	if cur >= p.list.GetItemCount() {
		cur = p.list.GetItemCount() - 1
	}
	if cur >= 0 {
		p.list.SetCurrentItem(cur)
	}
}

func (p *FilesPanel) SelectedEntry() *svn.FileEntry {
	idx := p.list.GetCurrentItem()
	if idx < 0 || idx >= len(p.entries) {
		return nil
	}
	return &p.entries[idx]
}

func (p *FilesPanel) Focus() {
	p.list.SetBorderColor(tcell.ColorBlue)
}

func (p *FilesPanel) Blur() {
	p.list.SetBorderColor(tcell.ColorWhite)
}

func statusLetter(s svn.Status) string {
	switch s {
	case svn.Modified:
		return "M"
	case svn.Added:
		return "A"
	case svn.Deleted:
		return "D"
	case svn.Untracked:
		return "?"
	case svn.Conflicted:
		return "C"
	default:
		return " "
	}
}

func statusColor(s svn.Status) string {
	switch s {
	case svn.Modified:
		return "yellow"
	case svn.Added:
		return "green"
	case svn.Deleted:
		return "red"
	case svn.Untracked:
		return "grey"
	case svn.Conflicted:
		return "red"
	default:
		return "white"
	}
}
```

---

## Task 5: Log panel (`internal/ui/log.go`)

Displays `svn.LogEntry` items with revision, author, date, and message. Same j/k navigation and `OnSelect` callback pattern as files panel.

**Files:**
- Create: `internal/ui/log.go`

- [ ] **Step 1: Create log.go**

```go
package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/maiyangzhan/lazysvn/internal/svn"
)

type LogPanel struct {
	list    *tview.List
	entries []svn.LogEntry
	OnSelect func(entry svn.LogEntry)
}

func NewLogPanel() *LogPanel {
	p := &LogPanel{}
	l := tview.NewList()
	l.ShowSecondaryText(false)
	l.SetHighlightFullLine(true)
	l.SetBorder(true)
	l.SetTitle(" Log ")
	l.SetTitleColor(tcell.ColorWhite)
	l.SetChangedFunc(func(index int, _, _ string, _ rune) {
		if p.OnSelect != nil && index >= 0 && index < len(p.entries) {
			p.OnSelect(p.entries[index])
		}
	})
	l.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'j':
			return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
		case 'k':
			return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
		case 'g':
			return tcell.NewEventKey(tcell.KeyHome, 0, tcell.ModNone)
		case 'G':
			return tcell.NewEventKey(tcell.KeyEnd, 0, tcell.ModNone)
		}
		return event
	})
	p.list = l
	return p
}

func (p *LogPanel) View() tview.Primitive {
	return p.list
}

func (p *LogPanel) SetEntries(entries []svn.LogEntry) {
	p.entries = entries
	cur := p.list.GetCurrentItem()
	p.list.Clear()
	for _, e := range entries {
		msg := e.Message
		if len(msg) > 60 {
			msg = msg[:57] + "..."
		}
		label := fmt.Sprintf("[cyan]r%d[-]  [grey]%s[-]  %s  %s",
			e.Revision, e.Author, e.Date.Format("01-02 15:04"), msg)
		p.list.AddItem(label, "", 0, nil)
	}
	if cur >= p.list.GetItemCount() {
		cur = p.list.GetItemCount() - 1
	}
	if cur >= 0 {
		p.list.SetCurrentItem(cur)
	}
}

func (p *LogPanel) SelectedEntry() *svn.LogEntry {
	idx := p.list.GetCurrentItem()
	if idx < 0 || idx >= len(p.entries) {
		return nil
	}
	return &p.entries[idx]
}

func (p *LogPanel) Focus() {
	p.list.SetBorderColor(tcell.ColorBlue)
}

func (p *LogPanel) Blur() {
	p.list.SetBorderColor(tcell.ColorWhite)
}
```

---

## Task 6: App wiring (`internal/ui/app.go`)

Central orchestration: creates all panels, builds the Flex layout, wires callbacks, manages focus, implements debounced preview loading.

**Files:**
- Create: `internal/ui/app.go`

- [ ] **Step 1: Create app.go**

```go
package ui

import (
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/maiyangzhan/lazysvn/internal/svn"
)

type App struct {
	app     *tview.Application
	client  *svn.Client
	files   *FilesPanel
	log     *LogPanel
	preview *PreviewPanel
	hints   *HintBar

	panels  []focusable
	focused int

	logLimit int
	debounce debouncer
}

type focusable interface {
	View() tview.Primitive
	Focus()
	Blur()
}

type debouncer struct {
	mu    sync.Mutex
	timer *time.Timer
}

func (d *debouncer) Do(delay time.Duration, fn func()) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.timer != nil {
		d.timer.Stop()
	}
	d.timer = time.AfterFunc(delay, fn)
}

func NewApp(client *svn.Client, logLimit int) *App {
	a := &App{
		app:      tview.NewApplication(),
		client:   client,
		files:    NewFilesPanel(),
		log:      NewLogPanel(),
		preview:  NewPreviewPanel(),
		hints:    NewHintBar(),
		logLimit: logLimit,
	}
	a.panels = []focusable{a.files, a.log}
	a.wireCallbacks()
	a.setHints()
	return a
}

func (a *App) wireCallbacks() {
	a.files.OnSelect = func(entry svn.FileEntry) {
		a.debounce.Do(100*time.Millisecond, func() {
			diff, err := a.client.Diff(entry.Path)
			a.app.QueueUpdateDraw(func() {
				if err != nil {
					a.preview.SetContent("Error: " + err.Error())
				} else if diff == "" {
					a.preview.SetContent("(no changes)")
				} else {
					a.preview.SetContent(diff)
				}
			})
		})
	}

	a.log.OnSelect = func(entry svn.LogEntry) {
		a.debounce.Do(100*time.Millisecond, func() {
			diff, err := a.client.DiffRevision(entry.Revision)
			a.app.QueueUpdateDraw(func() {
				if err != nil {
					a.preview.SetContent("Error: " + err.Error())
				} else if diff == "" {
					a.preview.SetContent("(no changes)")
				} else {
					a.preview.SetContent(diff)
				}
			})
		})
	}
}

func (a *App) setHints() {
	a.hints.Set([]Hint{
		{Key: "j/k", Label: "navigate"},
		{Key: "Tab", Label: "switch panel"},
		{Key: "R", Label: "refresh"},
		{Key: "q", Label: "quit"},
	})
}

func (a *App) Run() error {
	if err := a.refresh(); err != nil {
		return err
	}

	leftCol := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.files.View(), 0, 1, true).
		AddItem(a.log.View(), 0, 1, false)

	content := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(leftCol, 0, 2, true).
		AddItem(a.preview.View(), 0, 3, false)

	root := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(content, 0, 1, true).
		AddItem(a.hints.View(), 1, 0, false)

	a.app.SetRoot(root, true)
	a.setFocus(0)
	a.app.SetInputCapture(a.globalKeys)

	return a.app.Run()
}

func (a *App) globalKeys(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyTab:
		a.cycleFocus(1)
		return nil
	case tcell.KeyBacktab:
		a.cycleFocus(-1)
		return nil
	}
	switch event.Rune() {
	case 'q':
		a.app.Stop()
		return nil
	case 'R':
		go func() {
			a.refresh()
			a.app.QueueUpdateDraw(func() {})
		}()
		return nil
	}
	return event
}

func (a *App) cycleFocus(delta int) {
	next := (a.focused + delta + len(a.panels)) % len(a.panels)
	a.setFocus(next)
}

func (a *App) setFocus(idx int) {
	if a.focused < len(a.panels) {
		a.panels[a.focused].Blur()
	}
	a.focused = idx
	a.panels[a.focused].Focus()
	a.app.SetFocus(a.panels[a.focused].View())
}

func (a *App) refresh() error {
	entries, err := a.client.Status()
	if err != nil {
		return err
	}
	logs, err := a.client.Log(a.logLimit)
	if err != nil {
		return err
	}
	a.app.QueueUpdateDraw(func() {
		a.files.SetEntries(entries)
		a.log.SetEntries(logs)
	})
	return nil
}
```

**Key design decisions:**
- `focusable` interface: only files and log panels participate in focus cycling. Preview and hints are passive.
- `debouncer`: 100ms timer per the design spec. Uses `time.AfterFunc` + `sync.Mutex` — goroutine-safe, last-writer-wins.
- `QueueUpdateDraw`: all UI mutations from goroutines go through tview's thread-safe queue.
- `refresh()` calls `QueueUpdateDraw` internally so it works from both the main goroutine (initial load) and background goroutines (`R` key).

---

## Task 7: Replace `cmd/lazysvn/main.go`

Replace the throwaway verify binary with the tview app entry point.

**Files:**
- Modified: `cmd/lazysvn/main.go`

- [ ] **Step 1: Rewrite main.go**

```go
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/maiyangzhan/lazysvn/internal/svn"
	"github.com/maiyangzhan/lazysvn/internal/ui"
)

func main() {
	cwd := flag.String("cwd", ".", "working copy directory")
	logLimit := flag.Int("log-limit", 50, "number of log entries to show")
	flag.Parse()

	client := svn.New(*cwd)
	app := ui.NewApp(client, *logLimit)

	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "lazysvn: %v\n", err)
		os.Exit(1)
	}
}
```

**Changes from M1:**
- Removed `fmt.Println` debug output.
- Default `log-limit` changed from 5 to 50 (matching the design spec config default).
- Creates `ui.App` and calls `Run()` instead of printing raw data.

---

## Task 8: Build, test, iterate

- [ ] **Step 1: Compile**

```bash
cd /Users/myz/claude_work/svn_tui/lazysvn && go build ./...
```

Fix any compilation errors. Common issues: import paths, tview API mismatches.

- [ ] **Step 2: Run existing tests**

```bash
make test
```

All 13 M1 tests must still pass. The UI has no automated tests in M2 (state tests are M3 when write operations add testable state transitions).

- [ ] **Step 3: Cross-compile**

```bash
make linux
```

Verify `dist/lazysvn-linux-amd64` is produced and is a static ELF:

```bash
file dist/lazysvn-linux-amd64
```

- [ ] **Step 4: Manual smoke test**

Run `go run ./cmd/lazysvn --cwd <path-to-svn-working-copy>` against a real SVN working copy. Verify:
1. Three panels render with borders and titles.
2. Files panel shows status entries with color-coded status letters.
3. Log panel shows revision history.
4. Cursor moves with j/k in files and log panels.
5. Tab switches focus between files and log (blue border follows).
6. Preview panel auto-updates with diff content on cursor change.
7. Diff content is colorized (green +, red -, cyan @@).
8. R refreshes all panels.
9. q quits cleanly (terminal state restored).
10. g/G jump to first/last item.

Fix any issues found during smoke testing.

---

## Summary of key bindings (M2)

| Key | Context | Action |
|---|---|---|
| `j` / `k` | files, log | Move cursor down / up |
| `g` / `G` | files, log | Jump to first / last item |
| `Tab` / `Shift-Tab` | global | Cycle focus between panels |
| `R` | global | Refresh files + log from svn |
| `q` | global | Quit |

Preview updates automatically on cursor change (100ms debounce). No explicit `d` key needed — the preview is always live.

---

## What M2 does NOT include (deferred to M3+)

- Write operations: commit, revert, add, delete, resolved (`c`/`r`/`a`/`x`/`m`)
- Multi-mark with Space
- Update (`u`) with spinner
- Error toast on hint bar
- Confirm modals
- Editor integration (`o`/`Enter`)
- Config file loading
