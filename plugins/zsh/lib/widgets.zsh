# ----------------------------------------------------------------------------
# Widgets
# ----------------------------------------------------------------------------
githint-accept-line() {
    local current_pos
    current_pos="$(_githint_get_cursor_col)"

    local prompt_width=$(( current_pos - ${#BUFFER} ))
    (( prompt_width < 0 )) && prompt_width=0

    GITHINT_PROMPT_COL=$prompt_width

    # Suggestions live in POSTDISPLAY. Clear them before accepting the line
    # and suppress the pre-redraw hook until the next command starts, so the
    # dying widget does not redraw a list below the accepted line (the flicker
    # seen on Enter).
    typeset -g GITHINT_SUPPRESS_UPDATE=1
    POSTDISPLAY=""
    _githint_clear_own_highlights
    GITHINT_LAST_RESULT=""

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
            up-line-or-history)
                GITHINT_SELECTED=-1
                [[ -n "$fallback_widget" ]] && zle "$fallback_widget"
                ;;
            down-line-or-history)
                local old_buf="$BUFFER"
                [[ -n "$fallback_widget" ]] && zle "$fallback_widget"
                
                # Se o historico chegou ao fim (BUFFER nao mudou ao dar DOWN ou HISTNO voltou ao atual)
                if [[ "$BUFFER" == "$old_buf" || ${HISTNO:-0} -eq ${HISTCMD:-0} ]]; then
                    GITHINT_SELECTED=0
                else
                    GITHINT_SELECTED=-1
                fi
                ;;
            *)
                zle "$widget"
                ;;
        esac
    fi

    if [[ "$key" == "TAB" || "$key" == "ESC" ]]; then
        # TAB completa/entra-sai de grupos e ESC sai da tela expandida — ambos
        # mudam a lista renderizada, então o prompt precisa ser redesenhado.
        # Sem isso o ESC "não funciona": o estado sai da expansão no daemon,
        # mas a tela continua mostrando a versão expandida.
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
    _githint_key_handler "TAB" "${GITHINT_ORIG_TAB[$KEYMAP]:-expand-or-complete}"
}

githint-esc() {
    _githint_key_handler "ESC" "${GITHINT_ORIG_ESC[$KEYMAP]:-send-break}"
}
