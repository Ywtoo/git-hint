# Roadmap — Plugin Zsh com Engine em Go

---

# Visão do produto

Plugin para terminal com:
> sugestões contextuais + documentação em tempo real + memória de uso do usuário

Sem substituir shell ou terminal.
Atua apenas como camada inteligente sobre o prompt.

---

# Objetivos principais

Prioridades absolutas:

1. baixa latência
2. UX estável
3. zero travamentos
4. renderização limpa
5. sugestões realmente úteis

---

# Stack consolidada

## Shell/UI

* Zsh
* ZLE hooks
* render ANSI

## Engine

* Go

## Persistência

* SQLite → estado do usuário
* JSON → documentação estática

---

# Arquitetura geral

```text
Zsh (UI)
   ↓
Captura BUFFER
   ↓
Debounce (~40ms)
   ↓
Request → Engine
   ↓
Resposta estruturada
   ↓
Render abaixo do prompt
```

---

# Roadmap

# 🟢 Fase 1: Captura Literal e Sincronização em Tempo Real
**O que é:** Configurar o script Zsh (via ganchos do ZLE) para atuar como o "sistema nervoso".

**Objetivo:** Capturar o que o usuário digita no buffer ($BUFFER), incluindo caracteres, espaços e backspaces, enviando a string em tempo real para a Engine Go.

**Foco:** Garantir comunicação estável em 0ms sem quebrar o Oh My Zsh ou temas do terminal.

## Implementar
* [x ] Hook ZLE
* [ x] Captura de `$BUFFER`
* [ x] Chamada ao engine Go (em 0ms)
* [ ] Estabilidade com Oh My Zsh/Temas

---

# 🟡 Fase 2: O Sistema de UI (A Interface Fantasma)
**O que é:** Criar a camada visual do Git Hint que renderiza a lista de sugestões logo abaixo do cursor do usuário.

**Objetivo:** Ler o retorno da Engine Go e desenhar a listagem de forma discreta na tela.

## Implementar
* [ x ] Renderização abaixo do prompt
* [ ] Desenho da listagem (JSON output)
* [ ] Limite de sugestões
* [ ] Timeout de engine
* [ ] Fallback seguro

---

# 🟠 Fase 3: Navegação e Injeção Incremental (Tab e Setas)
**O que é:** Interceptar as teclas de navegação para realizar o completamento inteligente de código.

**Objetivo:** Fazer o avanço inteligente palavra por palavra (bloco lógico por bloco lógico).

## Implementar
* [ x ] Navegação ↑ ↓
* [ ] Injeção por blocos lógicos (ex: `git c` -> `git commit `)
* [ ] Super Tab (Injeção de mensagens de commits passados no campo de mensagem)
* [ ] Aceitação de sugestão via TAB

---

# 🔵 Fase 4: O Algoritmo de Histórico Dinâmico (Módulo de Ranking)
**O que é:** Acoplar a lógica de ordenação na Engine Go baseada no uso real do usuário.

**Objetivo:** Ler os últimos 500 comandos git do ~/.zsh_history em cache na memória, isolar e limpar as duas primeiras palavras de cada comando, e usar um Map para contar a frequência.

## Implementar
* [ ] Leitura de `~/.zsh_history`
* [ ] Módulo de Ranking (Map de frequência)
* [ ] Ordenação dinâmica (Comandos mais usados no topo)

---

# 🎨 Fase 5: Polimento de UI e UX Refinada
**O que é:** Garantir uma UI limpa, profissional e extremamente fluida através de decisões de design estratégico.

## Implementar
1. **Ocultação por Relevância (Filtro de Dicas):**
   * [ ] Comandos/flags comuns (score alto) exibem apenas a palavra nua.
   * [ ] Dicas/descrições apenas para comandos raros (score baixo).
2. **Delay de Hesitação Inteligente (Anti-Flicker):**
   * [ ] Autocompletar (palavra) em 0ms.
   * [ ] Renderização de explicações estáticas apenas após pausa de 40ms-50ms.
3. **Truncamento Dinâmico por Foco:**
   * [ ] Descrições longas cortadas com `...` em uma linha.
   * [ ] Expansão vertical (até 2 linhas) ao focar via seta para baixo.
4. **Abandono de Animações Contínuas:**
   * [ ] Decisão de não usar marquee/slides para evitar travamentos de renderização de terminal.

---

# Fase 6 — Daemon persistente (Arquitetura de Latência Mínima)

**Objetivo:** Migrar de spawn de binário para comunicação via Socket Unix.

## Implementar
* [ ] Migração para `githintd` (daemon Go)
* [ ] Socket Unix para comunicação Zsh ↔ Engine
* [ ] Cache persistente em RAM
* [ ] Warm startup e compartilhamento entre terminais

---

# Fase 7 — Expansão do ecossistema

## Shells
* [ ] Bash
* [ ] Fish

## Comandos
* [ ] Docker
* [ ] Kubernetes
* [ ] npm/pnpm
* [ ] cargo
* [ ] systemctl

---

# Estratégia de performance

## Regras fundamentais

Nunca:
* bloquear shell
* esperar IO síncrono longo
* recalcular tudo
* renderizar excessivamente

Sempre:
* cache agressivo
* debounce
* lazy updates
* fallback rápido

---

# Estratégia de segurança

Se qualquer parte falhar:
```text
terminal continua funcionando normalmente
```

## Implementar
* [ ] timeout de requests
* [ ] engine opcional
* [ ] recovery automático
* [ ] logs separados
* [ ] modo degradado

---

# Estrutura inicial do projeto

```text
project/
├── zsh-plugin/
│   └── githint.zsh
│
├── engine/
│   ├── main.go
│   ├── parser/
│   ├── ranking/
│   ├── cache/
│   └── storage/
│
├── data/
│   └── git.json
│
├── internal/
│   └── protocol/
│
├── scripts/
│
└── README.md
```

---

# Insight central

O projeto NÃO é apenas autocomplete.

É:

> uma camada de interpretação contextual + memória operacional da CLI em tempo real

com restrição severa de latência e UX.

---

# Conclusão técnica

## Escolhas corretas

* Go para engine
* Zsh para integração
* SQLite para estado
* JSON para docs
* arquitetura desacoplada

## Maiores desafios reais

* renderização terminal
* UX sem flicker
* compatibilidade shell
* latência percebida
* qualidade das sugestões

## Menores preocupações

* memória
* CPU
* capacidade do Go
* tamanho dos dados
