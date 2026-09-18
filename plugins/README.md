# Shell plugins

Shell plugins are adapters around the shared git-hint daemon/protocol.

- `zsh/` contains the current Zsh integration.
- `powershell/` is reserved for the PowerShell adapter.

Neither plugin should implement parsing, ranking, providers, or command-data
loading independently.
