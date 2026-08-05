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

# ----------------------------------------------------------------------------
# zsh-autosuggestions
# ----------------------------------------------------------------------------

typeset -g GITHINT_ORIG_AUTOSUGGEST_STYLE=""
typeset -g GITHINT_AUTOSUGGEST_STATE="on"

# ----------------------------------------------------------------------------
# Misc
# ----------------------------------------------------------------------------

typeset -g GITHINT_MISSING_WARNED=0