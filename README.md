# hyprdocked

`hyprdocked` is a laptop display helper for Hyprland that updates the display state based on a few factors.

## How it Works

`hyprdocked` listens for the following events:

- Displays being added or removed (via [Hyprland's IPC](https://wiki.hypr.land/IPC/))
- Laptop lid events (when it is opened or closed)
- `hyprdocked idle` or `hyprdocked resume` events (keep reading for details on this)

Any time one of the above events are received, `hyprdocked` applies settings based on the following statuses if changes are needed.

The laptop display is *enabled* if the following statuses are detected:

- Docked with lid opened
- Laptop only, lid opened

The laptop display is *disabled* if the device is detected as docked with lid closed.

By default, closing the lid in laptop-only mode triggers the same lock and suspend sequence as `hyprdocked idle`.

### Special Case: `hyprdocked idle` / `hyprdocked resume`

If `hyprdocked idle` is called (typically by your idle daemon), `hyprdocked` will:

1. Run your lock command
2. Enable the laptop display so it is ready when the device wakes back up
3. Suspend the device

The display stays enabled until `hyprdocked resume` is called, which releases this state and lets normal dock/lid logic take over again.

Why? If your laptop suspends while docked (lid closed) and you then unplug from the dock and open the lid, the laptop display is still disabled. With zero displays active, Hyprland shows its blank-screen fallback. Routing suspend through `hyprdocked idle` prevents this.

This needs some manual wiring because `hyprdocked` does not assume which idle utility you use. See the [Idle Daemon](#idle-daemon) section.

## Installation

> **Requires Hyprland v0.55 or newer.** `hyprdocked` uses `hyprctl eval` with Lua to manage monitors, which was introduced in v0.55.

### AUR

```
yay -S hyprdocked-bin
```

The service file is installed automatically. Skip to [Auto-Run](#auto-run).

### From Source

```
go install github.com/dsrosen6/hyprdocked@latest
```

Then copy the service file from the repo to your user systemd directory:

```
cp systemd/hyprdocked.service ~/.config/systemd/user/
```

## Configuration

### Auto-Run

The daemon is started via `hyprdocked listen`. Wire it up with a systemd user service or directly in your Hyprland config.

**systemd (recommended for UWSM users):**

```
systemctl --user enable hyprdocked --now
```

**Hyprland config (v0.55+):**

```lua
hl.on("hyprland.start", function()
    hl.exec_cmd("hyprdocked listen")
end)
```

### Identify Laptop Display

Run `hyprctl monitors` and find your laptop display. The default assumed name is `eDP-1`. If yours is different, set `laptop` in your config file (see below).

*If you need to add a common name, raise an issue — happy to expand the auto-detection list.*

### Hyprland Monitors

Make sure all your monitors are configured and enabled in your Hyprland config. At minimum, your laptop display must be defined so `hyprdocked` can restore its settings.

**Hyprland v0.55+ (Lua):**

```lua
hl.monitor({
    output = "DP-1",
    mode = "3440x1440@174.96",
    position = "0x0",
    scale = "1.0",
})

hl.monitor({
    output = "eDP-1",
    mode = "1920x1200",
    position = "3440x480",
    scale = "1.25",
})
```

### Config File

The config file lives at `~/.config/hypr/hyprdocked.yaml`. All settings have sensible defaults and the file is entirely optional — only set what you need to change.

```yaml
laptop: eDP-1          # laptop display name
settle-window: 1       # seconds to wait after an event before processing

log-file: ""           # path to log file (default: stderr); env vars like $HOME are expanded

lock-on-idle: true     # run lock command when hyprdocked idle is called
lock-cmd: ""           # lock command to run (default: "pidof hyprlock || hyprlock")
lock-delay: 1          # seconds to wait after locking before continuing

suspend-idle: true     # suspend device when hyprdocked idle is called
suspend-closed: true   # suspend device on lid close in laptop-only mode
suspend-delay: 1       # seconds to wait after enabling display before suspending

sequential-hooks: false  # run post-hooks one at a time instead of concurrently
post-hooks: []
```

Run `hyprdocked check-cfg` to verify your config is being read correctly and see all active values.

### Post-Hooks

Post-hooks are shell commands that run after `hyprdocked` processes an event. Each hook has two fields:

- `command` — the shell command to run
- `on-status-change` — if `true`, only runs when the display actually changed state (e.g. laptop display was enabled or disabled). If `false` (the default), runs on every processed event.

```yaml
post-hooks:
  - command: "notify-send 'display updated'"
    on-status-change: false
  - command: "~/.config/hypr/scripts/reload-waybar.sh"
    on-status-change: true
```

Hooks run concurrently by default. Set `sequential-hooks: true` to run them one at a time in order.

### Idle Daemon

If you're using `hypridle`, wire `hyprdocked idle` into your suspend timeout and `hyprdocked resume` into the after-sleep hook. Use `hyprdocked lock` to run your configured lock command directly without triggering suspend.

```ini
general {
    lock_cmd = hyprdocked lock
    after_sleep_cmd = hyprdocked resume
}

listener {
    # If hyprlock is already active when the device wakes, suspend again quickly.
    timeout = 30
    on-timeout = sh -c 'pidof hyprlock >/dev/null && hyprdocked idle'
}

listener {
    timeout = 540
    on-timeout = hyprdocked lock
}

listener {
    timeout = 600
    on-timeout = hyprdocked idle
}
```

Translate to your idle agent if you use a different one.
