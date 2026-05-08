# lazysvn — Design Spec

**Date:** 2026-05-08
**Status:** Approved, ready for implementation planning
**Supersedes:** `vim-svn` (VimScript plugin at github.com/maiyangzhan/vim-svn)

## Goal

A lazygit-style terminal UI for SVN, distributed as a single static Linux
binary. Works standalone in any terminal and is also launched from Vim via a
thin plugin (`:LazySvn`). Designed to be dropped onto an offline company Linux
server by scp, with no runtime installation and no Go toolchain on the target.

## Scope

### In scope (MVP)

Functional parity with the current `vim-svn` plugin:

- File-status panel grouped by Modified / Added / Deleted / Untracked / Conflicted.
- Log panel with auto-preview of revision diffs.
- Preview panel (right) auto-updating as the cursor moves in files/log.
- Hint bar (bottom) showing context-sensitive keys.
- Multi-mark with `Space`, batch operations across marked files.
- Keys: `d` diff, `c` commit, `r` revert, `a` add, `x` delete, `m` resolved,
  `R` refresh, `u` update, `q` quit.
- Vim integration via `:LazySvn` that opens the CLI in a terminal buffer.
- Open-file-in-editor (`o`/`Enter`) — new feature beyond the existing plugin;
  essential for the "stay in the TUI" workflow.

### Out of scope (MVP)

- Branch switching, stash, cherry-pick, blame, conflict-merge browser —
  deferred until MVP is in daily use.
- JSON/RPC mode, plugin system, theming.
- Multi-working-copy management in one session.
- Keeping the old VimScript UI. The old `svn/` and `svnfzf/` autoload trees are
  retired; users install the new `lazysvn` plugin from the new repo.

## Constraints

- **Target:** offline Linux x86_64 server. No internet, no package manager
  access, no Go toolchain.
- **Dev machine:** personal macOS. Cross-compiles with
  `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build`, producing a fully static
  ELF binary. Distribution is manual (user moves the binary via their own
  mechanism).
- **Runtime deps on server:** only the `svn` client (already present, the
  existing vim-svn plugin requires it).
- **Vim version:** 8.0+ (matches the old plugin). `popup_create` path requires
  8.2+ and falls back on older versions.

## Architecture

Three layers, strict one-way dependency (`cmd` → `ui` → `svn`):

```
+------------------------------------+
| cmd/lazysvn/main.go                |  flag parsing, DI, panic recovery
+------------------------------------+
| internal/ui    (tview)             |  three panels + hint bar + modals
+------------------------------------+
| internal/svn   (no UI)             |  shells out to `svn`, parses --xml
+------------------------------------+
```

### `internal/svn/` — data layer

Pure Go, no UI. All operations shell out to `svn --non-interactive` and parse
`--xml` output with `encoding/xml`. Parsing XML (not human-readable output) is
a deliberate choice — svn's text output drifts between versions, XML is a
stable contract.

**Exported surface:**

```go
type Client struct { cwd string }

type Status int
const (
    Modified Status = iota
    Added
    Deleted
    Untracked
    Conflicted
    // ...
)

type FileEntry struct {
    Path    string
    Status  Status
}

type LogEntry struct {
    Revision int64
    Author   string
    Date     time.Time
    Message  string
}

func (c *Client) Status() ([]FileEntry, error)
func (c *Client) Log(limit int) ([]LogEntry, error)
func (c *Client) Diff(path string) (string, error)
func (c *Client) DiffRevision(rev int64) (string, error)
func (c *Client) Commit(paths []string, msg string) error
func (c *Client) Revert(paths []string) error
func (c *Client) Add(paths []string) error
func (c *Client) Remove(paths []string) error
func (c *Client) Resolved(paths []string) error
func (c *Client) Update() (string, error)   // returns human summary
```

**Testing:** fixtures in `testdata/` hold recorded XML outputs for each
command; table-driven unit tests on the parsers. The `Client` itself (which
forks `svn`) is tested against a scripted mini working copy under
`testdata/repo/` built by a shell script in test setup — no mocks.

### `internal/ui/` — tview layer

No direct svn calls. Each panel receives callbacks wired by `app.go`.

**Files:**

- `app.go` — `tview.Application`, `Flex` root, wires panels, owns focus routing.
- `files.go` — files panel (top-left). State: entries, marks (set), cursor.
- `log.go` — log panel (bottom-left). State: entries, cursor.
- `preview.go` — preview panel (right). Passive; exposes `SetContent(text)`.
- `hints.go` — hint bar (bottom). Passive; exposes `Set(keys)`.
- `keymap.go` — one place for all key bindings; mirrors README table 1:1.
- `modal.go` — spinner modal, confirm modal, error toast.

