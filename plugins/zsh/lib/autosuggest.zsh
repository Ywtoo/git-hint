# ----------------------------------------------------------------------------
# Interop with zsh-autosuggestions
# ----------------------------------------------------------------------------

_githint_autosuggest_off() {
    if [[ -z "$GITHINT_ORIG_AUTOSUGGEST_STYLE" ]]; then
        GITHINT_ORIG_AUTOSUGGEST_STYLE="${ZSH_AUTOSUGGEST_HIGHLIGHT_STYLE:-fg=8}"
    fi

    (( ${+functions[_zsh_autosuggest_disable]} )) && _zsh_autosuggest_disable
    ZSH_AUTOSUGGEST_HIGHLIGHT_STYLE="none"
}

_githint_autosuggest_on() {
    ZSH_AUTOSUGGEST_HIGHLIGHT_STYLE="${GITHINT_ORIG_AUTOSUGGEST_STYLE:-fg=8}"
    unset _ZSH_AUTOSUGGEST_DISABLED
}