# ----------------------------------------------------------------------------
# Suggestion list update — called on every ZLE redraw
# ----------------------------------------------------------------------------
_githint_update() {
    (( PENDING > 0 )) && return

    local __t0=$EPOCHREALTIME

    local buffer="$BUFFER"

    if [[ "$buffer" != "$GITHINT_PREV_BUFFER" ]]; then
        # Se a mudanca veio de digitacao/backspace (nao do historico), reseta para 0
        if [[ ${HISTNO:-0} -eq ${HISTCMD:-0} ]]; then
            GITHINT_SELECTED=0
        fi
        GITHINT_PREV_BUFFER="$buffer"
    fi

    GITHINT_PROMPT_COL="$(_githint_calc_width_cached)"

    _githint_autosuggest_off

    if [[ ! -x "$GITHINT_BIN" ]]; then
        if (( ! GITHINT_MISSING_WARNED )); then
            print -u2 ""
            print -u2 "githint: binary not found at '$GITHINT_BIN'."
            print -u2 "         Run 'go build -o \"$GITHINT_BIN\"' — no need to source again afterwards."
            GITHINT_MISSING_WARNED=1
        fi
        return
    fi

    GITHINT_MISSING_WARNED=0

    local request=$'list\t'"$buffer"$'\t'"$GITHINT_SELECTED"$'\t'"$GITHINT_PROMPT_COL"$'\t'"$GITHINT_RENDER"
    local result=""
    if _githint_socket_call "$request"; then
        result="$__githint_socket_result"
    else
        _githint_ensure_daemon
    fi

    GITHINT_LAST_RESULT=""

    _githint_clear_own_highlights

    local new_postdisplay=""

    if [[ -n "$result" ]]; then
        local base=$(( ${#buffer} ))

        _githint_apply_highlight "$result" "$base"

        new_postdisplay=$'\n'"$__githint_clean"
    fi

    if [[ "$new_postdisplay" != "$POSTDISPLAY" ]]; then
        POSTDISPLAY="$new_postdisplay"
        
        #_githint_sync_start
        #zle reset-prompt
        #_githint_sync_end
    fi
}

_githint_pre_redraw_hook() {
    _githint_update
}

# ----------------------------------------------------------------------------
# Terminal synchronized output — hides redraw flicker
# ----------------------------------------------------------------------------
_githint_sync_start() {
    print -n -- $'\e[?2026h' > /dev/tty 2>/dev/null
}

_githint_sync_end() {
    print -n -- $'\e[?2026l' > /dev/tty 2>/dev/null
}