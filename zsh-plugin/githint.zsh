#!/usr/bin/env zsh
# ============================================================================
# githint — autocomplete inteligente para comandos git no zsh
# ============================================================================

zmodload zsh/terminfo

# ----------------------------------------------------------------------------
# Estado global do plugin
# ----------------------------------------------------------------------------
typeset -g GITHINT_SELECTED=0
typeset -g GITHINT_PREV_BUFFER=""
typeset -g GITHINT_PROMPT_COL=0
typeset -g GITHINT_RENDER="ohmyzsh"
typeset -g GITHINT_ORIG_AUTOSUGGEST_STYLE=""
typeset -g __githint_clean=""

# TODO: Path to githint binary
GITHINT_BIN="/media/storage_fixed/Programming/git-hint/githint"

# ----------------------------------------------------------------------------
# Cálculo de largura do prompt (específico para Oh My Zsh)
# ----------------------------------------------------------------------------
_githint_calc_width() {
    local folder_name=$(basename "$PWD")
    local folder_len=${#folder_name}
    local base_offset=1

    if git rev-parse --is-inside-work-tree &>/dev/null; then
        local branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null)
        local branch_len=${#branch}
        local git_deco_offset=10 # Tamanho de " git:() ✗ "
        echo $(( base_offset + folder_len + git_deco_offset + branch_len ))
    else
        echo $(( base_offset + folder_len ))
    fi
}

# ----------------------------------------------------------------------------
# Destaque de cor (sentinelas -> region_highlight)
#
# O binário Go emite texto puro com marcadores de controle ao redor dos
# trechos que devem ser coloridos:
#   \x01 ... \x02  -> verde escuro  (comentário, ex: "# descrição")
#   \x03 ... \x04  -> cinza         (nome do item, quando não selecionado)
#
# Essa função separa o texto limpo (sem marcadores) das posições que devem
# receber cor, e popula region_highlight -- único mecanismo do ZLE que
# realmente aplica cor no POSTDISPLAY (ANSI cru não funciona, é sanitizado).
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
            (( i++ )) # consome o marcador de fechamento
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
    done
}

# ----------------------------------------------------------------------------
# Interop com zsh-autosuggestions
#
# O plugin autosuggestions colore TODO o POSTDISPLAY com sua própria cor
# (fg=8 por padrão) via um hook independente em zle-line-pre-redraw, o que
# sobrescreve nosso destaque. Enquanto estivermos mostrando sugestões de
# comando git, desligamos o autosuggest e neutralizamos seu estilo.
# ----------------------------------------------------------------------------
_githint_autosuggest_off() {
    if [[ -z "$GITHINT_ORIG_AUTOSUGGEST_STYLE" ]]; then
        GITHINT_ORIG_AUTOSUGGEST_STYLE="${ZSH_AUTOSUGGEST_HIGHLIGHT_STYLE:-fg=8}"
    fi
    _zsh_autosuggest_disable
    ZSH_AUTOSUGGEST_HIGHLIGHT_STYLE="none"
}

_githint_autosuggest_on() {
    ZSH_AUTOSUGGEST_HIGHLIGHT_STYLE="$GITHINT_ORIG_AUTOSUGGEST_STYLE"
    unset _ZSH_AUTOSUGGEST_DISABLED
}