**Panel contract:**

```go
type Panel interface {
    View() tview.Primitive
    Refresh() error
    Focus()
}
```

Panels **never call each other directly.** Example wiring in `app.go`:

```go
files.OnCursorChange = func(f FileEntry) {
    preview.SetContent(debouncedDiff(f.Path))
}
files.OnMark = func() { hints.Set(markedKeys) }
```

This keeps `files` ignorant of `preview`'s existence and makes each panel
unit-testable by driving its public methods directly.

**State stored per panel** is exposed as small structs (e.g. `filesState`)
that are the test surface — rendering is not unit-tested, the state machine
is.

### `cmd/lazysvn/main.go`

- Parse flags: `--cwd`, `--log-limit`, `--no-color`, `--config`.
- Load `~/.config/lazysvn/config.toml` if present; CLI flags override.
- Construct `svn.Client`, inject into `ui.App`, run.
- `defer recover()` so terminal state is restored on panic; stderr dump goes
  to `~/.cache/lazysvn/log`.

## Interaction details

### Commit message input

Primary path: press `c` → tview suspends → spawn `$EDITOR` on a temp file →
on editor exit, read file → `svn commit -F <tmpfile> <paths>`. Handles
multi-line messages natively and reuses the user's existing editor config.

Nested-vim case: when launched from inside Vim, the Vim plugin sets
`$VIM_SERVERNAME=v:servername`. The editor launcher (below) detects this and
routes through `vim --servername <name> --remote-wait-silent <tmpfile>` so the
edit happens in the host Vim instance, not a nested one.

### Open file in editor (`o` / `Enter`)

Same launcher as above. When focused on the files panel, `o` opens the
current file in the editor, suspending lazysvn; returning from the editor
resumes lazysvn and refreshes the file list.

### Editor launcher (`internal/editor/launch.go`)

```
if $VIM_SERVERNAME is non-empty:
    # non-interactive; host Vim handles the edit in another window.
    # Do NOT suspend tview — lazysvn's terminal buffer stays visible.
    run-and-wait: vim --servername $VIM_SERVERNAME --remote-wait-silent <path>
elif $EDITOR is non-empty:
    suspend tview (release terminal to child)
    run-and-wait: $EDITOR <path>
    resume tview (force full redraw)
else:
    same as $EDITOR branch with `vi`
```

### Error reporting

SVN command failures:

- Replace hint bar with a red error message for 3 seconds, then restore hints.
- Full stderr appended to `~/.cache/lazysvn/log` (no rotation — single file).
- No modal dialogs for errors; they block and annoy.

### Long-running operations

`u` (svn update) can block on network for seconds. UI contract:

- Show a spinner modal ("Updating…").
- Run `svn update` in a goroutine.
- On completion: post a `tview` event via `app.QueueUpdateDraw` to close the
  modal, show a brief toast with the result summary, refresh files & log.
- No cancellation (svn has no clean cancel). User can force-quit with
  `Ctrl-C` which kills the whole program.

### Preview auto-refresh debounce

On cursor move in files/log, queue a preview update with a 100ms debounce.
Only the last cursor position triggers `svn diff`. Prevents
diff-per-keystroke thrash during `j`/`k` holds.

## Vim plugin

Deliberately minimal. Lives in `vim-plugin/` of the lazysvn repo.

**`vim-plugin/plugin/lazysvn.vim`** (~15 lines):

```vim
if exists('g:loaded_lazysvn') | finish | endif
let g:loaded_lazysvn = 1

if !exists('g:lazysvn_cmd')
  let g:lazysvn_cmd = 'lazysvn'
endif

command! LazySvn call lazysvn#open()

nnoremap <silent> <Plug>LazySvn :LazySvn<CR>
if !hasmapto('<Plug>LazySvn') && !exists('g:lazysvn_no_default_mapping')
  nmap <silent> <Leader>s <Plug>LazySvn
endif
```

**`vim-plugin/autoload/lazysvn.vim`** does:

1. Set `$VIM_SERVERNAME = v:servername` (empty string if clientserver
   unavailable; the Go launcher falls through to `$EDITOR`).
2. Prefer `popup_create` + `term_start` (Vim 8.2+) for a floating terminal.
3. Fall back to `tabnew | terminal ++curwin ++close <cmd>` on older Vim.
4. When the underlying `lazysvn` process exits, the terminal buffer
   auto-closes (`++close`).

