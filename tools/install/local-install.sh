#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BIN_DIR="${XDG_BIN_HOME:-$HOME/.local/bin}"
INSTALL_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/githint"
PLUGIN_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/githint/zsh"
ZSHRC="${ZDOTDIR:-$HOME}/.zshrc"
MARKER='# git-hint: managed plugin'
VERSION="${VERSION:-dev}"

install() {
  mkdir -p "$BIN_DIR" "$INSTALL_DIR" "$PLUGIN_DIR"
  (cd "$ROOT" && GOCACHE=/tmp/git-hint-gocache go build -ldflags "-X git-hint/internal/version.Value=$VERSION" -o "$BIN_DIR/githint" .)
  cp "$ROOT"/plugins/zsh/githint.zsh "$PLUGIN_DIR/"
  cp -R "$ROOT"/plugins/zsh/lib "$PLUGIN_DIR/"
  if [[ -f "$ZSHRC" ]] && grep -Fq "$MARKER" "$ZSHRC"; then
    return
  fi
  printf '\n%s\nsource %q/githint.zsh\n' "$MARKER" "$PLUGIN_DIR" >> "$ZSHRC"
}

uninstall() {
  rm -f "$BIN_DIR/githint"
  rm -rf "$PLUGIN_DIR"
  if [[ -f "$ZSHRC" ]]; then
    sed -i.bak "/$MARKER/d;/source .*githint\/zsh\/githint.zsh/d" "$ZSHRC"
    rm -f "$ZSHRC.bak"
  fi
}

case "${1:-install}" in
  install) install; echo "git-hint installed at $BIN_DIR/githint (version $VERSION)" ;;
  update) install; echo "git-hint updated at $BIN_DIR/githint (version $VERSION)" ;;
  uninstall) uninstall; echo "git-hint removed" ;;
  *) echo "usage: $0 [install|update|uninstall]" >&2; exit 2 ;;
esac
