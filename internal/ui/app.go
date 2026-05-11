package ui

import (
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
			diff, err := a.client.Diff(entry.Path)
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
		if cached, ok := a.diffs.getRev(entry.Revision); ok {
			a.preview.SetContent(diffOrEmpty(cached))
			return
		}
		a.debounce.Do(100*time.Millisecond, func() {
			diff, err := a.client.DiffRevision(entry.Revision)
			a.app.QueueUpdateDraw(func() {
				if err != nil {
					a.preview.SetContent("Error: " + err.Error())
					return
				}
				a.diffs.setRev(entry.Revision, diff)
				a.preview.SetContent(diffOrEmpty(diff))
			})
		})
	}

	a.files.OnAction = func(key rune) {
		switch key {
		case 'c':
			a.doCommit()
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
		}
	}
}

func (a *App) runOp(spinner, opName string, op func() error, onSuccess func()) {
	dismiss := ShowSpinner(a.app, a.root, spinner)
	go func() {
		err := op()
		dismiss()
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
			func() error { return a.client.Commit(paths, msg) },
			func() {
				a.files.ClearMarks()
				a.hints.ShowInfo(fmt.Sprintf("Committed %d file(s)", len(paths)))
			})
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
			func() error { return a.client.Revert(paths) },
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
		func() error { return a.client.Add(paths) },
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
			func() error { return a.client.Remove(paths) },
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
			func() error { return a.client.Resolve(paths, mode) },
			func() {
				a.files.ClearMarks()
				a.hints.ShowInfo(fmt.Sprintf("Resolved %d file(s) with --accept=%s", len(paths), mode))
			})
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

func (a *App) doUpdate() {
	var summary svn.UpdateSummary
	a.runOp("Updating...", "update",
		func() error {
			s, err := a.client.Update()
			if err != nil {
				return err
			}
			summary = s
			return nil
		},
		func() {
			a.hints.ShowInfo(fmt.Sprintf("Updated to r%d (%d updated, %d added, %d deleted)",
				summary.Revision, summary.Updated, summary.Added, summary.Deleted))
		})
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
			{Key: "^u/^d", Label: "scroll"},
			{Key: "Space", Label: "mark"},
			{Key: "c", Label: "commit"},
			{Key: "r", Label: "revert"},
			{Key: "a", Label: "add"},
			{Key: "x", Label: "delete"},
			{Key: "e", Label: "edit"},
			{Key: "m", Label: "resolve"},
			{Key: "u", Label: "update"},
			{Key: "R", Label: "refresh"},
			{Key: "q", Label: "quit"},
		})
	} else {
		a.hints.Set([]Hint{
			{Key: "j/k", Label: "nav"},
			{Key: "^u/^d", Label: "scroll"},
			{Key: "Tab", Label: "switch"},
			{Key: "u", Label: "update"},
			{Key: "R", Label: "refresh"},
			{Key: "q", Label: "quit"},
		})
	}
}

func (a *App) Run() error {
	entries, err := a.client.Status()
	if err != nil {
		return err
	}
	logs, err := a.client.Log(a.logLimit)
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
		entries, err := a.client.Status()
		if err != nil {
			a.reportError("status", err)
			return
		}
		logs, err := a.client.Log(a.logLimit)
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
