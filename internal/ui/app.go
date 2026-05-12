package ui

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/maiyangzhan/lazysvn/internal/editor"
	"github.com/maiyangzhan/lazysvn/internal/logfile"
	"github.com/maiyangzhan/lazysvn/internal/svn"
)

type App struct {
	app     *tview.Application
	client  *svn.Client
	files   *FilesPanel
	log     *LogPanel
	preview *PreviewPanel
	hints   *HintBar

	root    tview.Primitive
	panels  []focusable
	focused int

	logLimit    int
	modalActive bool
	debounce    debouncer
	diffs       *diffCache
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
	tapp := tview.NewApplication()
	a := &App{
		app:      tapp,
		client:   client,
		files:    NewFilesPanel(),
		log:      NewLogPanel(),
		preview:  NewPreviewPanel(),
		hints:    NewHintBar(tapp),
		logLimit: logLimit,
		diffs:    newDiffCache(),
	}
	a.panels = []focusable{a.files, a.log}
	a.wireCallbacks()
	return a
}

func (a *App) wireCallbacks() {
	a.files.OnSelect = func(entry svn.FileEntry) {
		if cached, ok := a.diffs.getFile(entry.Path); ok {
			a.preview.SetContent(diffOrEmpty(cached))
			return
		}
		a.debounce.Do(100*time.Millisecond, func() {
			diff, err := a.client.Diff(context.Background(), entry.Path)
			a.app.QueueUpdateDraw(func() {
				if err != nil {
					a.preview.SetContent("Error: " + err.Error())
					return
				}
				a.diffs.setFile(entry.Path, diff)
				a.preview.SetContent(diffOrEmpty(diff))
			})
		})
	}

	a.log.OnSelect = func(entry svn.LogEntry) {
		path := a.log.Path() // "" for repo-wide log; otherwise file-filter mode
		if cached, ok := a.diffs.getRev(entry.Revision, path); ok {
			a.preview.SetContent(diffOrEmpty(cached))
			return
		}
		a.debounce.Do(100*time.Millisecond, func() {
			var diff string
			var err error
			if path != "" {
				diff, err = a.client.DiffRevisionPath(context.Background(), entry.Revision, path)
			} else {
				diff, err = a.client.DiffRevision(context.Background(), entry.Revision)
			}
			a.app.QueueUpdateDraw(func() {
				if err != nil {
					a.preview.SetContent("Error: " + err.Error())
					return
				}
				a.diffs.setRev(entry.Revision, path, diff)
				a.preview.SetContent(diffOrEmpty(diff))
			})
		})
	}

	a.files.OnAction = func(key rune) {
		switch key {
		case 'c':
			a.doCommit()
		case 'C':
			a.doCommitEditor()
		case 'r':
			a.doRevert()
		case 'a':
			a.doAdd()
		case 'x':
			a.doRemove()
		case 'm':
			a.doResolved()
		case 'e':
			a.doEdit()
		case 'L':
			a.doFileLog()
		case '/':
			a.doFilter()
		}
	}

	a.log.OnLoadMore = func() {
		a.doLoadMoreLog()
	}
	a.log.OnTogglePath = func() {
		a.doFileLogExit()
	}
	a.log.OnPromptPath = func() {
		a.doFileLogPrompt()
	}
}

func (a *App) runOp(spinner, opName string, op func(ctx context.Context) error, onSuccess func()) {
	ctx, cancel := context.WithCancel(context.Background())
	dismiss := ShowSpinner(a.app, a.root, spinner, cancel)
	go func() {
		err := op(ctx)
		dismiss()
		cancel() // release ctx resources; no-op if already cancelled
		if ctx.Err() == context.Canceled {
			a.app.QueueUpdateDraw(func() {
				a.hints.ShowInfo(fmt.Sprintf("%s cancelled", opName))
			})
			a.refreshAsync()
			return
		}
		if err != nil {
			a.reportError(opName, err)
			return
		}
		if onSuccess != nil {
			a.app.QueueUpdateDraw(onSuccess)
		}
		a.refreshAsync()
	}()
}

