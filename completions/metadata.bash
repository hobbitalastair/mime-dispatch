_metadata() {
    local cur prev words cword
    _init_completion || return

    local commands="list add delete"
    local keys="mime_type datetime location comment title album artist album_artist composer genre year"

    # Find the position of the command (first non-flag arg)
    local cmd_idx=-1
    local idx
    for ((idx = 1; idx < cword; idx++)); do
        if [[ ! ${words[idx]} =~ ^- ]]; then
            cmd_idx=$idx
            break
        fi
    done

    if [[ $cmd_idx -eq -1 ]]; then
        # No command yet — suggest commands and flags
        if [[ $cur == -* ]]; then
            COMPREPLY=($(compgen -W "--xattr-only -x --file-only -f" -- "$cur"))
        else
            COMPREPLY=($(compgen -W "$commands" -- "$cur"))
        fi
        return
    fi

    local cmd=${words[cmd_idx]}

    # Determine which positional argument we are on (relative to the command)
    local rel_idx=$((cword - cmd_idx))

    case $cmd in
        list)
            if [[ $rel_idx -eq 1 ]]; then
                _filedir
            fi
            ;;
        add|delete)
            case $rel_idx in
                1) _filedir ;;
                2) COMPREPLY=($(compgen -W "$keys" -- "$cur")) ;;
            esac
            ;;
    esac

    # If still completing, offer flags
    if [[ $cur == -* ]]; then
        COMPREPLY+=($(compgen -W "--xattr-only -x --file-only -f" -- "$cur"))
    fi
} &&
complete -F _metadata metadata
