#!/usr/bin/env zsh
# ============================================================================
# githint — autocomplete inteligente para comandos git no zsh
# Inspirado no Fig (descontinuado), preenche o vácuo no Linux:
# completação inline com descrição documental em tempo real, ranking por
# histórico pessoal e placeholders dinâmicos (branches, commits, remotes...)
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
typeset -g GITHINT_AUTOSUGGEST_STATE="on"   # "on" | "off" — controla transição única
typeset -g __githint_clean=""
typeset -ga GITHINT_OWN_HIGHLIGHTS           # rastreia só as nossas entradas em region_highlight
typeset -gA GITHINT_ORIG_UP                  # widget original das setas — por keymap
typeset -gA GITHINT_ORIG_DOWN

# ----------------------------------------------------------------------------
# Resolução do binário — calcula o caminho esperado uma vez (sem exigir que
# o arquivo já exista), e cada chamada de _githint_update revalida antes de
# usar. Isso evita precisar dar "source" de novo toda vez que você roda
# `go build` durante o desenvolvimento — o caminho já está certo desde o
# início, só o conteúdo do binário muda.
#
# Usamos %x (nome do ARQUIVO-FONTE atual) em vez de %N (que dentro de uma
# função retorna o nome da FUNÇÃO, não do arquivo — testado e confirmado).
# ----------------------------------------------------------------------------
_githint_resolve_bin_path() {
    local plugin_dir="${${(%):-%x}:A:h}"

    if [[ -x "$plugin_dir/githint" ]]; then
        print -r -- "$plugin_dir/githint"
        return
    fi
    if [[ -x "$plugin_dir/bin/githint" ]]; then
        print -r -- "$plugin_dir/bin/githint"
        return
    fi
    if command -v githint >/dev/null 2>&1; then
        command -v githint
        return
    fi

    # Ainda não existe (ex: antes do primeiro `go build`) — aponta mesmo
    # assim pro caminho mais provável; _githint_update revalida a cada uso.
    print -r -- "$plugin_dir/githint"
}

GITHINT_BIN="$(_githint_resolve_bin_path)"
typeset -g GITHINT_MISSING_WARNED=0

# ----------------------------------------------------------------------------
# Cálculo de largura do prompt (Oh My Zsh)
# Mede o comprimento visual do prompt pra alinhar a lista de sugestões
# exatamente abaixo do cursor — específico pro tema com decoração git.
# ----------------------------------------------------------------------------
_githint_calc_width() {
    local folder=${PWD:t}
    local folder_len=${#folder}
    local base_offset=2  # Ajustado para remover espaços excedentes

    if git rev-parse --is-inside-work-tree &>/dev/null; then
        local branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null)
        local branch_len=${#branch}
        local git_deco_offset=9  # Ajustado para remover espaços excedentes
        echo $(( base_offset + folder_len + git_deco_offset + branch_len ))
    else
        echo $(( base_offset + folder_len ))
    fi
}

# ----------------------------------------------------------------------------
# Destaque de cor (sentinelas → region_highlight)
#
# ANSI cru (\x1b[...m) não funciona no POSTDISPLAY — o ZLE sanitiza qualquer
# byte de controle e exibe como texto literal (ex: "^[[32m"). O único
# mecanismo real é region_highlight: um array de "início fim especificação"
# que o ZLE aplica na hora de desenhar as células do terminal.
#
# Protocolo de comunicação com o binário Go:
#   \x01 ... \x02  → verde escuro fg=28  (comentário "# descrição")
#   \x03 ... \x04  → cinza       fg=8   (nome do item não-selecionado)
#
# A função separa o texto limpo das posições coloridas, sem usar $() pra
# evitar subshell — modificações em region_highlight morreriam na subshell.
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
            (( i++ ))  # consome o marcador de fechamento
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
        GITHINT_OWN_HIGHLIGHTS+=("$entry")  # registra como "nossa" pra remoção seletiva
    done
}

