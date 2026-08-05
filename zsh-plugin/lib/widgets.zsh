# ----------------------------------------------------------------------------
# Widgets
# ----------------------------------------------------------------------------
githint-accept-line() {
    local current_pos
    current_pos="$(_githint_get_cursor_col)"

    local prompt_width=$(( current_pos - ${#BUFFER} ))
    (( prompt_width < 0 )) && prompt_width=0

    GITHINT_PROMPT_COL=$prompt_width

    zle .accept-line
}

# ----------------------------------------------------------------------------
# Navigation (arrow keys and TAB)
# ----------------------------------------------------------------------------
_githint_key_handler() {
    local key="$1"
    local fallback_widget="$2"

    local request=$'key\t'"$key"$'\t'"$GITHINT_SELECTED"$'\t'"$BUFFER"
    local result=""

    if _githint_socket_call "$request"; then
        result="$__githint_socket_result"
    else
        # socket falhou (daemon subindo, caiu, ou timeout).
        # não spawna processo síncrono na tecla do usuário — só garante
        # que o daemon está subindo em background pra próxima tecla,
        # e cai direto no fallback_widget dessa vez.
        _githint_ensure_daemon
    fi

    if [[ -z "$result" ]]; then
        [[ -n "$fallback_widget" ]] && zle "$fallback_widget"
        _githint_update
        return
    fi

    local widget="${result%%|*}"
    local remainder="${result#*|}"
    local new_selection="${remainder%%|*}"
    local new_buffer="${remainder#*|}"

    [[ -n "$new_selection" ]] && GITHINT_SELECTED="$new_selection"

    if [[ -n "$new_buffer" ]]; then
        BUFFER="$new_buffer"
        CURSOR=${#BUFFER}
    fi

    if [[ -n "$widget" && "$widget" != '""' ]]; then
        case "$widget" in
            up-line-or-history|down-line-or-history)
                [[ -n "$fallback_widget" ]] && zle "$fallback_widget"
                ;;
            *)
                zle "$widget"
                ;;
        esac
    fi

    if [[ "$key" == "TAB" ]]; then
        _githint_sync_start
        zle reset-prompt
        _githint_sync_end
    fi
}

githint-arrow-up() {
    _githint_key_handler "arrowUP" "${GITHINT_ORIG_UP[$KEYMAP]:-up-line-or-history}"
}

githint-arrow-down() {
    _githint_key_handler "arrowDOWN" "${GITHINT_ORIG_DOWN[$KEYMAP]:-down-line-or-history}"
}

githint-tab() {
    _githint_key_handler "TAB" "expand-or-complete"
}