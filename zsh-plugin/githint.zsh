typeset -g GITHINT_SELECTED=1

_githint_update() {
    local resultado

    resultado=$(./githint "$BUFFER" 2>/dev/null)

    if [[ -n "$resultado" ]]; then
        _zsh_autosuggest_disable
        POSTDISPLAY=$'\n'"$resultado"
    else
        POSTDISPLAY=""
        unset _ZSH_AUTOSUGGEST_DISABLED
    fi
}

zle-line-pre-redraw() {
    _githint_update
}

zle -N zle-line-pre-redraw