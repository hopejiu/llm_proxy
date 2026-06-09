---
title: Autostart
description: Register your application to launch at user login on macOS, Windows, and Linux
image: ../../../assets/images/og-image.png
---

[Skip to content](#%5Ftop) 

# Autostart

## Autostart

[Section titled “Autostart”](#autostart)

`app.Autostart` registers your application to launch automatically when the user logs in. It picks the right native mechanism per platform and resolves symlinked install paths (Homebrew, Scoop) so registrations don’t break when the binary is upgraded.

Registration takes effect on the **next login**, not immediately.

## Quick Start

[Section titled “Quick Start”](#quick-start)

```

import "github.com/wailsapp/wails/v3/pkg/application"


// Register to launch at login

if err := app.Autostart.Enable(); err != nil {

    app.Logger.Error("autostart enable failed", "error", err)

}


// Stop launching at login

if err := app.Autostart.Disable(); err != nil {

    app.Logger.Error("autostart disable failed", "error", err)

}


// Check status

enabled, err := app.Autostart.IsEnabled()


```

## API

[Section titled “API”](#api)

### `Enable`

[Section titled “Enable”](#enable)

Registers the application to launch at login with default options.

```

func (m *AutostartManager) Enable() error


```

Calling `Enable` repeatedly is safe — the registration is overwritten each time, so it’s correct to call on every startup if you’ve persisted the user’s preference.

### `EnableWithOptions`

[Section titled “EnableWithOptions”](#enablewithoptions)

Registers with custom options.

```

func (m *AutostartManager) EnableWithOptions(opts AutostartOptions) error


```

**`AutostartOptions`:**

| Field      | Type       | Description                                                                              |
| ---------- | ---------- | ---------------------------------------------------------------------------------------- |
| Identifier | string     | Overrides the auto-derived registration ID. See “Identifier” below.                      |
| Arguments  | \[\]string | Extra arguments appended to the executable path when launched at login (e.g. \--hidden). |

### `Disable`

[Section titled “Disable”](#disable)

Removes the autostart registration. Returns `nil` if the application wasn’t registered — disable is idempotent.

```

func (m *AutostartManager) Disable() error


```

### `IsEnabled`

[Section titled “IsEnabled”](#isenabled)

Reports whether a registration exists. Fast — doesn’t validate the registered path.

```

func (m *AutostartManager) IsEnabled() (bool, error)


```

### `Status`

[Section titled “Status”](#status)

Returns full registration state.

```

func (m *AutostartManager) Status() (AutostartStatus, error)


```

**`AutostartStatus`:**

| Field    | Type              | Description                                                                                                                  |
| -------- | ----------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| Enabled  | bool              | Whether a registration exists.                                                                                               |
| Path     | string            | On-disk location of the registration artefact (plist path, .desktop path, or registry sub-key). Empty when Enabled is false. |
| Strategy | AutostartStrategy | Which mechanism registered the app (see [Platform Behaviour](#platform-behaviour)).                                          |

## Platform Behaviour

[Section titled “Platform Behaviour”](#platform-behaviour)

* [  macOS ](#tab-panel-58)
* [  Windows ](#tab-panel-59)
* [  Linux ](#tab-panel-60)
* [  iOS / Android / server ](#tab-panel-61)

Two mechanisms are used depending on how the app is packaged:

* **macOS 13+, bundled `.app`**: `SMAppService.mainAppService`. Works for sandboxed apps and Mac App Store builds. No TCC automation prompt (the historical AppleScript approach triggered one).
* **macOS pre-13, or unbundled binary**: a LaunchAgent plist written to `~/Library/LaunchAgents/<identifier>.plist` with `RunAtLoad=true`.

`Status()` returns `AutostartStrategySMAppService` or `AutostartStrategyLaunchAgent` so callers can tell which path was taken.

When the app upgrades from unbundled to bundled, both paths are checked on `Status()` and cleaned up by `Disable()` so an orphaned LaunchAgent doesn’t keep launching the old build.

A registry value is added under `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, with the value name set to the autostart `Identifier` and the data set to the quoted executable path plus any `Arguments`.

Argument quoting follows `CommandLineToArgvW` rules (backslashes doubled before quotes) so paths with spaces or quotes round-trip correctly.

`Status().Strategy` returns `AutostartStrategyRegistryRun`.

An XDG autostart entry is written to `$XDG_CONFIG_HOME/autostart/<identifier>.desktop` (defaulting to `~/.config/autostart/`) with:

```

Type=Application

Hidden=false

X-GNOME-Autostart-enabled=true

Exec=<executable> <arguments>


```

The `Exec` field is escaped per the [freedesktop.org Desktop Entry spec](https://specifications.freedesktop.org/desktop-entry-spec/) — reserved characters (`"`, `` ` ``, `$`, `\\`) are backslash-escaped and the value is double-quoted when it contains whitespace.

`Status().Strategy` returns `AutostartStrategyXDGAutostart`.

Not supported. All methods return `ErrAutostartNotSupported`. Use `errors.Is(err, application.ErrAutostartNotSupported)` to detect this cleanly:

```

if err := app.Autostart.Enable(); err != nil {

    if errors.Is(err, application.ErrAutostartNotSupported) {

        // hide the toggle in the UI

        return

    }

    // real failure — surface it

}


```

## Identifier

[Section titled “Identifier”](#identifier)

If `Options.Identifier` is empty, a default is derived from your app’s name:

| Platform          | Default identifier                                                                             |
| ----------------- | ---------------------------------------------------------------------------------------------- |
| macOS (bundled)   | The app’s bundle identifier, e.g. com.example.MyApp                                            |
| macOS (unbundled) | wails.autostart.<slug>, where <slug> is derived from application.Options.Name                  |
| Windows           | Slug of application.Options.Name (lowercase, non-A-Za-z0-9.\_- stripped, spaces become dashes) |
| Linux             | Same slug as Windows                                                                           |

Identifiers must match `^[A-Za-z0-9._-]+$` and be no longer than 200 characters. Reverse-DNS form is recommended for macOS (it matches how launchd Labels are conventionally written).

When `AutostartOptions.Identifier` is overridden, the same identifier is reused as the registry value name on Windows and the `.desktop` filename on Linux, so a single string identifies the registration cross-platform.

## Stale Detection

[Section titled “Stale Detection”](#stale-detection)

`Disable()` and `Status()` locate the registration by **matching the registered executable path against `os.Executable()` (resolved through any symlinks)**, not by looking up the identifier. This means:

* **Changing the identifier between releases is safe.** The old registration is still discoverable by `Status()` and cleaned up by `Disable()` — as long as the executable path is the same.
* **A second copy of the app at a different path won’t clobber the first’s registration.** Each binary location is tracked independently.
* **Symlinked installs (Homebrew, Scoop) are stable.** `filepath.EvalSymlinks` is applied to `os.Executable()` before matching, so a Homebrew upgrade that swaps the target doesn’t strand the entry.

What this does _not_ cover: if the user moves or renames the binary to an unrelated path, the old registration becomes orphaned (it points at the now-missing file). Apps that ship from a stable install path don’t need to worry about this; apps shipped as portable single-file binaries should call `Disable()` before moving themselves, or always launch through a stable symlink.

## Example

[Section titled “Example”](#example)

```

package main


import (

    "errors"


    "github.com/wailsapp/wails/v3/pkg/application"

)


func main() {

    app := application.New(application.Options{

        Name: "My App",

    })


    // Restore the user's preference on startup

    if userPrefersAutostart() {

        if err := app.Autostart.Enable(); err != nil {

            if !errors.Is(err, application.ErrAutostartNotSupported) {

                app.Logger.Error("autostart", "error", err)

            }

        }

    }


    app.Run()

}


```

A complete runnable example with status / enable / disable buttons is in [examples/autostart/](https://github.com/wailsapp/wails/tree/master/v3/examples/autostart).

```json
{"@context":"https://schema.org","@type":"SoftwareApplication","name":"Wails","description":"Build beautiful desktop applications using Go and modern web technologies.","url":"https://v3.wails.io","applicationCategory":"DeveloperApplication","downloadUrl":"https://github.com/wailsapp/wails/releases","softwareVersion":"3.0","operatingSystem":["Windows","macOS","Linux"],"offers":{"@type":"Offer","price":"0","priceCurrency":"USD"},"author":{"@type":"Organization","name":"Wails Contributors","url":"https://github.com/wailsapp/wails"},"provider":{"@type":"Organization","name":"Wails Contributors","url":"https://github.com/wailsapp/wails"}}
{"@context":"https://schema.org","@type":"Organization","name":"Wails","url":"https://v3.wails.io","logo":"../../../assets/images/favicon.svg","sameAs":["https://github.com/wailsapp/wails","https://x.com/wailsapp","https://discord.gg/JDdSxwjhGf"]}
```
