# lazysvn

A [lazygit](https://github.com/jesseduffield/lazygit)-style terminal UI for Subversion (SVN).

Browse file status, view diffs, commit, revert, and update — all without leaving the terminal. Ships as a single static binary with zero dependencies beyond `svn` itself.

![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)
![Platform](https://img.shields.io/badge/Platform-Linux%20x86__64-lightgrey)
![License](https://img.shields.io/badge/License-MIT-blue)

## Features

- **Three-panel layout** — file status, commit log, and diff preview side by side
- **Vim-style navigation** — `j`/`k`, `g`/`G`, `Ctrl-U`/`Ctrl-D` for preview scrolling
- **File operations** — commit, revert, add, delete, all with single-key shortcuts
- **Multi-select** — `Space` to mark multiple files for batch operations
- **Live diff preview** — auto-updates as you navigate, with syntax coloring and per-path caching
- **Conflict resolution** — pick `mine-conflict` / `theirs-conflict` / `mine-full` / `theirs-full` from a modal when resolving
- **In-app editor launch** — `e` opens `$EDITOR` on the current file (honors `VIM_SERVERNAME`); UI returns and refreshes automatically
- **Non-blocking writes** — commits, reverts, updates run in the background with a spinner, never freezing the TUI
- **SVN update** — `u` to update with a progress indicator
- **Status grouping** — files grouped by status (Conflicted > Modified > Added > Deleted > Untracked); selection preserved by path across refresh
- **Vim integration** — optional `:LazySvn` command to launch from inside Vim
- **Offline-friendly** — single static binary, just `scp` to your server and run

## Install

### Download prebuilt binary

Grab the latest release from the [Releases](https://github.com/maiyangzhan/lazysvn/releases) page:

```bash
curl -LO https://github.com/maiyangzhan/lazysvn/releases/latest/download/lazysvn-linux-amd64
chmod +x lazysvn-linux-amd64
sudo mv lazysvn-linux-amd64 /usr/local/bin/lazysvn
```

### Build from source

Requires Go 1.22+.

```bash
git clone https://github.com/maiyangzhan/lazysvn.git
cd lazysvn
make build          # build for current platform
make linux          # cross-compile for Linux x86_64 (static binary)
```

The output binary is at `dist/lazysvn-linux-amd64` (cross-compiled) or `./lazysvn` (native).

## Usage

```bash
lazysvn                     # run in current SVN working copy
lazysvn --cwd /path/to/wc   # specify a different working copy
lazysvn --log-limit 100     # show more log entries (default: 50)
```

## Layout

```
┌─────── Files ────────┬──────── Preview ──────────┐
│  M  src/main.go      │ --- a/src/main.go         │
│  M  src/util.go      │ +++ b/src/main.go         │
│  A  src/new.go       │ @@ -10,3 +10,5 @@        │
├─────── Log ──────────┤ +func newHelper() {       │
│  r42  alice  05-08   │ +    return nil            │
│  r41  bob    05-07   │ +}                         │
├──────────────────────┴───────────────────────────┤
│ j/k:nav ^u/^d:scroll Space:mark c:commit q:quit │
└──────────────────────────────────────────────────┘
```

## Key Bindings

### Navigation

| Key | Action |
|---|---|
| `j` / `k` | Move cursor down / up |
| `g` / `G` | Jump to first / last item |
| `Ctrl-U` / `Ctrl-D` | Scroll preview half-page up / down |
| `Tab` / `Shift-Tab` | Switch focus between Files and Log panels |

### File Operations (Files panel)

| Key | Action |
|---|---|
| `Space` | Toggle mark on current file |
| `c` | Commit marked/current file(s) |
| `r` | Revert marked/current file(s) (with confirmation) |
| `a` | Add untracked file(s) to version control |
| `x` | Delete file(s) (with confirmation) |
| `e` | Open current file in `$EDITOR` (or `vi`); honors `VIM_SERVERNAME` |
| `m` | Resolve conflict(s): pick `mine-conflict` / `theirs-conflict` / `mine-full` / `theirs-full` / mark resolved |

### Global

| Key | Action |
|---|---|
| `u` | Run `svn update` (with progress indicator) |
| `R` | Refresh all panels |
| `q` | Quit |

## Vim Integration

An optional Vim plugin lets you launch lazysvn in a terminal buffer without leaving your editor.

### Install the plugin

With [vim-plug](https://github.com/junegunn/vim-plug):

```vim
Plug 'maiyangzhan/lazysvn', { 'rtp': 'vim-plugin' }
```

Or manually:

```bash
cp -r vim-plugin/* ~/.vim/
```

### Usage

- `:LazySvn` or `<Leader>s` to open
- Uses a floating popup on Vim 8.2+, falls back to a new tab on older versions
- Automatically closes when lazysvn exits

### Configuration

| Variable | Default | Description |
|---|---|---|
| `g:lazysvn_cmd` | `"lazysvn"` | Path to the lazysvn binary |
| `g:lazysvn_no_default_mapping` | (unset) | Set to `1` to disable the `<Leader>s` mapping |

## Requirements

- **Runtime:** `svn` command-line client on `$PATH`
- **Build:** Go 1.22+
- **Platform:** Linux x86_64 (prebuilt), macOS (build from source)

## License

MIT
