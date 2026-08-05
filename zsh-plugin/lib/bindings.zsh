# ----------------------------------------------------------------------------
# Key bindings
# ----------------------------------------------------------------------------

_githint_nuclear_bind() {
    local key="$1" widget="$2" orig_array="$3"
    [[ -z "$key" ]] && return

    local m
    for m in main emacs viins vicmd; do
        if [[ -n "$orig_array" ]]; then
            local original
            original=$(bindkey -M "$m" "$key" 2>/dev/null | awk '{print $2}' | tr -d "'")

            if [[ -n "$original" &&
                  "$original" != "undefined-key" &&
                  "$original" != "$widget" ]]; then
                eval "${orig_array}[\$m]=\"\$original\""
            fi
        fi

        bindkey -M "$m" -r "$key" 2>/dev/null
        bindkey -M "$m" "$key" "$widget"
    done
}

_githint_bind_keys() {
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
}

_githint_restore_key() {
    local key="$1" orig_array="$2"
    [[ -z "$key" ]] && return

    local m
    for m in main emacs viins vicmd; do
        local -n ref="$orig_array"
        local orig="${ref[$m]}"
        if [[ -n "$orig" ]]; then
            bindkey -M "$m" "$key" "$orig"
        else
            bindkey -M "$m" -r "$key" 2>/dev/null
        fi
    done
}

_githint_unbind_keys() {
    local key
    for key in "${terminfo[kcuu1]}" '^[[A' '^[OA' '^P'; do
        _githint_restore_key "$key" GITHINT_ORIG_UP
    done

    for key in "${terminfo[kcud1]}" '^[[B' '^[OB' '^N'; do
        _githint_restore_key "$key" GITHINT_ORIG_DOWN
    done

    local m
    for m in main emacs viins vicmd; do
        bindkey -M "$m" -r '^I' 2>/dev/null
        bindkey -M "$m" -r '^M' 2>/dev/null
    done
}