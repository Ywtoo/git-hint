# ----------------------------------------------------------------------------
# Color highlighting (sentinels → region_highlight)
# ----------------------------------------------------------------------------
_githint_apply_highlight() {
    local raw="$1"
    local base=$2

    local clean=""
    local pos=$base
    local i=1

    local -a ranges
    local entry

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

    for entry in "${ranges[@]}"; do
        region_highlight+=("$entry")
        GITHINT_OWN_HIGHLIGHTS+=("$entry")
    done
}

_githint_clear_own_highlights() {
    (( ${#GITHINT_OWN_HIGHLIGHTS} == 0 )) && return

    local -a kept
    local entry
    local own_entry
    local found

    for entry in "${region_highlight[@]}"; do
        found=0

        for own_entry in "${GITHINT_OWN_HIGHLIGHTS[@]}"; do
            if [[ "$entry" == "$own_entry" ]]; then
                found=1
                break
            fi
        done

        (( ! found )) && kept+=("$entry")
    done

    region_highlight=("${kept[@]}")
    GITHINT_OWN_HIGHLIGHTS=()
}