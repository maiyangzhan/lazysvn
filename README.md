# lazysvn

A lazygit-style terminal UI for SVN. Single static binary, works standalone or from Vim.

## Install

### CLI

Download the binary for your platform and put it on your `$PATH`:

```bash
# On your Mac (build from source)
make linux                          # cross-compile for Linux
scp dist/lazysvn-linux-amd64 server:~/bin/lazysvn

# On the server
chmod +x ~/bin/lazysvn
```

### Vim plugin

Copy the `vim-plugin/` contents into your Vim runtime path:

```bash
cp -r vim-plugin/* ~/.vim/
```

Or with a plugin manager (e.g. vim-plug):

```vim
Plug 'maiyangzhan/lazysvn', { 'rtp': 'vim-plugin' }
```

## Usage

```bash
lazysvn                     # run in current directory
lazysvn --cwd /path/to/wc   # specify working copy
lazysvn --log-limit 100     # show more log entries (default: 50)
```

From Vim: `:LazySvn` or `<Leader>s`.

## Key Bindings

### Navigation

| Key | Action |
|---|---|
| `j` / `k` | Move cursor down / up |
| `g` / `G` | Jump to first / last item |
| `Tab` / `Shift-Tab` | Switch focus between panels |

### File Operations (Files panel)

| Key | Action |
|---|---|
| `Space` | Toggle mark on current file |
| `c` | Commit marked/current file(s) via `$EDITOR` |
| `r` | Revert marked/current file(s) (confirm) |
| `a` | Add untracked file(s) |
| `x` | Delete file(s) (confirm) |
| `m` | Mark conflict as resolved |

### Global

| Key | Action |
|---|---|
| `u` | SVN update (with progress indicator) |
| `R` | Refresh all panels |
| `q` | Quit |

## Layout

```
+---------------------+----------------------------+
|   Files             |                            |
|   M  src/foo.sv     |     Preview (diff)         |
|   A  src/bar.sv     |     +line                  |
+---------------------+     -line                  |
|   Log               |                            |
|   r42 alice 05-08   |                            |
+---------------------+----------------------------+
| c:commit r:revert a:add x:delete Space:mark q:quit |
+----------------------------------------------------+
```

## Nested Vim

When launched from inside Vim via `:LazySvn`, the plugin sets `$VIM_SERVERNAME`.
Pressing `c` (commit) will open the commit message in your host Vim instance
via `--remote-wait-silent`, avoiding a nested Vim.

## Configuration

| Vim variable | Default | Description |
|---|---|---|
| `g:lazysvn_cmd` | `"lazysvn"` | Path to the binary |
| `g:lazysvn_no_default_mapping` | (unset) | Set to disable `<Leader>s` |

## Build from Source

Requires Go 1.22+.

```bash
make build    # macOS binary
make linux    # Linux x86_64 static binary
make test     # run tests
make clean    # remove build artifacts
```

## License

MIT
