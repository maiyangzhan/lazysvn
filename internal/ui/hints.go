package ui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Hint struct {
	Key   string
	Label string
}

type HintBar struct {
	view      *tview.TextView
	app       *tview.Application
	lastHints []Hint
	mu        sync.Mutex
	timer     *time.Timer
}

func NewHintBar(app *tview.Application) *HintBar {
	tv := tview.NewTextView()
	tv.SetDynamicColors(true)
	tv.SetBackgroundColor(tcell.ColorDarkSlateGray)
	tv.SetTextAlign(tview.AlignLeft)
	return &HintBar{view: tv, app: app}
}

func (h *HintBar) View() tview.Primitive {
	return h.view
}

func (h *HintBar) Set(hints []Hint) {
	h.mu.Lock()
	h.lastHints = hints
	h.mu.Unlock()
	h.renderHints(hints)
}

func (h *HintBar) ShowError(msg string) {
	h.mu.Lock()
	if h.timer != nil {
		h.timer.Stop()
	}
	saved := h.lastHints
	h.mu.Unlock()

	h.view.SetText(" [red::b]" + tview.Escape(msg) + "[-::-]")

	h.mu.Lock()
	h.timer = time.AfterFunc(3*time.Second, func() {
		h.app.QueueUpdateDraw(func() {
			h.renderHints(saved)
		})
	})
	h.mu.Unlock()
}

func (h *HintBar) ShowInfo(msg string) {
	h.mu.Lock()
	if h.timer != nil {
		h.timer.Stop()
	}
	saved := h.lastHints
	h.mu.Unlock()

	h.view.SetText(" [green::b]" + tview.Escape(msg) + "[-::-]")

	h.mu.Lock()
	h.timer = time.AfterFunc(3*time.Second, func() {
		h.app.QueueUpdateDraw(func() {
			h.renderHints(saved)
		})
	})
	h.mu.Unlock()
}

func (h *HintBar) renderHints(hints []Hint) {
	var parts []string
	for _, hint := range hints {
		parts = append(parts, fmt.Sprintf("[yellow]%s[white]: %s", hint.Key, hint.Label))
	}
	h.view.SetText(" " + strings.Join(parts, "  "))
}
