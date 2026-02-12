<div align="center">

# remindy

**A tiny reminder system for your terminal, built for [Omarchy](https://github.com/basecamp/omarchy).**

One-time, daily, and weekly reminders — with desktop notifications, a Waybar widget, and a launcher menu.

[![License: MIT](https://img.shields.io/badge/License-MIT-f9e2af.svg)](LICENSE)
[![Bash](https://img.shields.io/badge/Bash-5.0+-89b4fa.svg)](https://www.gnu.org/software/bash/)
[![Omarchy](https://img.shields.io/badge/Omarchy-skill-cba6f7.svg)](https://github.com/basecamp/omarchy)

<!-- ![demo](https://raw.githubusercontent.com/Ceereals/remindy/main/assets/demo.gif) -->

</div>

---

## Highlights

- **Set it and forget it** — one-time, daily, or weekly schedules
- **Desktop notifications** via mako with optional sound
- **Waybar widget** showing the next upcoming reminder
- **Walker / Elephant** launcher integration
- **Interactive TUI** powered by [gum](https://github.com/charmbracelet/gum)
- **CLI mode** for scripting and aliases
- **systemd timer** daemon — checks every minute, stays out of the way

## Installation

**One-liner:**

```bash
curl -sSL https://raw.githubusercontent.com/Ceereals/remindy/main/install.sh | bash
```

**From a local clone:**

```bash
git clone https://github.com/Ceereals/remindy.git
cd remindy
./install.sh
```

This copies all scripts to `~/.local/bin/`, creates data and config directories, and enables the systemd timer daemon.

<details>
<summary><b>Uninstall</b></summary>

```bash
# Remote
curl -sSL https://raw.githubusercontent.com/Ceereals/remindy/main/uninstall.sh | bash

# Local
./uninstall.sh
```

Add `--purge` to also remove reminders, config, and sounds:

```bash
./uninstall.sh --purge
```

</details>

<details>
<summary><b>Dependencies</b></summary>

All pre-installed on Omarchy:

| Dependency | Purpose |
|---|---|
| `jq` | JSON processing |
| `gum` | Interactive TUI |
| `flock` (util-linux) | File locking |
| `notify-send` (libnotify) | Desktop notifications |
| `paplay` (pulseaudio-utils) | Notification sounds |
| `systemctl` | Daemon management |
| `xxd` / `od` | ID generation |
| GNU `date` | Date arithmetic |

</details>

## Usage

### Interactive menu

```bash
remindy
```

Opens a gum-powered menu to add, list, or remove reminders.

### Adding reminders

Without arguments, `remindy-add` launches an interactive prompt. With arguments, it works as a CLI:

```bash
# Relative time
remindy-add "Standup call" in 30m
remindy-add "Deploy" in 1h30m
remindy-add "Vacation" in 2D

# Absolute time
remindy-add "Lunch" at 12:30
remindy-add "Meeting" at tomorrow 14:00

# Daily
remindy-add "Drink water" every day at 09:00

# Weekly
remindy-add "Weekly review" every monday at 10:00
remindy-add "Gym" every monday,wednesday,friday at 18:00
```

### Listing and removing

```bash
remindy-list              # Show all reminders in a table
remindy-remove            # Interactive picker
remindy-remove a1b2c3d4   # Remove by ID
```

### Daemon

```bash
remindy-daemon enable     # Create and start the systemd timer
remindy-daemon disable    # Stop and remove it
```

The daemon runs `remindy-check` every minute to fire notifications for due reminders.

### Waybar output

```bash
remindy-next
```

Returns JSON for Waybar's custom module — shows the next upcoming reminder with a tooltip listing today's schedule.

## Time formats

| Format | Example | Description |
|---|---|---|
| `in <duration>` | `in 30m`, `in 1h30m`, `in 2D` | Relative — from now |
| `at <time>` | `at 14:30`, `at tomorrow 09:00` | Absolute — bumps to tomorrow if past |
| `every day at <time>` | `every day at 09:00` | Daily recurring |
| `every <days> at <time>` | `every monday,friday at 18:00` | Weekly recurring |

**Duration units:** `Y` years, `M` months, `D` days, `h` hours, `m` minutes, `s` seconds.

**Day names:** `monday`/`mon` through `sunday`/`sun` — comma-separated for multiple days.

## Configuration

Config file: `~/.config/remindy/config`

```bash
sound=true                                                    # Play sound on notification
sound_file="$HOME/.local/share/remindy/sounds/remindy.ogg"   # Sound file path
cleanup_hours=24                                              # Auto-remove fired one-time reminders after N hours
notification_timeout=30000                                    # Notification display time (ms)
```

> [!TIP]
> Drop an `.ogg` file at the `sound_file` path to enable notification sounds.

## Integrations

### Waybar

Add the custom module to `~/.config/waybar/config.jsonc`:

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

Then add `"custom/remindy"` to your modules list and append to `~/.config/waybar/style.css`:

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

> [!NOTE]
> Run `omarchy-restart-waybar` after editing Waybar config.

### Mako

Add to `~/.config/mako/config`:

```ini
[app-name=Remindy]
default-timeout=30000
border-color=#f9e2af
```

### Hyprland

Suggested keybindings for `~/.config/hypr/bindings.conf`:

```
bindd = SUPER, R, Reminders, exec, uwsm-app -- xdg-terminal-exec -e remindy
bindd = SUPER SHIFT, R, Add reminder, exec, uwsm-app -- xdg-terminal-exec -e remindy-add
```

> [!IMPORTANT]
> Check for conflicts with `omarchy-menu-keybindings --print` before adding.

### Walker / Elephant

```bash
cp config/elephant/remindy.lua ~/.config/elephant/menus/remindy.lua
```

Add the prefix to `~/.config/walker/config.toml`:

```toml
[[providers.prefixes]]
prefix = "!"
provider = "menus:remindy"
```

Type `!` in Walker to access reminders.

## Data storage

Reminders live in `~/.local/share/remindy/reminders.json`:

```json
{
  "reminders": [
    { "id": "a1b2c3d4", "text": "Standup call", "type": "once", "time": "2025-02-12T14:30:00", "notified": false },
    { "id": "e5f6g7h8", "text": "Drink water", "type": "daily", "time": "09:00", "last_notified": "2025-02-12" },
    { "id": "i9j0k1l2", "text": "Weekly review", "type": "weekly", "time": "10:00", "days": [1], "last_notified": "2025-02-10" }
  ]
}
```

If the file gets corrupted:

```bash
echo '{"reminders":[]}' > ~/.local/share/remindy/reminders.json
```

## Project structure

```
remindy/
├── bin/
│   ├── remindy                  # Entry point — gum menu
│   ├── remindy-add              # Add reminder (CLI + interactive)
│   ├── remindy-list             # List reminders (gum table)
│   ├── remindy-remove           # Remove reminder (by ID or picker)
│   ├── remindy-check            # Daemon — check & notify
│   ├── remindy-next             # Waybar JSON output
│   ├── remindy-daemon           # Enable/disable systemd timer
│   └── remindy-common           # Shared library (sourced by all)
├── config/
│   ├── config.default           # Default config template
│   ├── waybar-module.jsonc      # Waybar module snippet
│   ├── waybar-style.css         # Waybar CSS snippet
│   ├── mako-rule.conf           # Mako notification rule
│   └── elephant/
│       └── remindy.lua          # Walker/Elephant menu plugin
├── sounds/.gitkeep
├── install.sh
├── uninstall.sh
└── LICENSE
```

## License

[MIT](LICENSE)
