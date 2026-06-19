# git-hint

Plugin para terminal com sugestões contextuais e documentação em tempo real.

## Arquitetura e Extensibilidade

### History Provider (Interface de Histórico)
Para suportar múltiplos shells (Zsh, Bash, PowerShell) de forma escalável, o sistema utiliza o padrão de **Provider**. 
- O algoritmo de ranking é agnóstico ao shell.
- Cada shell possui seu próprio `HistoryProvider` que implementa a leitura dos comandos específicos do seu ambiente.
- Isso permite a expansão para novos shells apenas adicionando um novo módulo de leitura de histórico.

## Roadmap
(Ver ROADMAP.md para detalhes)
