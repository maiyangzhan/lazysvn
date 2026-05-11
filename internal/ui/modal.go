package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func Confirm(app *tview.Application, root tview.Primitive, message string, onDone func(yes bool)) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"Yes", "No"}).
		SetDoneFunc(func(_ int, label string) {
			app.SetRoot(root, true)
			onDone(label == "Yes")
		})
	modal.SetBackgroundColor(tcell.ColorDarkSlateGray)
	app.SetRoot(modal, true)
}

func ShowSpinner(app *tview.Application, root tview.Primitive, message string) func() {
	modal := tview.NewModal().
		SetText(message)
	modal.SetBackgroundColor(tcell.ColorDarkSlateGray)
	app.QueueUpdateDraw(func() {
		app.SetRoot(modal, true)
	})
	return func() {
		app.QueueUpdateDraw(func() {
			app.SetRoot(root, true)
		})
	}
}

// ResolvePrompt shows a modal asking the user how to resolve conflict(s)
// on the selected file(s). onDone receives the svn --accept mode (empty
// string means cancelled).
func ResolvePrompt(app *tview.Application, root tview.Primitive, count int, onDone func(mode string)) {
	labels := []string{
		"Mark resolved (working)",
		"Mine (conflict hunks)",
		"Theirs (conflict hunks)",
		"Mine (full file)",
		"Theirs (full file)",
		"Cancel",
	}
	modes := []string{
		"working",
		"mine-conflict",
		"theirs-conflict",
		"mine-full",
		"theirs-full",
		"",
	}
	text := fmt.Sprintf("Resolve %d file(s) — pick strategy:", count)
	modal := tview.NewModal().
		SetText(text).
		AddButtons(labels).
		SetDoneFunc(func(idx int, _ string) {
			app.SetRoot(root, true)
			if idx < 0 || idx >= len(modes) {
				onDone("")
				return
			}
			onDone(modes[idx])
		})
	modal.SetBackgroundColor(tcell.ColorDarkSlateGray)
	app.SetRoot(modal, true)
}

func CommitPrompt(app *tview.Application, root tview.Primitive, onDone func(msg string, cancelled bool)) {
	input := tview.NewInputField()
	input.SetLabel("Message: ")
	input.SetFieldWidth(46)
	input.SetLabelColor(tcell.ColorYellow)

	hint := tview.NewTextView()
	hint.SetDynamicColors(true)
	hint.SetTextAlign(tview.AlignCenter)
	hint.SetText("[yellow]Enter[white]: submit  [yellow]Esc[white]: cancel")
	hint.SetBackgroundColor(tcell.ColorDarkSlateGray)

	frame := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(input, 1, 0, true).
		AddItem(nil, 0, 1, false).
		AddItem(hint, 1, 0, false)
	frame.SetBorder(true)
	frame.SetTitle(" Commit ")
	frame.SetTitleColor(tcell.ColorYellow)
	frame.SetBorderColor(tcell.ColorBlue)

	wrapper := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(frame, 60, 0, true).
			AddItem(nil, 0, 1, false),
			6, 0, true).
		AddItem(nil, 0, 1, false)

	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			msg := strings.TrimSpace(input.GetText())
			app.SetRoot(root, true)
			onDone(msg, false)
			return
		}
		if key == tcell.KeyEscape {
			app.SetRoot(root, true)
			onDone("", true)
		}
	})

	app.SetRoot(wrapper, true)
	app.SetFocus(input)
}
