---
title: "Window API"
description: "Complete reference for the Window API"
---

## Overview

The Window API provides methods to control window appearance, behaviour, and lifecycle. Access through window instances or the `app.Window` manager.

## Visibility

### Show()

```go
func (w *WebviewWindow) Show() Window
```

### Hide()

```go
func (w *WebviewWindow) Hide() Window
```

### Close()

```go
func (w *WebviewWindow) Close()
```

## Window Properties

### SetTitle()

```go
func (w *WebviewWindow) SetTitle(title string) Window
```

### Name()

```go
func (w *WebviewWindow) Name() string
```

## Size and Position

### SetSize()

```go
func (w *WebviewWindow) SetSize(width, height int) Window
```

### Size()

```go
func (w *WebviewWindow) Size() (width, height int)
```

### SetMinSize() / SetMaxSize()

```go
func (w *WebviewWindow) SetMinSize(width, height int) Window
func (w *WebviewWindow) SetMaxSize(width, height int) Window
```

### SetPosition()

```go
func (w *WebviewWindow) SetPosition(x, y int)
```

### Position()

```go
func (w *WebviewWindow) Position() (x, y int)
```

### Center()

```go
func (w *WebviewWindow) Center()
```

### Focus()

```go
func (w *WebviewWindow) Focus()
```

## Window State

### Minimise() / UnMinimise()

```go
func (w *WebviewWindow) Minimise() Window
func (w *WebviewWindow) UnMinimise()
```

### Maximise() / UnMaximise()

```go
func (w *WebviewWindow) Maximise() Window
func (w *WebviewWindow) UnMaximise()
```

### Fullscreen() / UnFullscreen() / ToggleFullscreen()

```go
func (w *WebviewWindow) Fullscreen() Window
func (w *WebviewWindow) UnFullscreen()
func (w *WebviewWindow) ToggleFullscreen()
```

### IsMinimised() / IsMaximised() / IsFullscreen()

```go
func (w *WebviewWindow) IsMinimised() bool
func (w *WebviewWindow) IsMaximised() bool
func (w *WebviewWindow) IsFullscreen() bool
```

## Window Content

### SetURL()

```go
func (w *WebviewWindow) SetURL(url string) Window
```

### SetHTML()

```go
func (w *WebviewWindow) SetHTML(html string) Window
```

### Reload()

```go
func (w *WebviewWindow) Reload()
```

## Window Events

### OnWindowEvent()

```go
func (w *WebviewWindow) OnWindowEvent(
    eventType events.WindowEventType,
    callback func(event *WindowEvent),
) func()
```

### RegisterHook()

```go
func (w *WebviewWindow) RegisterHook(
    eventType events.WindowEventType,
    callback func(event *WindowEvent),
) func()
```

Hooks run before listeners and can prevent the event by calling `event.Cancel()`.

### EmitEvent()

```go
func (w *WebviewWindow) EmitEvent(name string, data ...any) bool
```

## Other Methods

### SetEnabled(), SetBackgroundColour(), SetResizable(), SetAlwaysOnTop()

```go
func (w *WebviewWindow) SetEnabled(enabled bool)
func (w *WebviewWindow) SetBackgroundColour(colour RGBA) Window
func (w *WebviewWindow) SetResizable(resizable bool) Window
func (w *WebviewWindow) SetAlwaysOnTop(alwaysOnTop bool) Window
```

### Print()

```go
func (w *WebviewWindow) Print() error
```

### AttachModal()

```go
func (w *WebviewWindow) AttachModal(modalWindow Window)
```

## Platform-Specific Options

### Linux MenuStyle

```go
window := app.Window.NewWithOptions(application.WebviewWindowOptions{
    Title: "My Application",
    Linux: application.LinuxWindow{
        MenuStyle: application.LinuxMenuStylePrimaryMenu,
    },
})
```

| Value                     | Description                                         |
| ------------------------- | --------------------------------------------------- |
| LinuxMenuStyleMenuBar     | Traditional menu bar below the title bar (default)  |
| LinuxMenuStylePrimaryMenu | Primary menu button in the header bar (GNOME style) |
