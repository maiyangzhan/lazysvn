# lazysvn

A [lazygit](https://github.com/jesseduffield/lazygit)-style terminal UI for Subversion (SVN).

Browse file status, view diffs, commit, revert, and update — all without leaving the terminal. Ships as a single static binary with zero dependencies beyond `svn` itself.

![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)
![Platform](https://img.shields.io/badge/Platform-Linux%20%7C%20macOS-lightgrey)
![License](https://img.shields.io/badge/License-MIT-blue)
[![ci](https://github.com/maiyangzhan/lazysvn/actions/workflows/ci.yml/badge.svg)](https://github.com/maiyangzhan/lazysvn/actions/workflows/ci.yml)

## Features

- **Three-panel layout** — file status, commit log, and diff preview side by side
- **Mouse support** — click any panel to focus it, wheel to scroll. (Most terminals require Shift-click or Option-drag to copy text once mouse mode is active.) Set `LAZYSVN_NO_MOUSE=1` to disable mouse mode entirely if your terminal misbehaves.
- **Vim-style navigation** — `j`/`k`, `g`/`G`, `Ctrl-U`/`Ctrl-D` for preview scrolling
- **Diff search** — focus the Preview pane (`Tab` cycles Files → Log → Preview, or click it), then `/` to search the diff text; `n`/`N` jump between matches, `Esc` clears
- **File operations** — commit, revert, add, delete, all with single-key shortcuts
- **Multi-select** — `Space` to mark multiple files for batch operations
- **File filter / fuzzy find** — `/` opens an `fzf` picker over the panel and jumps the cursor to the selected entry; falls back to a substring filter when fzf isn't on `$PATH`
- **Directory operations** — the Files panel also shows synthesized directory rows (path ends with `/`, status = worst child); marking a directory and pressing `c` / `r` / `a` / `x` / `m` operates on the whole subtree (`svn revert` / `svn resolve` get `--depth=infinity` automatically). `X` fuzzy-picks **any** path in the working copy (clean or dirty, files or dirs) for `svn rm`.
- **Single-file log drill-down** — `L` on a file shows only its history and scopes the preview to that file's changes at each revision; `M` loads more older entries; in the Log panel, `L` opens `fzf` (when available) to fuzzy-pick any path in the working copy, including files with no pending changes
- **Live diff preview** — auto-updates as you navigate, with syntax coloring and per-path caching
- **Conflict resolution** — pick `mine-conflict` / `theirs-conflict` / `mine-full` / `theirs-full` from a modal when resolving
- **In-app editor launch** — `e` opens `$EDITOR` on the current file (honors `VIM_SERVERNAME`); UI returns and refreshes automatically
- **Multi-line commit** — `c` for a quick one-liner, `C` to compose in `$EDITOR`
- **Non-blocking writes** — commits, reverts, updates run in the background with a spinner, never freezing the TUI
- **SVN update** — `u` to update with a progress indicator; warns when conflicts are produced
- **Status grouping** — files grouped by status (Conflicted > Modified > Added > Deleted > Untracked); selection preserved by path across refresh
- **Built-in help** — `?` shows the full keybinding reference
- **Vim integration** — optional `:LazySvn` command to launch from inside Vim
- **Offline-friendly** — single static binary, just `scp` to your server and run

## Install

### Download prebuilt binary

Grab the latest release from the [Releases](https://github.com/maiyangzhan/lazysvn/releases) page. Prebuilt binaries are published for:

| OS | Arch | Asset |
|---|---|---|
| Linux | x86_64 | `lazysvn-linux-amd64` |
| Linux | arm64  | `lazysvn-linux-arm64` |
| macOS | Intel  | `lazysvn-darwin-amd64` |
| macOS | Apple Silicon | `lazysvn-darwin-arm64` |

```bash
# pick the asset that matches your platform
curl -LO https://github.com/maiyangzhan/lazysvn/releases/latest/download/lazysvn-linux-amd64
chmod +x lazysvn-linux-amd64
sudo mv lazysvn-linux-amd64 /usr/local/bin/lazysvn
```

Each asset has a matching `.sha256` file for integrity verification.

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
lazysvn --log-limit 100     # initial log entry count (default: 50; M loads more)
lazysvn --version           # print version and exit
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
| `Tab` / `Shift-Tab` | Cycle focus: Files → Log → Preview |
| Mouse click | Focus the clicked panel |
| Mouse wheel | Scroll the panel under the cursor |

### File Operations (Files panel)

| Key | Action |
|---|---|
| `Space` | Toggle mark on current entry (file or directory) |
| `/` | Fuzzy-find an entry across the whole panel via `fzf` (jumps cursor to the picked entry). Falls back to a text-substring filter that narrows the panel when fzf isn't on `$PATH`. |
| `c` | Commit marked/current entry/entries — works on directories too |
| `C` | Commit via `$EDITOR` — multi-line message |
| `r` | Revert marked/current entry/entries (with confirmation; recurses into directories via `--depth=infinity`) |
| `a` | Add untracked entry/entries to version control (recurses into directories) |
| `x` | Delete entry/entries (with confirmation) |
| `X` | Fuzzy-pick any path in the working copy via `fzf` (files **and** directories, even ones with no pending changes) and `svn rm` it. Tab in fzf to multi-select for batch removal. Requires fzf on `$PATH`. |
| `e` | Open current file in `$EDITOR` (or `vi`); honors `VIM_SERVERNAME` |
| `m` | Resolve conflict(s): pick `mine-conflict` / `theirs-conflict` / `mine-full` / `theirs-full` / mark resolved (recurses into directories) |
| `L` | Toggle single-file log for the current item (toggle) |

### Log panel

| Key | Action |
|---|---|
| `M` | Load more older log entries |
| `L` | Pick any path in the working copy and drill into its single-file log. When `fzf` is on `$PATH`, fzf streams candidates from `$LAZYSVN_FZF_CMD` (if set) or from the fastest available of `fd` → `fdfind` → `find` — all of which emit both files and directories. The defaults pass `--no-ignore-vcs` so an upstream `.gitignore` doesn't hide everything in an SVN WC, while `.ignore` / `.fdignore` files **are** respected. Your shell's `$FZF_DEFAULT_COMMAND` is intentionally ignored to avoid git-oriented recipes hiding your SVN files; set `$LAZYSVN_FZF_CMD` for a lazysvn-specific override. When fzf is missing, falls back to a plain text prompt. |
| `Esc` | Exit single-file log mode (return to repo-wide log) |

### Preview panel

| Key | Action |
|---|---|
| `/` | Search the diff (case-insensitive substring). Matches are highlighted; the first one is scrolled into view. |
| `n` / `N` | Jump to next / previous match |
| `Esc` | Clear the active search |

### Global

| Key | Action |
|---|---|
| `u` | Run `svn update` (spinner; warns if conflicts are produced) |
| `R` | Refresh all panels |
| `?` | Show the full keybinding help |
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
| `g:lazysvn_vim_remote` | (unset) | Set to `1` to propagate `v:servername` so lazysvn's `C` / `e` open files in the parent vim via `--remote-wait-silent`. Requires a vim built with `+clientserver`. Off by default because of portability issues (minimal vim, Neovim, user autocmds); when off, `C`/`e` suspend lazysvn and run `$EDITOR` in the same terminal. |

## Troubleshooting

Every `svn` subprocess (start, finish, duration, exit status, stderr tail on error) is logged to `~/.cache/lazysvn/log`. If an operation appears to hang:

- Press `Esc` while the spinner is showing to cancel the running subprocess.
- Tail the log to see which `svn` command is stuck: `tail -f ~/.cache/lazysvn/log`.
- A common cause in locked-down environments is `svn+ssh://` where `ssh` needs a passphrase and reads `/dev/tty` directly; set up `ssh-agent` or a key without passphrase so the subprocess doesn't block on a prompt hidden behind the TUI.

## Requirements

- **Runtime:** `svn` command-line client on `$PATH`
- **Build:** Go 1.22+
- **Platform:** Linux (x86_64, arm64) and macOS (Intel, Apple Silicon); prebuilt binaries provided for all four

## License

MIT
