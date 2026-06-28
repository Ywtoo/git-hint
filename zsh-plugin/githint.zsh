#!/usr/bin/env zsh

zmodload zsh/terminfo

# Calcula a largura do prompt manualmente para Oh My Zsh
_githint_calc_width() {
    local folder_name=$(basename "$PWD")
    local folder_len=${#folder_name}
    local base_offset=1 # Ajustado de 4 para 1 (remover 3 espaços)

    if git rev-parse --is-inside-work-tree &>/dev/null; then
        local branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null)
        local branch_len=${#branch}
        local git_deco_offset=10 # Tamanho de " git:() ✗ "
        echo $(( base_offset + folder_len + git_deco_offset + branch_len ))
    else
        echo $(( base_offset + folder_len ))
    fi
}

typeset -g GITHINT_SELECTED=0
typeset -g GITHINT_PREV_BUFFER=""
typeset -g GITHINT_PROMPT_COL=0
typeset -g GITHINT_RENDER="ohmyzsh"

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

    # Cálculo manual da largura do prompt (específico para Oh My Zsh)
    GITHINT_PROMPT_COL=$(_githint_calc_width)

    # Disable autosuggest immediately if command starts with 'git'
    if [[ "$buffer" =~ ^git ]]; then
        _zsh_autosuggest_disable
    else
        unset _ZSH_AUTOSUGGEST_DISABLED
        POSTDISPLAY=""
        return
    fi

    local resultado
    resultado=$("$GITHINT_BIN" list "$buffer" "$GITHINT_SELECTED" "$GITHINT_PROMPT_COL" "$GITHINT_RENDER")

    if [[ -n "$resultado" ]]; then
        POSTDISPLAY=$'\n'"${resultado}"
    else
        POSTDISPLAY=""
    fi

    zle reset-prompt
}

zle-line-pre-redraw() { _githint_update}
zle -N zle-line-pre-redraw

# Captura a posição real do cursor via terminal (ESC[6n)
get_cursor_col() {
    # Só executa se estivermos em um terminal interativo
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

    # DEBUG: Log da resposta bruta do terminal
    echo "RAW=[$pos]" >> /tmp/githint-debug.log

    # Extrai a coluna de ESC[linha;colunaR
    local col=$(echo "$pos" | sed -E 's/.*;([0-9]+)R/\1/')
    echo "COLPARSED=[$col]" >> /tmp/githint-debug.log
    echo "${col:-0}"
}

# Interceptador de Enter para capturar a largura do prompt
githint-accept-line() {
    local current_pos=$(get_cursor_col)
    # Largura do Prompt = Posição do Cursor - Comprimento do Buffer
    local prompt_width=$(( current_pos - ${#BUFFER} ))
    [[ $prompt_width -lt 0 ]] && prompt_width=0
    GITHINT_PROMPT_COL=$prompt_width
    zle .accept-line
}
zle -N githint-accept-line

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

# Navigation widgets
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

# Enter binding (captures cursor position for prompt alignment)
_githint_nuclear_bind '^M' githint-accept-line
