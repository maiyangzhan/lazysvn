package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type PreviewPanel struct {
	view *tview.TextView

	rawText     string
	searchTerm  string
	matchCount  int
	currentHit  int

	OnSearchPrompt func()   // user pressed '/' while preview is focused
	OnSearchClear  func()   // user pressed Esc while a search is active
}

func NewPreviewPanel() *PreviewPanel {
	tv := tview.NewTextView()
	tv.SetDynamicColors(true)
	tv.SetRegions(true)
	tv.SetBorder(true)
	tv.SetTitle(" Preview ")
	tv.SetTitleColor(tcell.ColorWhite)
	tv.SetScrollable(true)
	tv.SetWrap(false)
	p := &PreviewPanel{view: tv}
	tv.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case '/':
			if p.OnSearchPrompt != nil {
				p.OnSearchPrompt()
			}
			return nil
		case 'n':
			p.NextMatch()
			return nil
		case 'N':
			p.PrevMatch()
			return nil
		}
		if event.Key() == tcell.KeyEscape && p.searchTerm != "" {
			p.ClearSearch()
			if p.OnSearchClear != nil {
				p.OnSearchClear()
			}
			return nil
		}
		return event
	})
	return p
}

func (p *PreviewPanel) View() tview.Primitive {
	return p.view
}

func (p *PreviewPanel) Focus() {
	p.view.SetBorderColor(tcell.ColorBlue)
}

func (p *PreviewPanel) Blur() {
	p.view.SetBorderColor(tcell.ColorWhite)
}

func (p *PreviewPanel) ScrollHalfPageDown() {
	_, _, _, h := p.view.GetInnerRect()
	row, col := p.view.GetScrollOffset()
	p.view.ScrollTo(row+h/2, col)
}

func (p *PreviewPanel) ScrollHalfPageUp() {
	_, _, _, h := p.view.GetInnerRect()
	row, col := p.view.GetScrollOffset()
	newRow := row - h/2
	if newRow < 0 {
		newRow = 0
	}
	p.view.ScrollTo(newRow, col)
}

func (p *PreviewPanel) SetContent(text string) {
	p.rawText = text
	// Preserve any active search across content updates so the user
	// can navigate panel selections while keeping their query alive.
	p.applyContent()
	if p.searchTerm == "" || p.matchCount == 0 {
		p.view.ScrollToBeginning()
	} else {
		// keep current hit if still valid; otherwise jump to first.
		if p.currentHit < 0 || p.currentHit >= p.matchCount {
			p.currentHit = 0
		}
		p.view.Highlight(fmt.Sprintf("m%d", p.currentHit)).ScrollToHighlight()
	}
}

// SetSearch applies a substring search (case-insensitive). Empty term
// clears the search. Returns the number of matches found.
func (p *PreviewPanel) SetSearch(term string) int {
	p.searchTerm = term
	p.applyContent()
	p.currentHit = 0
	if term == "" || p.matchCount == 0 {
		p.view.Highlight()
		return p.matchCount
	}
	p.view.Highlight(fmt.Sprintf("m%d", p.currentHit)).ScrollToHighlight()
	return p.matchCount
}

func (p *PreviewPanel) ClearSearch() {
	p.SetSearch("")
}

// SearchTerm returns the active search query, "" when none.
func (p *PreviewPanel) SearchTerm() string {
	return p.searchTerm
}

// MatchInfo returns (current, total) with current 1-based; (0, 0) when
// there is no active search or no matches.
func (p *PreviewPanel) MatchInfo() (int, int) {
	if p.searchTerm == "" || p.matchCount == 0 {
		return 0, 0
	}
	return p.currentHit + 1, p.matchCount
}

func (p *PreviewPanel) NextMatch() {
	if p.searchTerm == "" || p.matchCount == 0 {
		return
	}
	p.currentHit = (p.currentHit + 1) % p.matchCount
	p.view.Highlight(fmt.Sprintf("m%d", p.currentHit)).ScrollToHighlight()
}

func (p *PreviewPanel) PrevMatch() {
	if p.searchTerm == "" || p.matchCount == 0 {
		return
	}
	p.currentHit = (p.currentHit - 1 + p.matchCount) % p.matchCount
	p.view.Highlight(fmt.Sprintf("m%d", p.currentHit)).ScrollToHighlight()
}

func (p *PreviewPanel) applyContent() {
	rendered, n := renderDiff(p.rawText, p.searchTerm)
	p.matchCount = n
	p.view.Clear()
	p.view.SetText(rendered)
}

// renderDiff colorizes a diff and, if searchTerm is non-empty, wraps
// each case-insensitive match in a unique tview region tag ("m0", "m1",
// ...). Returns the rendered text and the number of matches.
func renderDiff(text, searchTerm string) (string, int) {
	if text == "" {
		return "", 0
	}
	lowerTerm := strings.ToLower(searchTerm)
	var buf strings.Builder
	matchCount := 0
	for _, line := range strings.Split(text, "\n") {
		openTag, closeTag := diffLineColor(line)
		if searchTerm == "" {
			buf.WriteString(openTag)
			buf.WriteString(tview.Escape(line))
			buf.WriteString(closeTag)
			buf.WriteByte('\n')
			continue
		}
		buf.WriteString(openTag)
		lowerLine := strings.ToLower(line)
		cursor := 0
		for {
			i := strings.Index(lowerLine[cursor:], lowerTerm)
			if i < 0 {
				buf.WriteString(tview.Escape(line[cursor:]))
				break
			}
			abs := cursor + i
			buf.WriteString(tview.Escape(line[cursor:abs]))
			// close active color so the region highlight reads cleanly,
			// then reopen after the match.
			buf.WriteString(closeTag)
			buf.WriteString(fmt.Sprintf(`["m%d"]`, matchCount))
			buf.WriteString(tview.Escape(line[abs : abs+len(searchTerm)]))
			buf.WriteString(`[""]`)
			buf.WriteString(openTag)
			matchCount++
			cursor = abs + len(searchTerm)
		}
		buf.WriteString(closeTag)
		buf.WriteByte('\n')
	}
	return buf.String(), matchCount
}

// diffLineColor returns the (open, close) tview color tags appropriate
// for a diff line — driven by the leading marker characters.
func diffLineColor(line string) (string, string) {
	switch {
	case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
		return "[white::b]", "[-::-]"
	case strings.HasPrefix(line, "+"):
		return "[green]", "[-]"
	case strings.HasPrefix(line, "-"):
		return "[red]", "[-]"
	case strings.HasPrefix(line, "@@"):
		return "[cyan]", "[-]"
	case strings.HasPrefix(line, "Index:") || strings.HasPrefix(line, "====="):
		return "[grey]", "[-]"
	default:
		return "", ""
	}
}
