# ----------------------------------------------------------------------------
# Key bindings
# ----------------------------------------------------------------------------

_githint_nuclear_bind() {
    local key="$1" widget="$2" orig_array="$3"
    [[ -z "$key" ]] && return

    local m entry original
    for m in main emacs viins vicmd; do
        if [[ -n "$orig_array" ]]; then
            entry="$(bindkey -M "$m" "$key" 2>/dev/null)"
            original="${entry#* }"
            original="${original//\'/}"
            original="${original##* }"

            if [[ -n "$original" &&
                  "$original" != "undefined-key" &&
                  "$original" != "$widget" ]]; then
                eval "${orig_array}[\$m]=\${original}" >/dev/null 2>&1
            fi
        fi

        bindkey -M "$m" -r "$key" >/dev/null 2>&1
        bindkey -M "$m" "$key" "$widget" >/dev/null 2>&1
    done
}

_githint_bind_keys() {
    _githint_nuclear_bind "${terminfo[kcuu1]}" githint-arrow-up GITHINT_ORIG_UP
    _githint_nuclear_bind '^[[A'               githint-arrow-up GITHINT_ORIG_UP
    _githint_nuclear_bind '^[OA'               githint-arrow-up GITHINT_ORIG_UP

    _githint_nuclear_bind "${terminfo[kcud1]}" githint-arrow-down GITHINT_ORIG_DOWN
    _githint_nuclear_bind '^[[B'               githint-arrow-down GITHINT_ORIG_DOWN
    _githint_nuclear_bind '^[OB'               githint-arrow-down GITHINT_ORIG_DOWN

    _githint_nuclear_bind '^I' githint-tab GITHINT_ORIG_TAB
    _githint_nuclear_bind '^M' githint-accept-line
}

_githint_restore_key() {
    local key="$1" orig_array="$2"
    [[ -z "$key" ]] && return

    local m
    for m in main emacs viins vicmd; do
        local orig
        eval "orig=\"\${${orig_array}[\$m]}\""
        if [[ -n "$orig" ]]; then
            bindkey -M "$m" "$key" "$orig" >/dev/null 2>&1
        else
            bindkey -M "$m" -r "$key" >/dev/null 2>&1
        fi
    done
}

_githint_unbind_keys() {
    local key
    for key in "${terminfo[kcuu1]}" '^[[A' '^[OA'; do
        _githint_restore_key "$key" GITHINT_ORIG_UP
    done

    for key in "${terminfo[kcud1]}" '^[[B' '^[OB'; do
        _githint_restore_key "$key" GITHINT_ORIG_DOWN
    done

    _githint_restore_key '^I' GITHINT_ORIG_TAB

    local m
    for m in main emacs viins vicmd; do
        bindkey -M "$m" -r '^M' >/dev/null 2>&1
    done
}