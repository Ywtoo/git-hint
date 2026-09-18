package state

import "sync"

// Aliases holds the shell aliases sent by the zsh plugin on every request.
// The daemon is long-lived while shells come and go, so aliases always
// travel with the request instead of being discovered once at startup —
// this keeps new `alias foo=...` definitions visible on the next keystroke.
var (
	aliasesMu sync.RWMutex
	aliases   = make(map[string]string)
)

// SetAliases replaces the alias table in one shot.
func SetAliases(next map[string]string) {
	aliasesMu.Lock()
	defer aliasesMu.Unlock()
	aliases = next
	if aliases == nil {
		aliases = make(map[string]string)
	}
}

// GetAliases returns a copy of the current alias table.
func GetAliases() map[string]string {
	aliasesMu.RLock()
	defer aliasesMu.RUnlock()
	out := make(map[string]string, len(aliases))
	for k, v := range aliases {
		out[k] = v
	}
	return out
}

// LookupAlias returns the alias definition for name (ok=false when the name
// is not an alias).
func LookupAlias(name string) (string, bool) {
	aliasesMu.RLock()
	defer aliasesMu.RUnlock()
	v, ok := aliases[name]
	return v, ok
}
