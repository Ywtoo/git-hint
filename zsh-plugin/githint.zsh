#!/usr/bin/env zsh

zmodload zsh/terminfo
zmodload zsh/net/socket
zmodload zsh/datetime

GITHINT_PLUGIN_DIR="${${(%):-%N}:A:h}"
local dir="$GITHINT_PLUGIN_DIR"

if (( ${+functions[_githint_cleanup]} )); then
    _githint_cleanup
fi

source "$dir/lib/state.zsh"
source "$dir/lib/daemon.zsh"
source "$dir/lib/prompt.zsh"
source "$dir/lib/highlight.zsh"
source "$dir/lib/autosuggest.zsh"
source "$dir/lib/update.zsh"
source "$dir/lib/widgets.zsh"
source "$dir/lib/bindings.zsh"
source "$dir/lib/lifecycle.zsh"

_githint_init
_githint_enable