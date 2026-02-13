#!/bin/bash
set -eEo pipefail

REPO="Ceereals/remindy"
REPO_URL="https://github.com/$REPO.git"

INSTALL_DIR="$HOME/.local/share/omarchy/bin"
DATA_DIR="$HOME/.local/share/remindy"
CONFIG_DIR="$HOME/.config/remindy"

# Check required dependencies
for cmd in jq gum flock notify-send systemctl; do
  command -v "$cmd" &>/dev/null || { echo "Missing dependency: $cmd" >&2; exit 1; }
done

SCRIPTS=(
  remindy
  remindy-add
  remindy-check
  remindy-common
  remindy-daemon
  remindy-list
  remindy-migrate
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
fi

# Install scripts
mkdir -p "$INSTALL_DIR"
for script in "${SCRIPTS[@]}"; do
  cp "$SRC_DIR/$script" "$INSTALL_DIR/$script"
  chmod +x "$INSTALL_DIR/$script"
done

# Install migrations
mkdir -p "$INSTALL_DIR/migrations"
for migration in "$SRC_DIR"/migrations/*.sh; do
  [[ -f "$migration" ]] || continue
  cp "$migration" "$INSTALL_DIR/migrations/"
done

# Create data and config directories
mkdir -p "$DATA_DIR/sounds"
mkdir -p "$CONFIG_DIR"

# Copy default config if not present
if [[ ! -f "$CONFIG_DIR/config" ]]; then
  cat > "$CONFIG_DIR/config" <<'EOF'
sound=true
sound_file="$HOME/.local/share/remindy/sounds/remindy.ogg"
notification_timeout=30000
EOF
fi

# Initialize empty reminders JSON if not present
if [[ ! -f "$DATA_DIR/reminders.json" ]]; then
  echo '{"reminders":[]}' > "$DATA_DIR/reminders.json"
fi

# Run migrations
MIGRATIONS_DIR="$SRC_DIR/migrations" "$INSTALL_DIR/remindy-migrate"

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
echo "  - Elephant:       config/elephant/remindy/ (make install)"
