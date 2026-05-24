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

// ShowSpinner displays a modal that blocks user input until the returned
// dismiss function is called. Press Esc on the spinner to invoke onCancel
// (typically cancels the context driving the running subprocess).
//
// ShowSpinner must be called from the tview main goroutine (i.e. from an
// input handler). It sets the root directly rather than via QueueUpdateDraw
// because tview.Application.QueueUpdate blocks on the main loop — calling
// it from the main loop deadlocks. The dismiss function IS called from a
// background goroutine, so it uses QueueUpdateDraw.
func ShowSpinner(app *tview.Application, root tview.Primitive, message string, onCancel func()) func() {
	modal := tview.NewModal().
		SetText(message + "\n\n(Esc to cancel)")
	modal.SetBackgroundColor(tcell.ColorDarkSlateGray)
	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			if onCancel != nil {
				onCancel()
			}
			return nil
		}
		return event
	})
	app.SetRoot(modal, true)
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
	hint.SetText("[yellow]Enter[white]: submit  [yellow]Esc[white]: cancel  [grey](C for $EDITOR)[-]")
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

// PathPrompt asks for a path to filter the log panel on. Empty pattern
// means exit path mode. onDone receives the trimmed path and whether
// the user cancelled via Esc.
func PathPrompt(app *tview.Application, root tview.Primitive, initial string, onDone func(path string, cancelled bool)) {
	input := tview.NewInputField()
	input.SetLabel("Path: ")
	input.SetFieldWidth(46)
	input.SetLabelColor(tcell.ColorYellow)
	input.SetText(initial)

	hint := tview.NewTextView()
	hint.SetDynamicColors(true)
	hint.SetTextAlign(tview.AlignCenter)
	hint.SetText("[yellow]Enter[white]: apply  [yellow]Esc[white]: cancel  [grey](empty = exit path mode)[-]")
	hint.SetBackgroundColor(tcell.ColorDarkSlateGray)

	frame := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(input, 1, 0, true).
		AddItem(nil, 0, 1, false).
		AddItem(hint, 1, 0, false)
	frame.SetBorder(true)
	frame.SetTitle(" Single-file Log ")
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
			p := strings.TrimSpace(input.GetText())
			app.SetRoot(root, true)
			onDone(p, false)
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

// HelpModal shows a full keybinding reference. Calls onClose when
// dismissed (Esc / ? / q). Caller is responsible for tracking modalActive.
func HelpModal(app *tview.Application, root tview.Primitive, onClose func()) {
	const content = `[yellow::b]Navigation[-::-]
  j / k              move cursor down / up
  g / G              jump to first / last item
  Ctrl-U / Ctrl-D    scroll preview half-page up / down
  Tab / Shift-Tab    cycle focus: Files → Log → Preview
  Mouse              click to focus a panel; wheel to scroll

[yellow::b]Files panel[-::-]
  Space              toggle mark on current file
  /                  fuzzy-find a file (fzf if available; jumps cursor)
                       fallback: text filter that narrows the panel
  c                  commit (single-line prompt)
  C                  commit via $EDITOR (multi-line)
  r                  revert (with confirmation)
  a                  add untracked to version control
  x                  delete (with confirmation)
  X                  fzf-pick any path (incl. clean files / dirs) and
                       svn-rm it; --multi for batch
  e                  open current file in $EDITOR
  m                  resolve conflict(s): mine / theirs / mark
  L                  single-file log for current item (toggle)

[yellow::b]Log panel[-::-]
  M                  load more older entries
  L                  single-file log for an arbitrary path
                       (fuzzy-picks via fzf when available;
                        otherwise a text prompt)
  Esc                exit single-file log mode

[yellow::b]Preview panel[-::-]
  /                  search the diff (case-insensitive substring)
  n / N              jump to next / previous match
  Esc                clear active search

[yellow::b]Global[-::-]
  u                  svn update (spinner; Esc to cancel; warns on conflicts)
  R                  refresh status + log
  ?                  show this help
  q                  quit

[yellow::b]Spinner (during long svn operations)[-::-]
  Esc                cancel the running subprocess

[grey]Operations are logged to ~/.cache/lazysvn/log for debugging.[-]
[grey]Press Esc, ? or q to close[-]`

	tv := tview.NewTextView()
	tv.SetDynamicColors(true)
	tv.SetScrollable(true)
	tv.SetWrap(false)
	tv.SetText(content)
	tv.SetBackgroundColor(tcell.ColorDarkSlateGray)
	tv.SetBorder(true)
	tv.SetTitle(" Help ")
	tv.SetTitleColor(tcell.ColorYellow)
	tv.SetBorderColor(tcell.ColorBlue)

	close := func() {
		app.SetRoot(root, true)
		if onClose != nil {
			onClose()
		}
	}

	tv.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || event.Rune() == '?' || event.Rune() == 'q' {
			close()
			return nil
		}
		return event
	})

	wrapper := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(tv, 64, 0, true).
			AddItem(nil, 0, 1, false),
			28, 0, true).
		AddItem(nil, 0, 1, false)

	app.SetRoot(wrapper, true)
	app.SetFocus(tv)
}

// FilterPrompt asks for a substring to filter the files panel by path.
// Empty pattern means clear filter. onDone receives the trimmed pattern
// and whether the user cancelled via Esc.
func FilterPrompt(app *tview.Application, root tview.Primitive, initial string, onDone func(pattern string, cancelled bool)) {
	input := tview.NewInputField()
	input.SetLabel("Filter: ")
	input.SetFieldWidth(46)
	input.SetLabelColor(tcell.ColorYellow)
	input.SetText(initial)

	hint := tview.NewTextView()
	hint.SetDynamicColors(true)
	hint.SetTextAlign(tview.AlignCenter)
	hint.SetText("[yellow]Enter[white]: apply  [yellow]Esc[white]: cancel  [grey](empty = clear)[-]")
	hint.SetBackgroundColor(tcell.ColorDarkSlateGray)

	frame := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(input, 1, 0, true).
		AddItem(nil, 0, 1, false).
		AddItem(hint, 1, 0, false)
	frame.SetBorder(true)
	frame.SetTitle(" Filter Files ")
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
			p := strings.TrimSpace(input.GetText())
			app.SetRoot(root, true)
			onDone(p, false)
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


// SearchPrompt asks for a substring to search within the Preview panel.
// Empty pattern means clear search. onDone receives the trimmed pattern
// and whether the user cancelled via Esc.
func SearchPrompt(app *tview.Application, root tview.Primitive, initial string, onDone func(pattern string, cancelled bool)) {
	input := tview.NewInputField()
	input.SetLabel("Search: ")
	input.SetFieldWidth(46)
	input.SetLabelColor(tcell.ColorYellow)
	input.SetText(initial)

	hint := tview.NewTextView()
	hint.SetDynamicColors(true)
	hint.SetTextAlign(tview.AlignCenter)
	hint.SetText("[yellow]Enter[white]: search  [yellow]Esc[white]: cancel  [grey](empty = clear)[-]")
	hint.SetBackgroundColor(tcell.ColorDarkSlateGray)

	frame := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(input, 1, 0, true).
		AddItem(nil, 0, 1, false).
		AddItem(hint, 1, 0, false)
	frame.SetBorder(true)
	frame.SetTitle(" Search Diff ")
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
			p := strings.TrimSpace(input.GetText())
			app.SetRoot(root, true)
			onDone(p, false)
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
