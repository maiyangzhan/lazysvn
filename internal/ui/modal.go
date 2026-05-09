package ui

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func Confirm(app *tview.Application, root tview.Primitive, message string, onYes func()) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"Yes", "No"}).
		SetDoneFunc(func(_ int, label string) {
			app.SetRoot(root, true)
			if label == "Yes" {
				onYes()
			}
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

func CommitPrompt(app *tview.Application, root tview.Primitive, onSubmit func(msg string)) {
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
			if msg != "" {
				onSubmit(msg)
			}
			return
		}
		if key == tcell.KeyEscape {
			app.SetRoot(root, true)
		}
	})

	app.SetRoot(wrapper, true)
	app.SetFocus(input)
}
