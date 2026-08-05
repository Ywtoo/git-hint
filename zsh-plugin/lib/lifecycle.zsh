# ----------------------------------------------------------------------------
# Plugin lifecycle
# ----------------------------------------------------------------------------

_githint_init() {
    (( GITHINT_STATE != GITHINT_UNLOADED )) && return

    autoload -Uz add-zle-hook-widget

    # Register widgets (one time only)
    zle -N _githint_pre_redraw_hook
    zle -N githint-arrow-up
    zle -N githint-arrow-down
    zle -N githint-tab
    zle -N githint-accept-line

    GITHINT_STATE=$GITHINT_INITIALIZED
}

_githint_enable() {
    (( GITHINT_STATE == GITHINT_ENABLED )) && return
    (( GITHINT_STATE == GITHINT_UNLOADED )) && _githint_init

    GITHINT_BIN="$(_githint_resolve_bin_path)"
    if [[ -z "$GITHINT_BIN" ]]; then
        print -u2 "githint: binário não encontrado"
        return 1
    fi

    add-zle-hook-widget zle-line-pre-redraw _githint_pre_redraw_hook
    add-zle-hook-widget zle-line-init _githint_autosuggest_off

    _githint_bind_keys

    GITHINT_STATE=$GITHINT_ENABLED
}

_githint_disable() {
    (( GITHINT_STATE != GITHINT_ENABLED )) && return

    add-zle-hook-widget -d zle-line-pre-redraw _githint_pre_redraw_hook
    add-zle-hook-widget -d zle-line-init _githint_autosuggest_off

    _githint_unbind_keys
    _githint_autosuggest_on

    GITHINT_STATE=$GITHINT_INITIALIZED
}

_githint_cleanup() {
    _githint_disable

    GITHINT_STATE=$GITHINT_UNLOADED
}