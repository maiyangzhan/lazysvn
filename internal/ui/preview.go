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