func (a *App) doCommit() {
	targets := a.files.MarkedOrCurrent()
	if len(targets) == 0 {
		return
	}
	paths := entriesToPaths(targets)

	a.modalActive = true
	CommitPrompt(a.app, a.root, func(msg string, cancelled bool) {
		a.modalActive = false
		if cancelled || msg == "" {
			return
		}
		a.runOp("Committing...", "commit",
			func(ctx context.Context) error { return a.client.Commit(ctx, paths, msg) },
			func() {
				a.files.ClearMarks()
				a.hints.ShowInfo(fmt.Sprintf("Committed %d file(s)", len(paths)))
			})
	})
}

func (a *App) doCommitEditor() {
	targets := a.files.MarkedOrCurrent()
	if len(targets) == 0 {
		return
	}
	paths := entriesToPaths(targets)
	msg, err := editor.ForCommit(a.app)
	if err != nil {
		a.reportError("commit", err)
		return
	}
	if msg == "" {
		a.hints.ShowInfo("Commit cancelled (empty message)")
		return
	}
	a.runOp("Committing...", "commit",
		func(ctx context.Context) error { return a.client.Commit(ctx, paths, msg) },
		func() {
			a.files.ClearMarks()
			a.hints.ShowInfo(fmt.Sprintf("Committed %d file(s)", len(paths)))
		})
}

func (a *App) doRevert() {
	targets := a.files.MarkedOrCurrent()
	if len(targets) == 0 {
		return
	}
	paths := entriesToPaths(targets)
	msg := fmt.Sprintf("Revert %d file(s)?", len(paths))
	a.modalActive = true
	Confirm(a.app, a.root, msg, func(yes bool) {
		a.modalActive = false
		if !yes {
			return
		}
		a.runOp("Reverting...", "revert",
			func(ctx context.Context) error { return a.client.Revert(ctx, paths) },
			func() {
				a.files.ClearMarks()
				a.hints.ShowInfo(fmt.Sprintf("Reverted %d file(s)", len(paths)))
			})
	})
}

func (a *App) doAdd() {
	targets := a.files.MarkedOrCurrent()
	if len(targets) == 0 {
		return
	}
	paths := entriesToPaths(targets)
	a.runOp("Adding...", "add",
		func(ctx context.Context) error { return a.client.Add(ctx, paths) },
		func() {
			a.files.ClearMarks()
			a.hints.ShowInfo(fmt.Sprintf("Added %d file(s)", len(paths)))
		})
}

func (a *App) doRemove() {
	targets := a.files.MarkedOrCurrent()
	if len(targets) == 0 {
		return
	}
	paths := entriesToPaths(targets)
	msg := fmt.Sprintf("Delete %d file(s)?", len(paths))
	a.modalActive = true
	Confirm(a.app, a.root, msg, func(yes bool) {
		a.modalActive = false
		if !yes {
			return
		}
		a.runOp("Removing...", "remove",
			func(ctx context.Context) error { return a.client.Remove(ctx, paths) },
			func() {
				a.files.ClearMarks()
				a.hints.ShowInfo(fmt.Sprintf("Removed %d file(s)", len(paths)))
			})
	})
}

func (a *App) doResolved() {
	targets := a.files.MarkedOrCurrent()
	if len(targets) == 0 {
		return
	}
	paths := entriesToPaths(targets)
	a.modalActive = true
	ResolvePrompt(a.app, a.root, len(paths), func(mode string) {
		a.modalActive = false
		if mode == "" {
			return
		}
		a.runOp(fmt.Sprintf("Resolving (%s)...", mode), "resolve",
			func(ctx context.Context) error { return a.client.Resolve(ctx, paths, mode) },
			func() {
				a.files.ClearMarks()
				a.hints.ShowInfo(fmt.Sprintf("Resolved %d file(s) with --accept=%s", len(paths), mode))
			})
	})
}

