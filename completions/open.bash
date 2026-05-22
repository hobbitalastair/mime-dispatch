_open() {
    local cur prev words cword
    _init_completion || return
    _filedir
} &&
complete -F _open open