# ----------------------------------------------------------------------------
# Atualização da lista de sugestões
#
# Nota: NÃO fazemos cache do resultado para pular redraws "iguais". Isso foi
# tentado antes para evitar flicker, mas é arriscado: outros pontos do código
# (TAB, troca de contexto git/não-git, o próprio reset-prompt reentrante)
# podem alterar o que está na tela sem que essa cache saiba, causando telas
# em branco ou desatualizadas. Se a lentidão do processo Go for o problema
# real por trás do flicker, ela deve ser resolvida ali (cache/async no
# provider), não escondida aqui.
# ----------------------------------------------------------------------------
_githint_update() {
    local buffer="$BUFFER"

    if [[ "$buffer" != "$GITHINT_PREV_BUFFER" ]]; then
        GITHINT_SELECTED=0
        GITHINT_PREV_BUFFER="$buffer"
    fi

    GITHINT_PROMPT_COL=$(_githint_calc_width)

    if [[ "$buffer" =~ ^git ]]; then
        if [[ "$GITHINT_AUTOSUGGEST_STATE" == "on" ]]; then
            _githint_autosuggest_off
            GITHINT_AUTOSUGGEST_STATE="off"
        fi
    else
        if [[ "$GITHINT_AUTOSUGGEST_STATE" == "off" ]]; then
            _githint_autosuggest_on
            GITHINT_AUTOSUGGEST_STATE="on"
        fi
        POSTDISPLAY=""
        return
    fi

    local resultado
    resultado=$("$GITHINT_BIN" list "$buffer" "$GITHINT_SELECTED" "$GITHINT_PROMPT_COL" "$GITHINT_RENDER")

    region_highlight=()

    if [[ -n "$resultado" ]]; then
        local base=$(( ${#buffer} + 1 ))
        _githint_apply_highlight "$resultado" "$base"
        POSTDISPLAY=$'\n'"$__githint_clean"
    else
        POSTDISPLAY=""
    fi

    zle reset-prompt
}

zle-line-pre-redraw() { _githint_update; }
zle -N zle-line-pre-redraw

# ----------------------------------------------------------------------------
# Captura da posição real do cursor (ESC[6n) — usada para alinhar o prompt
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
# Navegação (setas e TAB) — delega a decisão pro binário Go
# ----------------------------------------------------------------------------
_githint_key_handler() {
    local tecla="$1"
    local resultado
    local widget new_sel new_buffer

    resultado=$("$GITHINT_BIN" key "$tecla" "$GITHINT_SELECTED" "$BUFFER")
    [[ -z "$resultado" ]] && { _githint_update; return; }

    widget="${resultado%%|*}"
    local remainder="${resultado#*|}"
    new_sel="${remainder%%|*}"
    new_buffer="${remainder#*|}"

    [[ -n "$new_sel" ]] && GITHINT_SELECTED=$new_sel

    if [[ -n "$new_buffer" ]]; then
        BUFFER="$new_buffer"
        CURSOR=${#BUFFER}
    fi

    if [[ -n "$widget" && "$widget" != '""' ]]; then
        zle "$widget"
    fi

    [[ "$tecla" == "TAB" ]] && zle reset-prompt

    _githint_update
}

githint-arrow-up()   { _githint_key_handler "arrowUP"; }
githint-arrow-down() { _githint_key_handler "arrowDOWN"; }
githint-tab()        { _githint_key_handler "TAB"; }
zle -N githint-arrow-up
zle -N githint-arrow-down
zle -N githint-tab

# ----------------------------------------------------------------------------
# Key bindings — cobre múltiplos keymaps e fallbacks de terminal
# ----------------------------------------------------------------------------
_githint_nuclear_bind() {
    local key="$1" widget="$2"
    [[ -z "$key" ]] && return

    local m
    for m in main emacs viins vicmd; do
        bindkey -M "$m" -r "$key" 2>/dev/null
        bindkey -M "$m" "$key" "$widget"
    done
}

_githint_nuclear_bind "${terminfo[kcuu1]}" githint-arrow-up  # seta fisica pra cima
_githint_nuclear_bind '^[[A'               githint-arrow-up  # fallback ANSI
_githint_nuclear_bind '^[OA'               githint-arrow-up  # fallback modo aplicação
_githint_nuclear_bind '^P'                 githint-arrow-up  # emacs Ctrl+P

_githint_nuclear_bind "${terminfo[kcud1]}" githint-arrow-down
_githint_nuclear_bind '^[[B'               githint-arrow-down
_githint_nuclear_bind '^[OB'               githint-arrow-down
_githint_nuclear_bind '^N'                 githint-arrow-down

_githint_nuclear_bind '^I' githint-tab       # TAB — sobrescreve o completion nativo
_githint_nuclear_bind '^M' githint-accept-line # Enter — captura posição do cursor