func (a *App) doUpdate() {
	var summary svn.UpdateSummary
	a.runOp("Updating...", "update",
		func(ctx context.Context) error {
			s, err := a.client.Update(ctx)
			if err != nil {
				return err
			}
			summary = s
			return nil
		},
		func() {
			msg := fmt.Sprintf("Updated to r%d (%d updated, %d added, %d deleted)",
				summary.Revision, summary.Updated, summary.Added, summary.Deleted)
			if summary.Conflicted > 0 {
				a.hints.ShowError(fmt.Sprintf("%s — ⚠ %d conflict(s); press e to edit, m to resolve",
					msg, summary.Conflicted))
			} else {
				a.hints.ShowInfo(msg)
			}
		})
}

func (a *App) doLoadMoreLog() {
	oldest := a.log.OldestRevision()
	if oldest <= 1 {
		a.hints.ShowInfo("No more log entries")
		return
	}
	path := a.log.Path()
	go func() {
		ctx := context.Background()
		var entries []svn.LogEntry
		var err error
		if path != "" {
			entries, err = a.client.LogPathBefore(ctx, path, oldest, a.logLimit)
		} else {
			entries, err = a.client.LogBefore(ctx, oldest, a.logLimit)
		}
		if err != nil {
			a.reportError("log", err)
			return
		}
		if len(entries) == 0 {
			a.app.QueueUpdateDraw(func() {
				a.hints.ShowInfo("No more log entries")
			})
			return
		}
		a.app.QueueUpdateDraw(func() {
			a.log.AppendEntries(entries)
			a.hints.ShowInfo(fmt.Sprintf("Loaded %d more log entries", len(entries)))
		})
	}()
}

func (a *App) doFileLog() {
	entry := a.files.SelectedEntry()
	if entry == nil {
		return
	}
	a.doFileLogFor(entry.Path)
}

// doFileLogPrompt is bound to L in the log panel: lets the user type a
// path (any path, even one that has no pending changes and therefore
// isn't shown in the Files panel).
func (a *App) doFileLogPrompt() {
	a.modalActive = true
	PathPrompt(a.app, a.root, a.log.Path(), func(path string, cancelled bool) {
		a.modalActive = false
		if cancelled {
			return
		}
		if path == "" {
			a.doFileLogExit()
			return
		}
		a.doFileLogFor(path)
	})
}

// doFileLogFor switches the Log panel into path-filter mode for the
// given path, or exits path mode if it's already the active path.
func (a *App) doFileLogFor(path string) {
	if a.log.Path() == path {
		a.doFileLogExit()
		return
	}
	a.log.SetPathMode(path)
	go func() {
		logs, err := a.client.LogPath(context.Background(), path, a.logLimit)
		if err != nil {
			a.reportError("log", err)
			a.app.QueueUpdateDraw(func() { a.log.SetPathMode("") })
			return
		}
		a.app.QueueUpdateDraw(func() {
			a.log.SetEntries(logs)
			a.hints.ShowInfo(fmt.Sprintf("Filtered log to %s (%d entries) — Esc to exit", path, len(logs)))
		})
	}()
}

func (a *App) doFileLogExit() {
	a.log.SetPathMode("")
	a.refreshAsync()
}

func (a *App) doHelp() {
	a.modalActive = true
	HelpModal(a.app, a.root, func() {
		a.modalActive = false
	})
}

func (a *App) doFilter() {
	a.modalActive = true
	FilterPrompt(a.app, a.root, a.files.Filter(), func(pattern string, cancelled bool) {
		a.modalActive = false
		if cancelled {
			return
		}
		a.files.SetFilter(pattern)
		if pattern == "" {
			a.hints.ShowInfo("Filter cleared")
		} else {
			a.hints.ShowInfo(fmt.Sprintf("Filter: %q", pattern))
		}
	})
}

func (a *App) doEdit() {
	entry := a.files.SelectedEntry()
	if entry == nil {
		return
	}
	path := entry.Path
	if err := editor.Launch(a.app, path); err != nil {
		a.reportError("edit", err)
		return
	}
	// File content may have changed; drop its cached diff and refresh.
	a.diffs.clearFiles()
	a.refreshAsync()
}

