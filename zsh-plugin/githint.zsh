#!/usr/bin/env zsh

zmodload zsh/terminfo

typeset -g GITHINT_SELECTED=0
typeset -g GITHINT_PREV_BUFFER=""

#TODO: Path to githint binary
GITHINT_BIN="/media/storage_fixed/Programming/git-hint/githint"

# Update suggestion list
_githint_update() {
    local buffer="$BUFFER"

    if [[ "$buffer" != "$GITHINT_PREV_BUFFER" ]]; then
        if [[ $GITHINT_SELECTED -ge 0 ]]; then
            GITHINT_SELECTED=0
        fi
        GITHINT_PREV_BUFFER="$buffer"
    fi

    # Disable autosuggest immediately if command starts with 'git'
    if [[ "$buffer" =~ ^git ]]; then
        _zsh_autosuggest_disable
    else
        unset _ZSH_AUTOSUGGEST_DISABLED
        POSTDISPLAY=""
        return
    fi

    local resultado
    resultado=$("$GITHINT_BIN" list "$buffer" "$GITHINT_SELECTED")

    if [[ -n "$resultado" ]]; then
        POSTDISPLAY=$'\n'"$resultado"
    else
        POSTDISPLAY=""
    fi

    zle reset-prompt
}

zle-line-pre-redraw() { _githint_update}
zle -N zle-line-pre-redraw

# Navigation handler
_githint_key_handler() {
    local tecla="$1"
    local resultado
    local widget
    local new_sel
    local new_buffer

    # Pass current state to Go engine
    resultado=$("$GITHINT_BIN" key "$tecla" "$GITHINT_SELECTED" "$BUFFER")

    if [[ -n "$resultado" ]]; then
        # Split "widget|selected|buffer"
        widget="${resultado%%|*}"
        local remainder="${resultado#*|}"
        new_sel="${remainder%%|*}"
        new_buffer="${remainder#*|}"

        # Update global selection
        if [[ -n "$new_sel" ]]; then
            GITHINT_SELECTED=$new_sel
        fi

        # Update the shell buffer
        if [[ -n "$new_buffer" ]]; then
            BUFFER="$new_buffer"
            CURSOR=${#BUFFER}
        fi

        # Execute ZLE widget if available
        if [[ -n "$widget" && "$widget" != '""' ]]; then
            zle "$widget"
        fi
    fi

    _githint_update
}

githint-arrow-up() { _githint_key_handler "arrowUP"; }
zle -N githint-arrow-up

githint-arrow-down() { _githint_key_handler "arrowDOWN"; }
zle -N githint-arrow-down

githint-tab() { _githint_key_handler "TAB"; }
zle -N githint-tab

# Key bindings
_githint_nuclear_bind() {
    local key="$1"
    local widget="$2"

    [[ -z "$key" ]] && return

    local maps=("main" "emacs" "viins" "vicmd")

    for m in $maps; do
        bindkey -M "$m" -r "$key" 2>/dev/null
        bindkey -M "$m" "$key" "$widget"
    done
}

# Dynamic mapping via terminfo and fallbacks
_githint_nuclear_bind "${terminfo[kcuu1]}" githint-arrow-up  # Physical Up
_githint_nuclear_bind '^[[A' githint-arrow-up                # ANSI Fallback
_githint_nuclear_bind '^[OA' githint-arrow-up                # Application Mode Fallback
_githint_nuclear_bind '^P' githint-arrow-up                  # Emacs Ctrl+P

_githint_nuclear_bind "${terminfo[kcud1]}" githint-arrow-down  # Physical Down
_githint_nuclear_bind '^[[B' githint-arrow-down                # ANSI Fallback
_githint_nuclear_bind '^[OB' githint-arrow-down                # Application Mode Fallback
_githint_nuclear_bind '^N' githint-arrow-down                  # Emacs Ctrl+N

# TAB binding (override Zsh default autocomplete)
_githint_nuclear_bind '^I' githint-tab
