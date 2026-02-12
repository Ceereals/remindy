#!/bin/bash
set -eEo pipefail

REPO="Ceereals/remindy"
REPO_URL="https://github.com/$REPO.git"

INSTALL_DIR="$HOME/.local/bin"
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

# Determine source directory: local clone or temp git clone
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" 2>/dev/null && pwd)"

if [[ -f "$SCRIPT_DIR/remindy-common" ]]; then
  SRC_DIR="$SCRIPT_DIR"
  CLEANUP=""
else
  echo "Cloning $REPO..."
  SRC_DIR=$(mktemp -d)
  CLEANUP="$SRC_DIR"
  git clone --depth 1 "$REPO_URL" "$SRC_DIR" 2>/dev/null
fi

# Detect existing installation
UPDATING=false
if [[ -f "$INSTALL_DIR/remindy" ]]; then
  UPDATING=true
  echo "Existing installation detected — updating..."
  echo ""
  echo "⚠  Check the release page for breaking changes:"
  echo "   https://github.com/$REPO/releases"
  echo ""
fi

# Install scripts
mkdir -p "$INSTALL_DIR"
for script in "${SCRIPTS[@]}"; do
  cp "$SRC_DIR/$script" "$INSTALL_DIR/$script"
  chmod +x "$INSTALL_DIR/$script"
done

# Create data and config directories
mkdir -p "$DATA_DIR/sounds"
mkdir -p "$CONFIG_DIR"

# Copy default config if not present
if [[ ! -f "$CONFIG_DIR/config" ]]; then
  cat > "$CONFIG_DIR/config" <<'EOF'
sound=true
sound_file="$HOME/.local/share/remindy/sounds/remindy.ogg"
cleanup_hours=24
notification_timeout=30000
EOF
fi

# Initialize empty reminders JSON if not present
if [[ ! -f "$DATA_DIR/reminders.json" ]]; then
  echo '{"reminders":[]}' > "$DATA_DIR/reminders.json"
fi

# Enable daemon (re-enable on update to pick up any service file changes)
"$INSTALL_DIR/remindy-daemon" enable

# Cleanup temp dir
[[ -n "$CLEANUP" ]] && rm -rf "$CLEANUP"

echo ""
if [[ "$UPDATING" == true ]]; then
  echo "Updated! Scripts are in $INSTALL_DIR"
  echo "Your config and reminders were preserved."
else
  echo "Installed! Scripts are in $INSTALL_DIR"
fi
echo ""
echo "Integration snippets are at: https://github.com/$REPO/tree/main/config"
echo "  - Waybar module:  config/waybar-module.jsonc + config/waybar-style.css"
echo "  - Mako rule:      config/mako-rule.conf"
echo "  - Hyprland keys:  config/hypr-remindy.conf"
echo "  - Walker/Elephant: config/elephant/remindy.lua + config/walker/"
