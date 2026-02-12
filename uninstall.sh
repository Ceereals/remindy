#!/bin/bash
set -eEo pipefail

INSTALL_DIR="$HOME/.local/share/omarchy/bin"
DATA_DIR="$HOME/.local/share/remindy"
CONFIG_DIR="$HOME/.config/remindy"

SCRIPTS=(
  remindy
  remindy-add
  remindy-check
  remindy-common
  remindy-daemon
  remindy-list
  remindy-next
  remindy-remove
)

# Disable daemon
"$INSTALL_DIR/remindy-daemon" disable 2>/dev/null || true

# Remove scripts
for script in "${SCRIPTS[@]}"; do
  rm -f "$INSTALL_DIR/$script"
done

# Remove data and config if --purge flag passed (safe for piped execution)
if [[ "${1:-}" == "--purge" ]]; then
  rm -rf "$DATA_DIR"
  rm -rf "$CONFIG_DIR"
  echo "Data and configuration removed."
elif [[ -t 0 ]]; then
  # Only prompt if stdin is a terminal (not piped)
  if command -v gum &>/dev/null; then
    if gum confirm "Remove data and configuration too?"; then
      rm -rf "$DATA_DIR"
      rm -rf "$CONFIG_DIR"
      echo "Data and configuration removed."
    fi
  else
    read -rp "Remove data and configuration too? [y/N] " answer
    if [[ "$answer" =~ ^[Yy]$ ]]; then
      rm -rf "$DATA_DIR"
      rm -rf "$CONFIG_DIR"
      echo "Data and configuration removed."
    fi
  fi
else
  echo "Run with --purge to also remove data and config."
fi

echo "Uninstalled."
