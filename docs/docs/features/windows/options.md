---
title: Window Options
description: Complete reference for WebviewWindowOptions - all window configuration options across platforms
---

# Window Options

Wails provides comprehensive window configuration with dozens of options for size, position, appearance, and behaviour. This reference covers all available options across Windows, macOS, and Linux as the complete reference for `WebviewWindowOptions`.

## WebviewWindowOptions

```go
type WebviewWindowOptions struct {
    // Identity
    Name  string
    Title string

    // Size and Position
    Width           int
    Height          int
    X               int
    Y               int
    MinWidth        int
    MinHeight       int
    MaxWidth        int
    MaxHeight       int
    InitialPosition WindowStartPosition // WindowCentered (default) or WindowXY
    Screen          *Screen             // target screen for initial placement

    // Initial State
    Hidden        bool
    Frameless     bool
    DisableResize bool        // inverted vs v2's `Resizable`
    AlwaysOnTop   bool
    StartState    WindowState // WindowStateNormal | Minimised | Maximised | Fullscreen

    // Appearance
    BackgroundColour RGBA
    BackgroundType   BackgroundType
    Zoom             float64
    ZoomControlEnabled bool

    // Content
    URL  string
    HTML string
    JS   string
    CSS  string

    // Behaviour
    EnableFileDrop              bool
    IgnoreMouseEvents           bool
    HideOnFocusLost             bool
    DevToolsEnabled             bool
    DefaultContextMenuDisabled  bool
    ContentProtectionEnabled    bool
    KeyBindings                 map[string]func(window *WebviewWindow)

    // Window-control button states
    MinimiseButtonState ButtonState
    MaximiseButtonState ButtonState
    CloseButtonState    ButtonState

    // Menu
    UseApplicationMenu bool

    // Platform-specific (per-window)
    Mac     MacWindow
    Windows WindowsWindow
    Linux   LinuxWindow
}
```

## Options Reference

### Name

| Property | Value |
|----------|-------|
| Type | `string` |
| Default | Auto-generated |

The unique identifier for the window. Used for programmatic access and window management. If not provided, Wails generates one automatically.

### Title

| Property | Value |
|----------|-------|
| Type | `string` |
| Default | `""` |
| Platform | All |

The text displayed in the title bar.

### Width / Height

| Property | Value |
|----------|-------|
| Type | `int` |
| Default | `800` / `600` |
| Platform | All |

The initial dimensions of the window in pixels.

### X / Y

| Property | Value |
|----------|-------|
| Type | `int` |
| Default | `0` |
| Platform | All |

The initial position of the window from the left/top of the screen.

### MinWidth / MinHeight

| Property | Value |
|----------|-------|
| Type | `int` |
| Default | `0` (no minimum) |
| Platform | All |

The minimum allowed dimensions for the window.

### MaxWidth / MaxHeight

| Property | Value |
|----------|-------|
| Type | `int` |
| Default | `0` (no maximum) |
| Platform | All |

The maximum allowed dimensions for the window.

### Hidden

| Property | Value |
|----------|-------|
| Type | `bool` |
| Default | `false` |
| Platform | All |

When `true`, the window is created but not shown until `window.Show()` is called.

### Frameless

| Property | Value |
|----------|-------|
| Type | `bool` |
| Default | `false` |
| Platform | All |

Removes the platform-native title bar and window borders. When enabled, you must implement:
- Draggable regions using `--wails-draggable: drag`
- System buttons (close, minimise, maximise)
- Resize handles

### DisableResize

| Property | Value |
|----------|-------|
| Type | `bool` |
| Default | `false` |
| Platform | All |

When `true`, prevents the user from resizing the window. Note this is inverted vs v2's `Resizable`.

### AlwaysOnTop

| Property | Value |
|----------|-------|
| Type | `bool` |
| Default | `false` |
| Platform | All |

When `true`, the window stays above all other windows.

### StartState

| Property | Value |
|----------|-------|
| Type | `WindowState` |
| Default | `WindowStateNormal` |
| Platform | All |

The initial window state. Valid values:
- `WindowStateNormal` - Normal state
- `WindowStateMinimised` - Minimised
- `WindowStateMaximised` - Maximised
- `WindowStateFullscreen` - Fullscreen

### BackgroundColour

| Property | Value |
|----------|-------|
| Type | `RGBA` |
| Default | White |
| Platform | All |

The background colour of the webview. Set using `application.NewRGB(r, g, b)`.

### BackgroundType

| Property | Value |
|----------|-------|
| Type | `BackgroundType` |
| Default | `BackgroundTypeSolid` |
| Platform | All |

Controls the background rendering. Use `BackgroundTypeTransparent` for transparent windows.

### URL

| Property | Value |
|----------|-------|
| Type | `string` |
| Default | `""` |
| Platform | All |

The URL to load in the webview. Use `http://wails.localhost/` during development.

### HTML

| Property | Value |
|----------|-------|
| Type | `string` |
| Default | `""` |
| Platform | All |

Raw HTML content to load directly instead of a URL.

### EnableFileDrop

| Property | Value |
|----------|-------|
| Type | `bool` |
| Default | `false` |
| Platform | All |

When `true`, allows users to drag and drop files onto the window.

### ContentProtectionEnabled

| Property | Value |
|----------|-------|
| Type | `bool` |
| Default | `false` |
| Platform | All |

When `true`, prevents the window content from being captured by screen recording or screenshot tools.

### IgnoreMouseEvents

| Property | Value |
|----------|-------|
| Type | `bool` |
| Default | `false` |
| Platform | All |

When `true`, mouse events pass through the window.

### HideOnFocusLost

| Property | Value |
|----------|-------|
| Type | `bool` |
| Default | `false` |
| Platform | All |

When `true`, the window hides when it loses focus.

### DevToolsEnabled

| Property | Value |
|----------|-------|
| Type | `bool` |
| Default | `false` |
| Platform | All |

When `true`, enables browser dev tools in the webview.

### DefaultContextMenuDisabled

| Property | Value |
|----------|-------|
| Type | `bool` |
| Default | `false` |
| Platform | All |

When `true`, disables the default right-click context menu.

### KeyBindings

| Property | Value |
|----------|-------|
| Type | `map[string]func(window *WebviewWindow)` |
| Default | `nil` |
| Platform | All |

Custom keyboard shortcuts mapped to window actions.

## Platform-Specific Options

### Mac Options

```go
type MacWindow struct {
    TitleBar            MacTitleBar
    InvisibleTitleBarHeight int
    Backdrop            MacBackdropType
}

type MacTitleBar struct {
    AppearsTransparent bool
    FullSizeContentView bool
    HideToolbarSeparator bool
    HideTitleBar bool
    ToolbarShown bool
}
```

### Windows Options

```go
type WindowsWindow struct {
    DisableIcon                       bool
    BackdropType                      WindowsBackdropType
    CustomTheme                       ThemeSettings
    DisableFramelessWindowDecorations bool
}
```

### Linux Options

```go
type LinuxWindow struct {
    DisableIcon bool
}
```

## Complete Example

```go
package main

import (
    "github.com/wailsapp/wails/v3/pkg/application"
)

func main() {
    app := application.New(application.Options{
        Name: "My Application",
    })

    window := app.Window.NewWithOptions(application.WebviewWindowOptions{
        Title:            "My Application",
        Width:            1024,
        Height:           768,
        MinWidth:         800,
        MinHeight:        600,
        MaxWidth:         1920,
        MaxHeight:        1080,
        BackgroundColour: application.NewRGB(255, 255, 255),
        URL:              "http://wails.localhost/",
    })

    window.Center()
    window.Show()
    app.Run()
}
```