No user-facing autoload functions beyond `lazysvn#open()`.

## Configuration

**CLI — `~/.config/lazysvn/config.toml`** (all fields optional):

```toml
log_limit      = 50       # corresponds to old g:svn_log_limit
diff_highlight = true     # auto-detect delta/bat on $PATH
color_scheme   = "default"
```

**CLI flags** (override config file): `--log-limit`, `--no-color`, `--cwd`,
`--config`.

**Vim plugin variables:**

- `g:lazysvn_cmd` — path to the binary (default `"lazysvn"`).
- `g:lazysvn_no_default_mapping` — disable the `<Leader>s` mapping.

No other config for MVP.

## Repo layout

```
lazysvn/
├── cmd/lazysvn/main.go
├── internal/
│   ├── svn/
│   │   ├── client.go
│   │   ├── parser.go
│   │   └── parser_test.go
│   ├── ui/
│   │   ├── app.go
│   │   ├── files.go
│   │   ├── log.go
│   │   ├── preview.go
│   │   ├── hints.go
│   │   ├── keymap.go
│   │   ├── modal.go
│   │   └── state_test.go
│   ├── editor/
│   │   └── launch.go
│   └── config/
│       └── config.go
├── vim-plugin/
│   ├── plugin/lazysvn.vim
│   ├── autoload/lazysvn.vim
│   └── doc/lazysvn.txt
├── testdata/
│   ├── xml/                   # recorded `svn --xml` outputs
│   └── repo-setup.sh          # scripts a mini working copy for integration tests
├── Makefile
├── go.mod
├── go.sum
└── README.md
```

## Build & distribution

**Makefile targets:**

- `make build` — host build for dev on macOS.
- `make linux` — `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o dist/lazysvn-linux-amd64`.
- `make test` — unit tests + integration tests against the scripted repo.
- `make clean`.

**Distribution flow:** developer runs `make linux`, transfers
`dist/lazysvn-linux-amd64` to the target server via their own channel (manual
transit — not scripted). Target user drops it in `~/bin/lazysvn` and
`chmod +x`. No installer, no config required to start.

## Testing strategy

- **TDD** applied per the `test-driven-development` skill. Write failing tests
  first, then implementation.
- `internal/svn/parser_test.go` — table-driven on recorded XML fixtures. This
  is the critical correctness surface; exhaustive coverage here.
- `internal/svn/client_test.go` — integration against the scripted working
  copy under `testdata/repo/`. Exercises each `Client` method end-to-end
  against a real `svn`.
- `internal/ui/state_test.go` — state transitions of `filesState` and
  `logState` (mark/unmark, cursor move, refresh merge). UI rendering is NOT
  unit-tested.
- `internal/editor/launch_test.go` — editor selection logic with env-var
  fixtures. The actual exec is tested with a fake editor script (prints its
  arg, exits 0).
- Each milestone ends with `make linux` + manual scp + smoke test on a real
  server before starting the next milestone.

## Milestones

Each milestone produces a runnable binary and a manual verification pass.

**M1 · svn data layer.**
All methods on `Client` + XML parsing + fixture tests + scripted-repo
integration. No UI; a throwaway `main.go` prints `Status()` / `Log()` output
for manual verification.

**M2 · UI skeleton (read-only).**
tview three-panel Flex layout + focus routing + hint bar. Wires `Status()`
and `Log()` into files/log panels. `d` shows diff in preview with debounce.
No write operations yet. Ship and dogfood.

**M3 · Write operations.**
`c` / `r` / `a` / `x` / `m` / `Space` with multi-mark. Commit via `$EDITOR`.
Confirm modals for destructive ops (`r`, `x`).

**M4 · Long ops + error reporting.**
`u` with spinner modal and goroutine. Error toast on hint bar. Log file at
`~/.cache/lazysvn/log`. Preview debounce tuned.

**M5 · Vim plugin + packaging.**
`vim-plugin/` contents. Makefile `make linux`. README with usage, install,
nested-vim caveats. Tag v0.1.0.

## Non-goals / deferred

Listed explicitly so future scope creep gets pushed back to a follow-up:

- Branch / switch / merge UI.
- Blame view.
- Stash-equivalent (SVN has changelists; not for MVP).
- `svn cleanup` / `svn relocate` / `svn export`.
- Multi-repo dashboard.
- Windows/WSL support (target is Linux).

## Open questions

None blocking. Items deferred to implementation judgment:

- Exact color palette for diff highlighting (solve during M2 when we can see it).
- Spinner glyph set (pick during M4).
- Default `log_limit` (start at 50, matching the old plugin).
