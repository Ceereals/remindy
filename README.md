# remindy

A Bash-based reminder system for [Omarchy](https://github.com/basecamp/omarchy) (Arch Linux + Hyprland). Supports one-time, daily, and weekly reminders with desktop notifications, a Waybar widget, and a Walker/Elephant launcher menu.

<!-- ![screenshot](screenshot.png) -->

## Features

- One-time, daily, and weekly reminders
- Desktop notifications via mako with sound support
- Waybar widget showing next upcoming reminder
- Walker/Elephant launcher integration
- Interactive TUI menus via gum
- CLI mode for scripting
- systemd timer daemon (checks every minute)

## Installation

One-liner:

```bash
bash <(curl -sSL https://raw.githubusercontent.com/Ceereals/remindy/main/install.sh)
```

Or from a local clone:

```bash
git clone https://github.com/Ceereals/remindy.git
cd remindy
./install.sh
```

This will:

- Copy all scripts to `~/.local/bin/`
- Create data and config directories
- Enable the systemd timer daemon

To uninstall:

```bash
bash <(curl -sSL https://raw.githubusercontent.com/Ceereals/remindy/main/uninstall.sh)
```

Or locally: `./uninstall.sh`. Add `--purge` to also remove data and config.

### Dependencies

All pre-installed on Omarchy:

- `jq` - JSON processing
- `gum` - interactive TUI
- `flock` (util-linux) - file locking
- `notify-send` (libnotify) - desktop notifications
- `paplay` (pulseaudio-utils) - notification sounds
- `systemctl` - daemon management
- `xxd` or `od` - ID generation

## Usage

### Interactive menu

```bash
remindy
```

Opens a gum menu to add, list, or remove reminders.

### Add a reminder

Interactive mode (no arguments):

```bash
remindy-add
```

CLI mode:

```bash
# One-time, relative
remindy-add "Standup call" in 30m
remindy-add "Deploy" in 1h30m
remindy-add "Vacation" in 2D

# One-time, absolute
remindy-add "Lunch" at 12:30
remindy-add "Meeting" at tomorrow 14:00

# Daily
remindy-add "Drink water" every day at 09:00

# Weekly
remindy-add "Weekly review" every monday at 10:00
remindy-add "Gym" every monday,wednesday,friday at 18:00
```

#### Time formats

| Format | Example | Description |
|--------|---------|-------------|
| `in <duration>` | `in 30m`, `in 1h30m`, `in 2D` | Relative time from now |
| `at <time>` | `at 14:30`, `at tomorrow 09:00` | Absolute time (auto-bumps to tomorrow if past) |
| `every day at <time>` | `every day at 09:00` | Daily recurring |
| `every <days> at <time>` | `every monday at 10:00` | Weekly recurring |

Duration units: `Y` (years), `M` (months), `D` (days), `h` (hours), `m` (minutes), `s` (seconds).

Day names: `monday`/`mon`, `tuesday`/`tue`, `wednesday`/`wed`, `thursday`/`thu`, `friday`/`fri`, `saturday`/`sat`, `sunday`/`sun`. Comma-separated for multiple days.

### List reminders

```bash
remindy-list
```

Displays all reminders in a table with ID, type, schedule, and text.

### Remove a reminder

Interactive mode (pick from list):

```bash
remindy-remove
```

By ID:

```bash
remindy-remove a1b2c3d4
```

### Daemon management

```bash
remindy-daemon enable   # Create and start systemd timer
remindy-daemon disable  # Stop and remove systemd timer
```

The daemon runs `remindy-check` every minute to send notifications for due reminders.

### Waybar output

```bash
remindy-next
```

Outputs JSON for Waybar's custom module: shows the next upcoming reminder with tooltip listing all of today's reminders.

## Configuration

Config file: `~/.config/remindy/config`

```bash
sound=true                                                        # Play sound on notification
sound_file="$HOME/.local/share/remindy/sounds/remindy.ogg"  # Sound file path
cleanup_hours=24                                                  # Auto-remove notified one-time reminders after N hours
notification_timeout=30000                                        # Notification display time in ms
```

Place your `.ogg` sound file at the `sound_file` path to enable notification sounds.

## Integrations

### Waybar

Add the custom module to your Waybar config (`~/.config/waybar/config.jsonc`):

```jsonc
"custom/remindy": {
    "exec": "remindy-next",
    "return-type": "json",
    "signal": 9,
    "interval": 60,
    "on-click": "remindy",
    "tooltip": true
}
```

Add it to your modules list, e.g.:

```jsonc
"modules-right": ["custom/remindy", ...]
```

Add the CSS to your Waybar stylesheet (`~/.config/waybar/style.css`):

```css
#custom-remindy {
    padding: 0 8px;
}
#custom-remindy.has-reminder {
    color: #f9e2af;
}
#custom-remindy.no-reminder {
    color: #585b70;
    font-size: 0;
}
```

### Mako

Add to your mako config (`~/.config/mako/config`):

```ini
[app-name=Remindy]
default-timeout=30000
border-color=#f9e2af
```

### Hyprland

Add keybindings to your Hyprland config:

```
bindd = SUPER, R, Reminders, exec, uwsm-app -- xdg-terminal-exec -e remindy
bindd = SUPER SHIFT, R, Add reminder, exec, uwsm-app -- xdg-terminal-exec -e remindy-add
```

Check for conflicts with `omarchy-menu-keybindings --print` before adding.

### Walker / Elephant

Copy the Elephant menu plugin:

```bash
cp config/elephant/remindy.lua ~/.config/elephant/menus/remindy.lua
```

Add the Walker prefix to `~/.config/walker/config.toml`:

```toml
[[providers.prefixes]]
prefix = "!"
provider = "menus:remindy"
```

Type `!` in Walker to access reminders.

## Data

Reminders are stored in `~/.local/share/remindy/reminders.json`. If the file becomes corrupted, reset it with:

```bash
echo '{"reminders":[]}' > ~/.local/share/remindy/reminders.json
```

## License

[MIT](LICENSE)
