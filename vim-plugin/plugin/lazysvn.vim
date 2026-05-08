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
