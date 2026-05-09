package ui

import (
	"fmt"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

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
	}
	a.panels = []focusable{a.files, a.log}
	a.wireCallbacks()
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
		}
	}
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
		if err := a.client.Commit(paths, msg); err != nil {
			a.reportError("commit", err)
			return
		}
		a.files.ClearMarks()
		a.hints.ShowInfo(fmt.Sprintf("Committed %d file(s)", len(paths)))
		a.refreshAsync()
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
		if err := a.client.Revert(paths); err != nil {
			a.reportError("revert", err)
			return
		}
		a.files.ClearMarks()
		a.hints.ShowInfo(fmt.Sprintf("Reverted %d file(s)", len(paths)))
		a.refreshAsync()
	})
}

func (a *App) doAdd() {
	targets := a.files.MarkedOrCurrent()
	if len(targets) == 0 {
		return
	}
	paths := entriesToPaths(targets)
	if err := a.client.Add(paths); err != nil {
		a.reportError("add", err)
		return
	}
	a.files.ClearMarks()
	a.hints.ShowInfo(fmt.Sprintf("Added %d file(s)", len(paths)))
	a.refreshAsync()
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
		if err := a.client.Remove(paths); err != nil {
			a.reportError("remove", err)
			return
		}
		a.files.ClearMarks()
		a.hints.ShowInfo(fmt.Sprintf("Removed %d file(s)", len(paths)))
		a.refreshAsync()
	})
}

func (a *App) doResolved() {
	targets := a.files.MarkedOrCurrent()
	if len(targets) == 0 {
		return
	}
	paths := entriesToPaths(targets)
	if err := a.client.Resolved(paths); err != nil {
		a.reportError("resolved", err)
		return
	}
	a.files.ClearMarks()
	a.hints.ShowInfo(fmt.Sprintf("Resolved %d file(s)", len(paths)))
	a.refreshAsync()
}

func (a *App) doUpdate() {
	dismiss := ShowSpinner(a.app, a.root, "Updating...")
	go func() {
		summary, err := a.client.Update()
		dismiss()
		if err != nil {
			a.reportError("update", err)
			return
		}
		a.app.QueueUpdateDraw(func() {
			a.hints.ShowInfo(fmt.Sprintf("Updated to r%d (%d updated, %d added, %d deleted)",
				summary.Revision, summary.Updated, summary.Added, summary.Deleted))
		})
		a.refreshAsync()
	}()
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
			{Key: "m", Label: "resolved"},
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
			return
		}
		logs, err := a.client.Log(a.logLimit)
		if err != nil {
			return
		}
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
