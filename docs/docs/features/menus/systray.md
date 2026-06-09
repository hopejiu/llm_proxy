---
title: System Tray Menus
description: Creating tray icons with menus, attaching windows, and handling click events.
sidebar_position: 2
---

# System Tray Menus

System tray icons allow your application to run in the background and provide quick access via the operating system's notification area or menu bar.

## Quick Start

```go title="main.go"
package main

import (
    "fmt"

    "github.com/wailsapp/wails/v3/pkg/application"
)

func main() {
    app := application.New(application.Options{})

    tray := app.SystemTray.New("my-tray")
    tray.SetIcon("/path/to/icon.png")
    tray.SetLabel("My App")

    menu := app.NewContextMenu("tray-menu")
    menu.Add("Open", func(ctx *application.ContextMenu) {
        fmt.Println("Open app")
    })
    menu.AddSeparator()
    menu.Add("Quit", func(ctx *application.ContextMenu) {
        app.Quit()
    })
    tray.SetMenu(menu)

    app.Run()
}
```

## Creating a Tray Icon

```go
tray := app.SystemTray.New("unique-id")
tray.SetIcon("/path/to/icon.png")
tray.SetLabel("My App")
```

### Properties

| Method | Description |
|--------|-------------|
| `SetIcon(path)` | Sets the tray icon image. |
| `SetLabel(text)` | Sets a text label (platform-dependent visibility). |
| `SetMenu(menu)` | Associates a context menu with the tray. |

## Event Handlers

```go
tray := app.SystemTray.New("my-tray")
tray.SetIcon("/path/to/icon.png")

tray.OnClick(func() {
    fmt.Println("Left click")
})

tray.OnRightClick(func() {
    fmt.Println("Right click - show menu")
})

tray.OnDoubleClick(func() {
    fmt.Println("Double click")
})

tray.OnMouseEnter(func() {
    fmt.Println("Mouse enter")
})

tray.OnMouseLeave(func() {
    fmt.Println("Mouse leave")
})
```

### Event Reference

| Event | Trigger |
|-------|---------|
| `OnClick` | Left mouse button click. |
| `OnRightClick` | Right mouse button click. |
| `OnDoubleClick` | Double left click. |
| `OnMouseEnter` | Mouse cursor enters tray icon area. |
| `OnMouseLeave` | Mouse cursor leaves tray icon area. |

## Attaching a Window

Attach a window to the tray so it shows and hides on click.

```go
window := app.NewWebviewWindow(webview.WindowOptions{
    Title: "My App",
})

tray := app.SystemTray.New("my-tray")
tray.SetIcon("/path/to/icon.png")
tray.AttachWindow(window)
```

When attached, clicking the tray icon toggles the window visibility.

## Dynamic Updates

Update the tray icon and menu at runtime.

```go
tray.Update(application.SystemTrayUpdate{
    Icon: "/path/to/new-icon.png",
    Label: "Updated Label",
})

menu := app.NewContextMenu("updated-menu")
menu.Add("New Item", handler)
tray.SetMenu(menu)
```

## Platform-Specific Behaviour

### macOS

- Icons appear in the menu bar.
- Use **template icons** (monochrome PNGs) for automatic dark/light mode adaptation.
- Labels are ignored on macOS — use icons only.

```go
// Template icons (rendered in system-appropriate colour)
tray.SetIcon("/path/to/template-icon.png")
```

### Windows

- Icons appear in the notification area (system tray).
- Supports tooltips via `SetLabel`.
- Balloon notifications are not supported via the tray API.

### Linux

- Uses **StatusNotifierItem** (SNI) when available.
- Falls back to legacy system tray on older desktops.
- Tooltip text is supported via `SetLabel`.

## Icon Requirements

| Platform | Format | Recommended Size |
|----------|--------|-----------------|
| macOS | PNG (template) | 22×22 px |
| Windows | ICO or PNG | 16×16 px, 32×32 px |
| Linux | PNG | 22×22 px or 48×48 px |

## Complete Example

```go title="main.go"
package main

import (
    "fmt"

    "github.com/wailsapp/wails/v3/pkg/application"
)

func main() {
    app := application.New(application.Options{})

    window := app.NewWebviewWindow(webview.WindowOptions{
        Title:    "My App",
        Width:    800,
        Height:   600,
        Visible:  false,
    })

    menu := app.NewContextMenu("tray-menu")
    menu.Add("Show Window", func(ctx *application.ContextMenu) {
        window.Show()
    })
    menu.Add("Hide Window", func(ctx *application.ContextMenu) {
        window.Hide()
    })
    menu.AddSeparator()
    menu.Add("Quit", func(ctx *application.ContextMenu) {
        app.Quit()
    })

    tray := app.SystemTray.New("my-tray")
    tray.SetIcon("./assets/tray-icon.png")
    tray.SetLabel("My App")
    tray.SetMenu(menu)
    tray.AttachWindow(window)
    tray.OnDoubleClick(func() {
        fmt.Println("Double-clicked tray icon")
        window.Show()
    })

    app.Run()
}
```

## See Also

- [Window Management](/docs/features/windows)
- [Application Menus](/docs/features/menus/application)
