package ui

import (
	"fmt"
	"sort"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/maiyangzhan/lazysvn/internal/svn"
)

type FilesPanel struct {
	list     *tview.List
	entries  []svn.FileEntry
	marks    map[string]bool
	OnSelect func(entry svn.FileEntry)
	OnAction func(key rune)
}

func NewFilesPanel() *FilesPanel {
	p := &FilesPanel{marks: map[string]bool{}}
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
		case ' ':
			p.ToggleMark()
			return nil
		case 'c', 'r', 'a', 'x', 'm', 'e':
			if p.OnAction != nil {
				p.OnAction(event.Rune())
			}
			return nil
		}
		return event
	})
	p.list = l
	return p
}

func (p *FilesPanel) View() tview.Primitive {
	return p.list
}

func (p *FilesPanel) ToggleMark() {
	idx := p.list.GetCurrentItem()
	if idx < 0 || idx >= len(p.entries) {
		return
	}
	path := p.entries[idx].Path
	if p.marks[path] {
		delete(p.marks, path)
	} else {
		p.marks[path] = true
	}
	p.renderItem(idx)
	// move cursor down after toggling
	if idx+1 < len(p.entries) {
		p.list.SetCurrentItem(idx + 1)
	}
}

func (p *FilesPanel) ClearMarks() {
	p.marks = map[string]bool{}
	for i := range p.entries {
		p.renderItem(i)
	}
}

func (p *FilesPanel) HasMarks() bool {
	return len(p.marks) > 0
}

func (p *FilesPanel) MarkedOrCurrent() []svn.FileEntry {
	if len(p.marks) > 0 {
		var result []svn.FileEntry
		for _, e := range p.entries {
			if p.marks[e.Path] {
				result = append(result, e)
			}
		}
		return result
	}
	if e := p.SelectedEntry(); e != nil {
		return []svn.FileEntry{*e}
	}
	return nil
}

func (p *FilesPanel) SetEntries(entries []svn.FileEntry) {
	sort.Slice(entries, func(i, j int) bool {
		oi, oj := statusOrder(entries[i].Status), statusOrder(entries[j].Status)
		if oi != oj {
			return oi < oj
		}
		return entries[i].Path < entries[j].Path
	})
	curPath := ""
	if prev := p.SelectedEntry(); prev != nil {
		curPath = prev.Path
	}
	prevIdx := p.list.GetCurrentItem()
	p.entries = entries
	p.list.Clear()
	// prune marks for entries that no longer exist
	valid := map[string]bool{}
	for _, e := range entries {
		valid[e.Path] = true
	}
	for path := range p.marks {
		if !valid[path] {
			delete(p.marks, path)
		}
	}
	for _, e := range entries {
		mark := " "
		if p.marks[e.Path] {
			mark = "◆"
		}
		label := fmt.Sprintf("%s [%s]%s[-]  %s", mark, statusColor(e.Status), statusLetter(e.Status), e.Path)
		p.list.AddItem(label, "", 0, nil)
	}
	newIdx := -1
	if curPath != "" {
		for i, e := range entries {
			if e.Path == curPath {
				newIdx = i
				break
			}
		}
	}
	if newIdx < 0 {
		newIdx = prevIdx
		if newIdx >= len(entries) {
			newIdx = len(entries) - 1
		}
	}
	if newIdx >= 0 {
		p.list.SetCurrentItem(newIdx)
	}
}

func (p *FilesPanel) renderItem(idx int) {
	if idx < 0 || idx >= len(p.entries) {
		return
	}
	e := p.entries[idx]
	mark := " "
	if p.marks[e.Path] {
		mark = "◆"
	}
	label := fmt.Sprintf("%s [%s]%s[-]  %s", mark, statusColor(e.Status), statusLetter(e.Status), e.Path)
	p.list.SetItemText(idx, label, "")
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

func statusOrder(s svn.Status) int {
	switch s {
	case svn.Conflicted:
		return 0
	case svn.Modified:
		return 1
	case svn.Added:
		return 2
	case svn.Deleted:
		return 3
	case svn.Untracked:
		return 4
	default:
		return 5
	}
}
