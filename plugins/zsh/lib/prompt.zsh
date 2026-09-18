# ----------------------------------------------------------------------------
# Prompt width calculation
# ----------------------------------------------------------------------------
_githint_calc_width() {
    local folder="${PWD:t}"
    local base_offset=2

    local width=$(( base_offset + ${#folder} ))

    if git rev-parse --is-inside-work-tree &>/dev/null; then
        local branch
        branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null)

        local git_deco_offset=9
        width=$(( width + git_deco_offset + ${#branch} ))
    fi

    print -r -- "$width"
}

_githint_calc_width_cached() {
    if [[ "$PWD" != "$GITHINT_PROMPT_COL_DIR" ]]; then
        GITHINT_PROMPT_COL_CACHE="$(_githint_calc_width)"
        GITHINT_PROMPT_COL_DIR="$PWD"
    fi

    print -r -- "$GITHINT_PROMPT_COL_CACHE"
}

# ----------------------------------------------------------------------------
# Cursor position (ESC[6n)
# ----------------------------------------------------------------------------
_githint_get_cursor_col() {
    [[ -t 0 ]] || {
        print -r -- 0
        return
    }

    local oldstty
    oldstty=$(stty -g 2>/dev/null) || {
        print -r -- 0
        return
    }

    local col=0

    {
        stty raw -echo min 0 time 5 2>/dev/null

        printf '\e[6n' > /dev/tty 2>/dev/null

        local pos=""
        local ch

        while read -t 0.05 -k 1 ch 2>/dev/null; do
            pos+="$ch"
            [[ "$ch" == R ]] && break
        done

        col=$(sed -E 's/.*;([0-9]+)R/\1/' <<<"$pos")
    } always {
        stty "$oldstty" 2>/dev/null
    }

    print -r -- "${col:-0}"
}