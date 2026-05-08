package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/maiyangzhan/lazysvn/internal/svn"
)

type LogPanel struct {
	list     *tview.List
	entries  []svn.LogEntry
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
