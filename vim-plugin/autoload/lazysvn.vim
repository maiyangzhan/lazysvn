function! lazysvn#open() abort
  let l:cmd = g:lazysvn_cmd

  if exists('v:servername') && v:servername !=# ''
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