# Remove do region_highlight APENAS as entradas que o githint colocou,
# preservando qualquer cor de outros plugins (ex: zsh-syntax-highlighting).
# Usar region_highlight=() seria agressivo demais — apagaria a cor verde
# do "git" digitado e qualquer outro destaque de terceiros.
_githint_clear_own_highlights() {
    (( ${#GITHINT_OWN_HIGHLIGHTS} == 0 )) && return

    local -a kept
    local entry own found
    for entry in "${region_highlight[@]}"; do
        found=0
        for own in "${GITHINT_OWN_HIGHLIGHTS[@]}"; do
            if [[ "$entry" == "$own" ]]; then
                found=1
                break
            fi
        done
        (( found == 0 )) && kept+=("$entry")
    done

    region_highlight=("${kept[@]}")
    GITHINT_OWN_HIGHLIGHTS=()
}

# ----------------------------------------------------------------------------
# Interop com zsh-autosuggestions
#
# O hook _zsh_autosuggest_highlight_apply (registrado em zle-line-pre-redraw)
# pinta TODO o POSTDISPLAY com fg=8 (cinza), sobrescrevendo nosso destaque.
# Solução: neutralizar o estilo enquanto o githint está ativo, restaurar ao sair.
#
# Otimização: só alternamos no momento da TRANSIÇÃO (git ↔ não-git), não a
# cada tecla — evita chamar _zsh_autosuggest_disable repetidamente quando
# você já está no meio de "git commit -m ...".
# ----------------------------------------------------------------------------
_githint_autosuggest_off() {
    if [[ -z "$GITHINT_ORIG_AUTOSUGGEST_STYLE" ]]; then
        GITHINT_ORIG_AUTOSUGGEST_STYLE="${ZSH_AUTOSUGGEST_HIGHLIGHT_STYLE:-fg=8}"
    fi
    _zsh_autosuggest_disable 2>/dev/null
    ZSH_AUTOSUGGEST_HIGHLIGHT_STYLE="none"
}

_githint_autosuggest_on() {
    ZSH_AUTOSUGGEST_HIGHLIGHT_STYLE="${GITHINT_ORIG_AUTOSUGGEST_STYLE:-fg=8}"
    unset _ZSH_AUTOSUGGEST_DISABLED
}

# ----------------------------------------------------------------------------
# Atualização da lista de sugestões — chamada a cada redraw do ZLE
# ----------------------------------------------------------------------------
_githint_update() {
    local buffer="$BUFFER"

    # Reset de seleção só quando o buffer muda de fato.
    # Se GITHINT_SELECTED < 0, estamos navegando no histórico; preservamos o
    # estado pra que a seta pra baixo (↓) consiga retornar dessa navegação.
    if [[ "$buffer" != "$GITHINT_PREV_BUFFER" ]]; then
        if (( GITHINT_SELECTED >= 0 )); then
            GITHINT_SELECTED=0
        fi
        GITHINT_PREV_BUFFER="$buffer"
    fi

    GITHINT_PROMPT_COL=$(_githint_calc_width)

    # Transição de contexto: git ↔ não-git (só na mudança, não em todo redraw)
    if [[ "$buffer" =~ ^git ]]; then
        if [[ "$GITHINT_AUTOSUGGEST_STATE" != "off" ]]; then
            _githint_autosuggest_off
            GITHINT_AUTOSUGGEST_STATE="off"
        fi
    else
        if [[ "$GITHINT_AUTOSUGGEST_STATE" != "on" ]]; then
            _githint_autosuggest_on
            GITHINT_AUTOSUGGEST_STATE="on"
        fi
        _githint_clear_own_highlights
        POSTDISPLAY=""
        return
    fi

    if [[ ! -x "$GITHINT_BIN" ]]; then
        if (( ! GITHINT_MISSING_WARNED )); then
            print -u2 ""
            print -u2 "githint: binário não encontrado em '$GITHINT_BIN'."
            print -u2 "         Rode 'go build -o \"$GITHINT_BIN\"' — não precisa dar source de novo depois."
            GITHINT_MISSING_WARNED=1
        fi
        return
    fi
    GITHINT_MISSING_WARNED=0

    local resultado
    resultado=$("$GITHINT_BIN" list "$buffer" "$GITHINT_SELECTED" "$GITHINT_PROMPT_COL" "$GITHINT_RENDER")

    _githint_clear_own_highlights  # remove só o que é nosso — preserva outros plugins

    if [[ -n "$resultado" ]]; then
        local base=$(( ${#buffer} + 1 ))
        _githint_apply_highlight "$resultado" "$base"
        POSTDISPLAY=$'\n'"$__githint_clean"
    else
        POSTDISPLAY=""
    fi

    zle reset-prompt
}

# ----------------------------------------------------------------------------
# Registro do hook — usa add-zle-hook-widget, não "zle -N zle-line-pre-redraw"
#
# "zle -N zle-line-pre-redraw" puro SOBRESCREVE qualquer coisa já registrada
# ali (testado e confirmado: apaga silenciosamente o hook de outros plugins
# que já usam o sistema moderno, como o zsh-autosuggestions, dependendo da
# ordem de carregamento). add-zle-hook-widget ADICIONA à cadeia em vez de
# substituir — é a MESMA técnica que o autosuggestions usa, o "envelope" que
# não atropela ninguém. Com isso, a posição do githint na lista de plugins
# deixa de importar: funciona antes, depois, ou entre outros que também
# usem esse mecanismo.
# ----------------------------------------------------------------------------
autoload -Uz add-zle-hook-widget
_githint_pre_redraw_hook() { _githint_update; }
zle -N _githint_pre_redraw_hook
add-zle-hook-widget zle-line-pre-redraw _githint_pre_redraw_hook

# ----------------------------------------------------------------------------
# Captura da posição real do cursor (ESC[6n)
# Usada no Enter pra medir a largura exata do prompt e calibrar o alinhamento.
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
#
# Fallback de seta: quando o Go sinaliza "modo histórico" (usuário saiu da
# lista de sugestões), usamos o widget ORIGINAL que estava mapeado antes do
# githint sobrescrever — preserva busca por prefixo (history-substring-search)
# ou qualquer outro comportamento que o usuário tinha configurado.
# ----------------------------------------------------------------------------
_githint_key_handler() {
    local tecla="$1"
    local fallback_widget="$2"
    local resultado widget new_sel new_buffer

    resultado=$("$GITHINT_BIN" key "$tecla" "$GITHINT_SELECTED" "$BUFFER")

    if [[ -z "$resultado" ]]; then
        # Binário não respondeu: executa o widget original sem interferência
        [[ -n "$fallback_widget" ]] && zle "$fallback_widget"
        _githint_update
        return
    fi

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
        # Go sinalizou fallback pro histórico: usa o widget original capturado,
        # não um valor fixo — preserva history-substring-search ou equivalente
        if [[ "$widget" == "up-line-or-history" || "$widget" == "down-line-or-history" ]]; then
            [[ -n "$fallback_widget" ]] && zle "$fallback_widget"
        else
            zle "$widget"
        fi
    fi

    [[ "$tecla" == "TAB" ]] && zle reset-prompt
    _githint_update
}

githint-arrow-up()   { _githint_key_handler "arrowUP"   "${GITHINT_ORIG_UP[$KEYMAP]:-up-line-or-history}"; }
githint-arrow-down() { _githint_key_handler "arrowDOWN" "${GITHINT_ORIG_DOWN[$KEYMAP]:-down-line-or-history}"; }
githint-tab()        { _githint_key_handler "TAB" "expand-or-complete"; }

zle -N githint-arrow-up
zle -N githint-arrow-down
zle -N githint-tab

# ----------------------------------------------------------------------------
# Key bindings — captura widgets originais ANTES de sobrescrever
#
# Ordem importa: capturamos primeiro, depois sobrescrevemos.
# Isso preserva history-substring-search, up-line-or-beginning-search,
# ou qualquer outro widget de seta que o usuário tinha configurado.
# ----------------------------------------------------------------------------
_githint_nuclear_bind() {
    local key="$1" widget="$2" orig_array="$3"
    [[ -z "$key" ]] && return

    local m
    for m in main emacs viins vicmd; do
        # Captura o widget atual ANTES de sobrescrever.
        # Usamos eval para indireção (nameref/typeset -n não está disponível
        # em todas as builds de zsh — eval funciona em qualquer versão).
        if [[ -n "$orig_array" ]]; then
            local orig
            orig=$(bindkey -M "$m" "$key" 2>/dev/null | awk '{print $2}' | tr -d "'")
            if [[ -n "$orig" && "$orig" != "undefined-key" && "$orig" != "$widget" ]]; then
                eval "${orig_array}[\$m]=\"\$orig\""
            fi
        fi

        bindkey -M "$m" -r "$key" 2>/dev/null
        bindkey -M "$m" "$key" "$widget"
    done
}

# Seta pra cima — captura widget original em GITHINT_ORIG_UP
_githint_nuclear_bind "${terminfo[kcuu1]}" githint-arrow-up GITHINT_ORIG_UP
_githint_nuclear_bind '^[[A'               githint-arrow-up GITHINT_ORIG_UP
_githint_nuclear_bind '^[OA'               githint-arrow-up GITHINT_ORIG_UP
_githint_nuclear_bind '^P'                 githint-arrow-up GITHINT_ORIG_UP

# Seta pra baixo — captura widget original em GITHINT_ORIG_DOWN
_githint_nuclear_bind "${terminfo[kcud1]}" githint-arrow-down GITHINT_ORIG_DOWN
_githint_nuclear_bind '^[[B'               githint-arrow-down GITHINT_ORIG_DOWN
_githint_nuclear_bind '^[OB'               githint-arrow-down GITHINT_ORIG_DOWN
_githint_nuclear_bind '^N'                 githint-arrow-down GITHINT_ORIG_DOWN

# TAB e Enter — sem fallback de captura (comportamento fixo)
_githint_nuclear_bind '^I' githint-tab
_githint_nuclear_bind '^M' githint-accept-line