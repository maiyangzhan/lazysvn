package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/maiyangzhan/lazysvn/internal/svn"
)

type LogPanel struct {
	list       *tview.List
	entries    []svn.LogEntry
	path       string // non-empty = path-filter mode ("single-file log")
	OnSelect   func(entry svn.LogEntry)
	OnLoadMore func()
	OnTogglePath func()
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
		case 'M':
			if p.OnLoadMore != nil {
				p.OnLoadMore()
			}
			return nil
		}
		if event.Key() == tcell.KeyEscape && p.path != "" {
			if p.OnTogglePath != nil {
				p.OnTogglePath()
			}
			return nil
		}
		return event
	})
	p.list = l
	return p
}

func (p *LogPanel) View() tview.Primitive {
	return p.list
}

func (p *LogPanel) Path() string {
	return p.path
}

func (p *LogPanel) SetEntries(entries []svn.LogEntry) {
	p.entries = entries
	cur := p.list.GetCurrentItem()
	p.list.Clear()
	for _, e := range entries {
		p.list.AddItem(formatLogEntry(e), "", 0, nil)
	}
	if cur >= p.list.GetItemCount() {
		cur = p.list.GetItemCount() - 1
	}
	if cur >= 0 {
		p.list.SetCurrentItem(cur)
	}
}

// AppendEntries adds older entries below existing ones and moves the
// cursor to the first newly-loaded entry.
func (p *LogPanel) AppendEntries(entries []svn.LogEntry) {
	if len(entries) == 0 {
		return
	}
	firstNew := len(p.entries)
	p.entries = append(p.entries, entries...)
	for _, e := range entries {
		p.list.AddItem(formatLogEntry(e), "", 0, nil)
	}
	p.list.SetCurrentItem(firstNew)
}

// OldestRevision returns the revision of the oldest loaded entry, or 0
// if the log is empty.
func (p *LogPanel) OldestRevision() int64 {
	if len(p.entries) == 0 {
		return 0
	}
	return p.entries[len(p.entries)-1].Revision
}

// SetPathMode switches the panel into/out of single-file log mode.
// Empty path returns to repo-wide mode. Entries are cleared; caller must
// supply new entries via SetEntries.
func (p *LogPanel) SetPathMode(path string) {
	p.path = path
	p.entries = nil
	p.list.Clear()
	if path == "" {
		p.list.SetTitle(" Log ")
	} else {
		p.list.SetTitle(" Log: " + path + " ")
	}
}

func formatLogEntry(e svn.LogEntry) string {
	msg := e.Message
	if len(msg) > 60 {
		msg = msg[:57] + "..."
	}
	return fmt.Sprintf("[cyan]r%d[-]  [grey]%s[-]  %s  %s",
		e.Revision, e.Author, e.Date.Format("01-02 15:04"), msg)
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
