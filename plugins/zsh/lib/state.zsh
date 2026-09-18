# ----------------------------------------------------------------------------
# Plugin lifecycle
# ----------------------------------------------------------------------------

if (( ! ${+GITHINT_UNLOADED} )); then
    typeset -g GITHINT_UNLOADED=0
    typeset -g GITHINT_INITIALIZED=1
    typeset -g GITHINT_ENABLED=2
fi

typeset -gi GITHINT_STATE=${GITHINT_STATE:-0}

# ----------------------------------------------------------------------------
# Daemon
# ----------------------------------------------------------------------------

typeset -g GITHINT_PLUGIN_DIR="${GITHINT_PLUGIN_DIR:-}"
typeset -g GITHINT_BIN=""
typeset -g GITHINT_SOCK="/tmp/githint-${USER}.sock"
typeset -g GITHINT_DAEMON_STARTING=0
typeset -gF GITHINT_DAEMON_START_TS=0
typeset -g __githint_socket_result=""

# ----------------------------------------------------------------------------
# UI state
# ----------------------------------------------------------------------------

typeset -g GITHINT_SELECTED=0
typeset -g GITHINT_PREV_BUFFER=""
typeset -g GITHINT_PROMPT_COL=0
typeset -g GITHINT_PROMPT_COL_CACHE=0
typeset -g GITHINT_PROMPT_COL_DIR=""
typeset -g GITHINT_RENDER="ohmyzsh"

# ----------------------------------------------------------------------------
# Highlights
# ----------------------------------------------------------------------------

typeset -g __githint_clean=""
typeset -ga GITHINT_OWN_HIGHLIGHTS

# ----------------------------------------------------------------------------
# Key bindings
# ----------------------------------------------------------------------------

typeset -gA GITHINT_ORIG_UP
typeset -gA GITHINT_ORIG_DOWN
typeset -gA GITHINT_ORIG_TAB
typeset -gA GITHINT_ORIG_ESC

# ----------------------------------------------------------------------------
# zsh-autosuggestions
# ----------------------------------------------------------------------------

typeset -g GITHINT_ORIG_AUTOSUGGEST_STYLE=""
typeset -g GITHINT_AUTOSUGGEST_STATE="on"

# ----------------------------------------------------------------------------
# Misc
# ----------------------------------------------------------------------------

typeset -g GITHINT_MISSING_WARNED=0
typeset -g GITHINT_ALIASES_PAYLOAD=""
typeset -g GITHINT_ALIASES_CACHED=0

# ----------------------------------------------------------------------------
# Aliases payload (sync shell aliases → daemon)
# ----------------------------------------------------------------------------

# _githint_aliases_payload serializes `alias -L` into tab-separated
# name<TAB>definition<TAB>name<TAB>definition... pairs. Cached per command
# execution and invalidated by precmd, so a fresh `alias` made in this same
# session is picked up on the next prompt.
typeset -ga _githint_watch_PRECMD
typeset -ga _githint_watch_PREEXEC

_githint_aliases_payload() {
    if (( GITHINT_ALIASES_CACHED )); then
        print -r -- "$GITHINT_ALIASES_PAYLOAD"
        return
    fi

    local line name def out=""
    local -a pairs

    while IFS= read -r line; do
        [[ "$line" == "="* ]] && continue
        name="${line%%=*}"
        def="${line#*=}"
        [[ -z "$name" || -z "$def" ]] && continue
        pairs+=("$name"$'\t'"$def")
    done < <(builtin alias -L 2>/dev/null)

    if (( ${#pairs[@]} )); then
        out="${(j:\t:)pairs}"
    fi

    GITHINT_ALIASES_PAYLOAD="$out"
    GITHINT_ALIASES_CACHED=1
    print -r -- "$out"
}

# Invalidate the alias cache whenever the prompt is about to be drawn, so a
# new alias defined since the last command shows up on the next completion.
autoload -Uz add-zsh-hook
_githint_invalidate_aliases() {
    GITHINT_ALIASES_CACHED=0
    # A new command is starting: re-enable the update hook after an accept-line
    # suppressed it (prevents the list from flickering while the line dies).
    GITHINT_SUPPRESS_UPDATE=0
}
add-zsh-hook precmd _githint_invalidate_aliases
