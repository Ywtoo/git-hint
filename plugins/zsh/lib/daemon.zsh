# ----------------------------------------------------------------------------
# Binary resolution
# ----------------------------------------------------------------------------
_githint_resolve_bin_path() {
    local plugin_dir="$GITHINT_PLUGIN_DIR"
    local repo_root="${plugin_dir:h}"

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
        print -r -- "$(command -v githint)"
        return
    fi

    print -r -- "$plugin_dir/githint"
}

# ----------------------------------------------------------------------------
# Daemon
# ----------------------------------------------------------------------------
_githint_ensure_daemon() {
    if [[ -S "$GITHINT_SOCK" ]]; then
        # A socket file can survive after the daemon crashes. Merely checking
        # -S would permanently prevent the plugin from restarting it.
        local fd
        if zsocket "$GITHINT_SOCK" 2>/dev/null; then
            fd=$REPLY
            exec {fd}>&-
            GITHINT_DAEMON_STARTING=0
            return 0
        fi
        rm -f "$GITHINT_SOCK" 2>/dev/null
    fi

    if (( GITHINT_DAEMON_STARTING )); then
        (( EPOCHREALTIME - GITHINT_DAEMON_START_TS > 2 )) && GITHINT_DAEMON_STARTING=0
        (( GITHINT_DAEMON_STARTING )) && return 1
    fi

    [[ -x "$GITHINT_BIN" ]] || return 1

    GITHINT_DAEMON_STARTING=1
    "$GITHINT_BIN" daemon &>/dev/null &!
    GITHINT_DAEMON_PID=$!
    disown 2>/dev/null
    GITHINT_DAEMON_START_TS=$EPOCHREALTIME

    return 1
}

_githint_kill_daemon() {
    if [[ -n "$GITHINT_DAEMON_PID" ]] && kill -0 "$GITHINT_DAEMON_PID" 2>/dev/null; then
        kill "$GITHINT_DAEMON_PID" 2>/dev/null
    elif [[ -S "$GITHINT_SOCK" ]]; then
        # PID desconhecido (ex: shell reaberto) — mata pelo dono do socket
        if command -v fuser >/dev/null 2>&1; then
            fuser -k "$GITHINT_SOCK" 2>/dev/null
        elif command -v lsof >/dev/null 2>&1; then
            lsof -t "$GITHINT_SOCK" 2>/dev/null | xargs -r kill 2>/dev/null
        fi
    fi
    unset GITHINT_DAEMON_PID
    rm -f "$GITHINT_SOCK" 2>/dev/null
}

# ----------------------------------------------------------------------------
# Manual restart (user-facing command)
# ----------------------------------------------------------------------------
githint-restart() {
    _githint_kill_daemon

    GITHINT_DAEMON_STARTING=0
    GITHINT_BIN="$(_githint_resolve_bin_path)"

    if [[ -z "$GITHINT_BIN" || ! -x "$GITHINT_BIN" ]]; then
        print -u2 "githint: binário não encontrado ou não executável"
        return 1
    fi

    _githint_ensure_daemon

    # dá um instante pro socket subir e confirma
    local tries=0
    while (( tries < 20 )); do
        [[ -S "$GITHINT_SOCK" ]] && { print "githint: daemon reiniciado ✅"; return 0 }
        sleep 0.1
        (( tries++ ))
    done

    print -u2 "githint: daemon não respondeu a tempo ⚠️"
    return 1
}

# githint-dev-reset: mata o daemon, apaga o data/ e reinicia do zero.
# Útil pra testar o warmup de primeiro run durante desenvolvimento.
githint-dev-reset() {
    local bin="$(_githint_resolve_bin_path)"
    local data_dir="${bin:h}/data"

    print "githint-dev-reset: matando daemon..."
    _githint_kill_daemon
    # garante que o processo morreu antes de continuar
    sleep 0.3

    print "githint-dev-reset: limpando $data_dir ..."
    rm -rf "$data_dir"
    mkdir -p "$data_dir"

    print "githint-dev-reset: reiniciando daemon..."
    githint-restart
}

_githint_socket_call() {
    local request="$1"
    local fd

    [[ -S "$GITHINT_SOCK" ]] || return 1

    zsocket "$GITHINT_SOCK" 2>/dev/null || {
        rm -f "$GITHINT_SOCK" 2>/dev/null
        return 1
    }
    fd=$REPLY

    print -u "$fd" -- "$request" || {
        exec {fd}>&-
        return 1
    }

    local size
    read -r -t 2 -u "$fd" size || {
        exec {fd}>&-
        return 1
    }

    local response=""

    if [[ "$size" == <-> ]] && (( size > 0 )); then
        local __old_lc_all="$LC_ALL"
        LC_ALL=C
        read -k "$size" -t 2 -u "$fd" response
        local __read_rc=$?
        LC_ALL="$__old_lc_all"

        (( __read_rc != 0 )) && {
            exec {fd}>&-
            return 1
        }
    fi

    exec {fd}>&-

    __githint_socket_result="$response"

    return 0
}
