# hyprlaptop

`hyprlaptop` is a helper for laptops using `hyprland` that are used in both regular laptop mode and docked to a monitor (both clamshell and open).

It listens for:

- External display plugged in/unplugged
- Lid switch opened/closed
- Resume from sleep
- Changes in the config file (live reload)

When any of the above events are received, `hyprlaptop` checks the current display arrangement against the expected arrangement in the config, and makes sure any changes are made to make it match the config.

#### Behavior

- Closing the laptop lid while docked will automatically disable the laptop display
  - Then, opening it will re-enable the laptop display
- Closing the laptop lid while not docked (which would suspend) and then docking will result in only the external being enabled on next wake
- Unplugging the laptop while docked will result in a smooth transition from multi-display to one display

#### Why?

Lots of people have done this via a bash script that runs on lid open/close. I used something similar but it resulted in a lot of the dreaded "oopsie daisy" display because I switch between docked and undocked a lot on my laptop. So I wrote this to provide better error handling and config management.

## Installation

Grab a binary from the latest release, or:

##### From Source

```go
go install github.com/dsrosen6/hyprlaptop@latest
```

##### NixOS

Coming soon!

## Setup

Add the following to your `hyprland` config:

```conf
exec-once = hyprlaptop listen # only if not using UWSM
exec-once = hyprlaptop # initial check at startup
bindl = , switch:off:Lid Switch, exec, hyprlaptop lid
bindl = , switch:on:Lid Switch, exec, hyprlaptop lid
```

If using UWSM with `hyprland`, disregard the first line in the above block and instead create a `systemd` user unit:

1. Take the file `hyprlaptop.service` in this repo and put it in `~/.config/systemd/user/`
2. Run the following commands:

```bash
    systemctl --user daemon-reload
    systemctl --user enable --now hyprlaptop.service
```

Additionally, add the following to your `hypridle` config:  

```conf
general = {
    after_sleep_cmd = hyprlaptop wake`
    # your other general items...
}
```

Log out and back in and everything should be up and running.

## Config

#### Easy Mode

Open your laptop lid and get your displays *exactly* as you would want them if you were using your externals with your lid open (even if you don't really ever do this like me). You can do this via `hyprctl` commands or via your `hyprland` config.  

Run the following command, replacing `eDP-1` if your laptop display is something different:

```bash
hyprlaptop save-displays eDP-1
```

This will freeze your current monitor state into `~/.config/hypr/hyprlaptop.json` with the laptop display as `eDP-1` and the rest as external displays.

#### Manual

You can also set up your config manually. Note that `hyprlaptop` live-reloads your displays when you save the config file.

Here is an example:

```json
{
    "laptop_display": {
        "name": "eDP-1",
        "width": 1920,
        "height": 1200,
        "refreshRate": 60.001,
        "x": 3440,
        "y": 0,
        "scale": 1.25
    },
    "external_displays": {
        "DP-1": {
            "name": "DP-1",
            "width": 3440,
            "height": 1440,
            "refreshRate": 174.96201,
            "x": 0,
            "y": 0,
            "scale": 1
        }
    }
}
```

This config was created via the `save-displays` command; it places the laptop display to the right of the external monitor so that moving the mouse past the right edge moves it to the laptop display.
