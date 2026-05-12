function! lazysvn#open() abort
  let l:cmd = g:lazysvn_cmd

  " Optional: propagate v:servername so lazysvn's 'C' (commit via
  " editor) and 'e' (edit file) open the buffer in this vim instance
  " via `vim --servername X --remote-wait-silent`.
  "
  " Disabled by default because the vim remote protocol has enough
  " environmental pitfalls (vim without +clientserver, neovim's
  " different RPC, user autocmds that misbehave with
  " --remote-wait-silent) that it surprises people. When disabled,
  " lazysvn suspends its TUI and launches $EDITOR in the same terminal
  " (a nested vim buffer inside the popup is mildly ugly but reliable).
  "
  " Opt in with: let g:lazysvn_vim_remote = 1
  if get(g:, 'lazysvn_vim_remote', 0)
        \ && exists('v:servername') && v:servername !=# ''
        \ && has('clientserver')
    let $VIM_SERVERNAME = v:servername
  endif

  if exists('*popup_create') && exists('*term_start')
    call s:open_popup(l:cmd)
  else
    call s:open_tab(l:cmd)
  endif
endfunction

function! s:open_popup(cmd) abort
  let l:buf = term_start(a:cmd, #{
        \ hidden: 1,
        \ term_finish: 'close',
        \ term_cols: &columns - 4,
        \ term_rows: &lines - 4,
        \ })
  call popup_create(l:buf, #{
        \ minwidth: &columns - 4,
        \ minheight: &lines - 4,
        \ border: [],
        \ padding: [0, 0, 0, 0],
        \ zindex: 200,
        \ })
endfunction

function! s:open_tab(cmd) abort
  tabnew
  execute 'terminal ++curwin ++close ' . a:cmd
  startinsert
endfunction
