# Architecture

git-hint has one shared completion engine and thin shell integrations.

```text
core/                 shared data model and filesystem paths
engine/               tokenization, navigation, ranking, rendering, providers
registry/             index and command-spec resolution
state/                short-lived completion state/cache
app/                  application orchestration
daemon/               Unix socket transport
scraper/              fallback discovery/crawling/parser
data/fig/             imported, editable Fig command specs
data/local/           local overrides and crawler output
tools/                offline import/generation utilities
plugins/zsh/          Zsh integration
plugins/powershell/   future PowerShell integration
```

Fig data is the primary command knowledge. Local overrides take precedence
when explicitly present, and the scraper is used only when no usable spec is
available. Shell plugins must remain adapters: completion behavior belongs in
the shared engine, not in Zsh or PowerShell code.

