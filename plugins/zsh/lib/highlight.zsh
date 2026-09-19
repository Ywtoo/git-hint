# ----------------------------------------------------------------------------
# Color highlighting (sentinels → region_highlight)
# ----------------------------------------------------------------------------
_githint_apply_highlight() {
    local raw="$1"
    local base=$2

    # Defensive sweep before every redraw: leftovers from the previous render
    # (or from another plugin rewriting region_highlight) would otherwise
    # stack up and leak color into rows that no longer exist.
    _githint_clear_own_highlights

    local clean=""
    local pos=$base
    local i=1

    local -a ranges
    local entry

    while (( i <= ${#raw} )); do
        local char="${raw[i]}"

        if [[ "$char" == $'\x01' || "$char" == $'\x03' || "$char" == $'\x05' || "$char" == $'\x07' ]]; then
            local style="fg=2"
            local closer=$'\x02'

            if [[ "$char" == $'\x03' ]]; then
                style="fg=8"
                closer=$'\x04'
            elif [[ "$char" == $'\x05' ]]; then
                # Truecolor sem bold: paletas remapeiam ANSI 0 pra tom claro e
                # bold em alguns temas troca pela variante bright (o
                # "contorno" estranho). #000000 sobre #ffffff, peso normal.
                style="fg=#000000,bg=#ffffff"
                closer=$'\x06'
            elif [[ "$char" == $'\x07' ]]; then
                style="fg=#008000,bg=#ffffff"
                closer=$'\x08'
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
            ranges+=("$start $pos $style")

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
    local style

    for entry in "${region_highlight[@]}"; do
        found=0

        for own_entry in "${GITHINT_OWN_HIGHLIGHTS[@]}"; do
            if [[ "$entry" == "$own_entry" ]]; then
                found=1
                break
            fi
        done

        if (( ! found )); then
            # Extra sweep: an entry may survive an earlier clear when another
            # plugin (zsh-syntax-highlighting, vi-mode) rewrote or reset the
            # array between our calls, making exact-string matching miss it.
            # Anything carrying one of OUR styles is ours — drop it so colors
            # never leak into other plugins' regions or stale rows.
            style="${entry#* }"
            style="${style#* }"
            if [[ "$style" == *"bg=#ffffff"* || "$style" == "fg=2"* || "$style" == *"fg=2"* || "$style" == "fg=8"* ]]; then
                found=1
            fi
        fi

        (( ! found )) && kept+=("$entry")
    done

    region_highlight=("${kept[@]}")
    GITHINT_OWN_HIGHLIGHTS=()
}
