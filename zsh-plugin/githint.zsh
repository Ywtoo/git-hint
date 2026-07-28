#!/usr/bin/env zsh
# ============================================================================
# githint — smart autocomplete for git commands in zsh
# ============================================================================

zmodload zsh/terminfo
zmodload zsh/net/socket

# ----------------------------------------------------------------------------
# Global plugin state
# ----------------------------------------------------------------------------
typeset -g GITHINT_SOCK="/tmp/githint-${USER}.sock"
typeset -g GITHINT_SELECTED=0
typeset -g GITHINT_PREV_BUFFER=""
typeset -g GITHINT_PROMPT_COL=0
typeset -g GITHINT_RENDER="ohmyzsh"
typeset -g GITHINT_ORIG_AUTOSUGGEST_STYLE=""
typeset -g GITHINT_AUTOSUGGEST_STATE="on"
typeset -g __githint_clean=""
typeset -g __githint_socket_result=""
typeset -ga GITHINT_OWN_HIGHLIGHTS
typeset -gA GITHINT_ORIG_UP
typeset -gA GITHINT_ORIG_DOWN

# ----------------------------------------------------------------------------
# Binary resolution
# ----------------------------------------------------------------------------
_githint_resolve_bin_path() {
    local plugin_dir="${${(%):-%x}:A:h}"
    local repo_root="${plugin_dir:h}"   # one level above zsh-plugin/

    if [[ -x "$plugin_dir/githint" ]]; then
        print -r -- "$plugin_dir/githint"
        return
    fi
    if [[ -x "$plugin_dir/bin/githint" ]]; then
        print -r -- "$plugin_dir/bin/githint"
        return
    fi
    if [[ -x "$repo_root/githint" ]]; then
        print -r -- "$repo_root/githint"
        return
    fi
    if command -v githint >/dev/null 2>&1; then
        command -v githint
        return
    fi

    print -r -- "$plugin_dir/githint"
}

GITHINT_BIN="$(_githint_resolve_bin_path)"
typeset -g GITHINT_MISSING_WARNED=0

# ----------------------------------------------------------------------------
# Daemon client — talks over a Unix socket instead of forking per keystroke.
# ----------------------------------------------------------------------------
typeset -g GITHINT_DAEMON_STARTING=0

_githint_ensure_daemon() {
    [[ -S "$GITHINT_SOCK" ]] && return 0
    (( GITHINT_DAEMON_STARTING )) && return 1
    [[ -x "$GITHINT_BIN" ]] || return 1

    GITHINT_DAEMON_STARTING=1
    "$GITHINT_BIN" daemon &>/dev/null &!

    # the next keystroke should already find the socket alive; reset the
    # flag after a short delay so future attempts aren't blocked in case
    # the daemon actually failed to start
    ( sleep 1; GITHINT_DAEMON_STARTING=0 ) &!
}

_githint_socket_call() {
    local request="$1"
    local fd

    [[ -S "$GITHINT_SOCK" ]] || return 1

    zsocket "$GITHINT_SOCK" 2>/dev/null || return 1
    fd=$REPLY

    print -u $fd -- "$request" || { exec {fd}>&-; return 1; }

    local size
    read -r -u $fd size || { exec {fd}>&-; return 1; }

    local response=""
    if [[ "$size" == <-> ]] && (( size > 0 )); then
        read -k $size -u $fd response
    fi

    exec {fd}>&-
    __githint_socket_result="$response"
    return 0
}

_githint_ensure_daemon

