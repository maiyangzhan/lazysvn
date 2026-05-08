package ui

import (
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
