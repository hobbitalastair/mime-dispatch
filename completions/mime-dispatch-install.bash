_mime_dispatch_install() {
    local cur prev words cword
    _init_completion || return

    # Check if the positional binary path argument has already been given
    local has_positional=0
    local has_level=0
    local word
    for word in "${words[@]:1:cword-1}"; do
        if [[ ! $word =~ ^- ]]; then
            has_positional=1
            break
        fi
        case $word in
            --user|--system|--vendor) has_level=1 ;;
        esac
    done

    # If binary path already given, nothing more to complete
    if [[ $has_positional -eq 1 ]]; then
        COMPREPLY=()
        return
    fi

    # Flag argument handling
    if [[ $prev == --mimetype ]]; then
        return
    fi

    local flags="--mimetype --uninstall --help"
    if [[ $has_level -eq 0 ]]; then
        flags="--user --system --vendor $flags"
    fi

    COMPREPLY=($(compgen -W "$flags" -- "$cur"))
    if [[ $cur != -* ]]; then
        local files
        _filedir files
        COMPREPLY+=("${files[@]}")
    fi
} &&
complete -F _mime_dispatch_install mime-dispatch-install