# ----------------------------------------------------------------------------
# Prompt width calculation (Oh My Zsh)
# ----------------------------------------------------------------------------
_githint_calc_width() {
    local folder=${PWD:t}
    local folder_len=${#folder}
    local base_offset=2

    if git rev-parse --is-inside-work-tree &>/dev/null; then
        local branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null)
        local branch_len=${#branch}
        local git_deco_offset=9
        echo $(( base_offset + folder_len + git_deco_offset + branch_len ))
    else
        echo $(( base_offset + folder_len ))
    fi
}

typeset -g GITHINT_PROMPT_COL_CACHE=0
typeset -g GITHINT_PROMPT_COL_DIR=""

_githint_calc_width_cached() {
    local cache_key="$PWD"
    if [[ "$cache_key" != "$GITHINT_PROMPT_COL_DIR" ]]; then
        GITHINT_PROMPT_COL_CACHE=$(_githint_calc_width)
        GITHINT_PROMPT_COL_DIR="$cache_key"
    fi
    echo "$GITHINT_PROMPT_COL_CACHE"
}

# ----------------------------------------------------------------------------
# Color highlighting (sentinels → region_highlight)
# ----------------------------------------------------------------------------
_githint_apply_highlight() {
    local raw="$1"
    local base=$2
    local clean=""
    local -a ranges
    local pos=$base
    local i=1

    while (( i <= ${#raw} )); do
        local char="${raw[i]}"

        if [[ "$char" == $'\x01' || "$char" == $'\x03' ]]; then
            local color=28
            local closer=$'\x02'
            if [[ "$char" == $'\x03' ]]; then
                color=8
                closer=$'\x04'
            fi

            local start=$pos
            local content=""
            (( i++ ))
            while (( i <= ${#raw} )) && [[ "${raw[i]}" != "$closer" ]]; do
                content+="${raw[i]}"
                (( i++ ))
            done
            clean+="$content"
            pos=$(( pos + ${#content} ))
            ranges+=("$start $pos fg=$color")
            (( i++ ))
        else
            clean+="$char"
            (( pos++ ))
            (( i++ ))
        fi
    done

    __githint_clean="$clean"
    local entry
    for entry in "${ranges[@]}"; do
        region_highlight+=("$entry")
        GITHINT_OWN_HIGHLIGHTS+=("$entry")
    done
}

_githint_clear_own_highlights() {
    (( ${#GITHINT_OWN_HIGHLIGHTS} == 0 )) && return

    local -a kept
    local entry own_entry found
    for entry in "${region_highlight[@]}"; do
        found=0
        for own_entry in "${GITHINT_OWN_HIGHLIGHTS[@]}"; do
            if [[ "$entry" == "$own_entry" ]]; then
                found=1
                break
            fi
        done
        (( found == 0 )) && kept+=("$entry")
    done

    region_highlight=("${kept[@]}")
    GITHINT_OWN_HIGHLIGHTS=()
}

# ----------------------------------------------------------------------------
# Interop with zsh-autosuggestions
# ----------------------------------------------------------------------------
_githint_autosuggest_off() {
    if [[ -z "$GITHINT_ORIG_AUTOSUGGEST_STYLE" ]]; then
        GITHINT_ORIG_AUTOSUGGEST_STYLE="${ZSH_AUTOSUGGEST_HIGHLIGHT_STYLE:-fg=8}"
    fi
    _zsh_autosuggest_disable 2>/dev/null
    ZSH_AUTOSUGGEST_HIGHLIGHT_STYLE="none"
}

_githint_autosuggest_on() {
    ZSH_AUTOSUGGEST_HIGHLIGHT_STYLE="${GITHINT_ORIG_AUTOSUGGEST_STYLE:-fg=8}"
    unset _ZSH_AUTOSUGGEST_DISABLED
}

# ----------------------------------------------------------------------------
# Terminal synchronized output — hides the clear/redraw flicker
# ----------------------------------------------------------------------------
_githint_sync_start() {
    print -n $'\e[?2026h' > /dev/tty 2>/dev/null
}

_githint_sync_end() {
    print -n $'\e[?2026l' > /dev/tty 2>/dev/null
}

# ----------------------------------------------------------------------------
# Suggestion list update — called on every ZLE redraw
# ----------------------------------------------------------------------------
_githint_update() {
    local buffer="$BUFFER"

    if [[ "$buffer" != "$GITHINT_PREV_BUFFER" ]]; then
        GITHINT_SELECTED=0
        GITHINT_PREV_BUFFER="$buffer"
    fi

    GITHINT_PROMPT_COL=$(_githint_calc_width_cached)

    if [[ "$GITHINT_AUTOSUGGEST_STATE" != "off" ]]; then
        _githint_autosuggest_off
        GITHINT_AUTOSUGGEST_STATE="off"
    fi

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

    local result
    if _githint_socket_call $'list\t'"$buffer"$'\t'"$GITHINT_SELECTED"$'\t'"$GITHINT_PROMPT_COL"$'\t'"$GITHINT_RENDER"; then
        result="$__githint_socket_result"
    else
        result=$("$GITHINT_BIN" list "$buffer" "$GITHINT_SELECTED" "$GITHINT_PROMPT_COL" "$GITHINT_RENDER")
        _githint_ensure_daemon
    fi

    typeset -g GITHINT_LAST_RESULT=""

    _githint_clear_own_highlights

    if [[ -n "$result" ]]; then
        local base=$(( ${#buffer} + 1 ))
        _githint_apply_highlight "$result" "$base"
        local new_postdisplay=$'\n'"$__githint_clean"
    else
        local new_postdisplay=""
    fi

    if [[ "$new_postdisplay" != "$POSTDISPLAY" ]]; then
        POSTDISPLAY="$new_postdisplay"
        _githint_sync_start
        zle reset-prompt
        _githint_sync_end
    fi
}

# ----------------------------------------------------------------------------
# Hook registration
# ----------------------------------------------------------------------------
autoload -Uz add-zle-hook-widget
_githint_pre_redraw_hook() { _githint_update; }
zle -N _githint_pre_redraw_hook
add-zle-hook-widget zle-line-pre-redraw _githint_pre_redraw_hook

# ----------------------------------------------------------------------------
# Real cursor position capture (ESC[6n)
# ----------------------------------------------------------------------------
_githint_get_cursor_col() {
    [[ -t 0 ]] || { echo "0"; return; }

    local oldstty
    oldstty=$(stty -g 2>/dev/null) || { echo "0"; return; }
    stty raw -echo min 0 time 5 2>/dev/null
    printf '\e[6n' > /dev/tty 2>/dev/null

    local pos=""
    while read -t 0.05 -k 1 ch 2>/dev/null; do
        pos+="$ch"
        [[ "$ch" == "R" ]] && break
    done
    stty "$oldstty" 2>/dev/null

    local col=$(echo "$pos" | sed -E 's/.*;([0-9]+)R/\1/')
    echo "${col:-0}"
}

githint-accept-line() {
    local current_pos=$(_githint_get_cursor_col)
    local prompt_width=$(( current_pos - ${#BUFFER} ))
    (( prompt_width < 0 )) && prompt_width=0
    GITHINT_PROMPT_COL=$prompt_width
    zle .accept-line
}
zle -N githint-accept-line

# ----------------------------------------------------------------------------
# Navigation (arrow keys and TAB)
# ----------------------------------------------------------------------------
_githint_key_handler() {
    local key="$1"
    local fallback_widget="$2"
    local result widget new_selection new_buffer

    if _githint_socket_call $'key\t'"$key"$'\t'"$GITHINT_SELECTED"$'\t'"$BUFFER"; then
        result="$__githint_socket_result"
    else
        result=$("$GITHINT_BIN" key "$key" "$GITHINT_SELECTED" "$BUFFER")
        _githint_ensure_daemon
    fi

    if [[ -z "$result" ]]; then
        [[ -n "$fallback_widget" ]] && zle "$fallback_widget"
        _githint_update
        return
    fi

    widget="${result%%|*}"
    local remainder="${result#*|}"
    new_selection="${remainder%%|*}"
    new_buffer="${remainder#*|}"

    [[ -n "$new_selection" ]] && GITHINT_SELECTED=$new_selection

    if [[ -n "$new_buffer" ]]; then
        BUFFER="$new_buffer"
        CURSOR=${#BUFFER}
    fi

    if [[ -n "$widget" && "$widget" != '""' ]]; then
        if [[ "$widget" == "up-line-or-history" || "$widget" == "down-line-or-history" ]]; then
            [[ -n "$fallback_widget" ]] && zle "$fallback_widget"
        else
            zle "$widget"
        fi
    fi

    [[ "$key" == "TAB" ]] && { _githint_sync_start; zle reset-prompt; _githint_sync_end; }
}

githint-arrow-up()   { _githint_key_handler "arrowUP"   "${GITHINT_ORIG_UP[$KEYMAP]:-up-line-or-history}"; }
githint-arrow-down() { _githint_key_handler "arrowDOWN" "${GITHINT_ORIG_DOWN[$KEYMAP]:-down-line-or-history}"; }
githint-tab()        { _githint_key_handler "TAB" "expand-or-complete"; }

zle -N githint-arrow-up
zle -N githint-arrow-down
zle -N githint-tab

# ----------------------------------------------------------------------------
# Key bindings — capture original widgets BEFORE overwriting them
# ----------------------------------------------------------------------------
_githint_nuclear_bind() {
    print -u2 -- "ENTER nuclear_bind"
    local key="$1" widget="$2" orig_array="$3"
    [[ -z "$key" ]] && return

    local m
    for m in main emacs viins vicmd; do
        if [[ -n "$orig_array" ]]; then
            local original
            bindkey -M "$m" "$key"
            print -u2 -- "AFTER bindkey"
            if [[ -n "$original" && "$original" != "undefined-key" && "$original" != "$widget" ]]; then
                typeset -gA "$orig_array"
:
            fi
        fi

        bindkey -M "$m" -r "$key" 2>/dev/null
        bindkey -M "$m" "$key" "$widget"
    done
}

_githint_nuclear_bind "${terminfo[kcuu1]}" githint-arrow-up GITHINT_ORIG_UP
_githint_nuclear_bind '^[[A'               githint-arrow-up GITHINT_ORIG_UP
_githint_nuclear_bind '^[OA'               githint-arrow-up GITHINT_ORIG_UP
_githint_nuclear_bind '^P'                 githint-arrow-up GITHINT_ORIG_UP

_githint_nuclear_bind "${terminfo[kcud1]}" githint-arrow-down GITHINT_ORIG_DOWN
_githint_nuclear_bind '^[[B'               githint-arrow-down GITHINT_ORIG_DOWN
_githint_nuclear_bind '^[OB'               githint-arrow-down GITHINT_ORIG_DOWN
_githint_nuclear_bind '^N'                 githint-arrow-down GITHINT_ORIG_DOWN

_githint_nuclear_bind '^I' githint-tab
_githint_nuclear_bind '^M' githint-accept-line