func (a *App) reportError(op string, err error) {
	msg := fmt.Sprintf("%s: %s", op, err.Error())
	logfile.Append(msg)
	a.app.QueueUpdateDraw(func() {
		a.hints.ShowError(msg)
	})
}

func (a *App) updateHints() {
	if a.focused == 0 {
		a.hints.Set([]Hint{
			{Key: "j/k", Label: "nav"},
			{Key: "Space", Label: "mark"},
			{Key: "/", Label: "filter"},
			{Key: "c/C", Label: "commit"},
			{Key: "r", Label: "revert"},
			{Key: "a", Label: "add"},
			{Key: "x", Label: "delete"},
			{Key: "e", Label: "edit"},
			{Key: "m", Label: "resolve"},
			{Key: "L", Label: "file log"},
			{Key: "u", Label: "update"},
			{Key: "?", Label: "help"},
			{Key: "q", Label: "quit"},
		})
	} else {
		a.hints.Set([]Hint{
			{Key: "j/k", Label: "nav"},
			{Key: "M", Label: "load more"},
			{Key: "L", Label: "path log"},
			{Key: "Esc", Label: "exit path"},
			{Key: "Tab", Label: "switch"},
			{Key: "u", Label: "update"},
			{Key: "?", Label: "help"},
			{Key: "q", Label: "quit"},
		})
	}
}

func (a *App) Run() error {
	ctx := context.Background()
	entries, err := a.client.Status(ctx)
	if err != nil {
		return err
	}
	logs, err := a.client.Log(ctx, a.logLimit)
	if err != nil {
		return err
	}
	a.files.SetEntries(entries)
	a.log.SetEntries(logs)

	leftCol := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.files.View(), 0, 1, true).
		AddItem(a.log.View(), 0, 1, false)

	content := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(leftCol, 0, 2, true).
		AddItem(a.preview.View(), 0, 3, false)

	root := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(content, 0, 1, true).
		AddItem(a.hints.View(), 1, 0, false)

	a.root = root
	a.app.SetRoot(root, true)
	a.setFocus(0)
	a.app.SetInputCapture(a.globalKeys)

	return a.app.Run()
}

func (a *App) globalKeys(event *tcell.EventKey) *tcell.EventKey {
	if a.modalActive {
		return event
	}
	switch event.Key() {
	case tcell.KeyTab:
		a.cycleFocus(1)
		return nil
	case tcell.KeyBacktab:
		a.cycleFocus(-1)
		return nil
	case tcell.KeyCtrlD:
		a.preview.ScrollHalfPageDown()
		return nil
	case tcell.KeyCtrlU:
		a.preview.ScrollHalfPageUp()
		return nil
	}
	switch event.Rune() {
	case 'q':
		a.app.Stop()
		return nil
	case 'R':
		a.refreshAsync()
		return nil
	case 'u':
		a.doUpdate()
		return nil
	case '?':
		a.doHelp()
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
	a.updateHints()
}

func (a *App) refreshAsync() {
	go func() {
		ctx := context.Background()
		entries, err := a.client.Status(ctx)
		if err != nil {
			a.reportError("status", err)
			return
		}
		var logs []svn.LogEntry
		if path := a.log.Path(); path != "" {
			logs, err = a.client.LogPath(ctx, path, a.logLimit)
		} else {
			logs, err = a.client.Log(ctx, a.logLimit)
		}
		if err != nil {
			a.reportError("log", err)
			return
		}
		a.diffs.clearFiles()
		a.app.QueueUpdateDraw(func() {
			a.files.SetEntries(entries)
			a.log.SetEntries(logs)
		})
	}()
}

func entriesToPaths(entries []svn.FileEntry) []string {
	paths := make([]string, len(entries))
	for i, e := range entries {
		paths[i] = e.Path
	}
	return paths
}

func diffOrEmpty(diff string) string {
	if diff == "" {
		return "(no changes)"
	}
	return diff
}
