---
title: Frameless Windows
description: Create frameless windows with CSS-based drag regions and custom window chrome
---

# Frameless Windows

Wails provides frameless window support with CSS-based drag regions and platform-native behaviour. Remove the platform-native title bar for complete control over window chrome, custom designs, and unique user experiences whilst maintaining essential functionality like dragging, resizing, and system controls.

## Creating a Frameless Window

```go
window := app.Window.NewWithOptions(application.WebviewWindowOptions{
    Title:     "Frameless App",
    Width:     800,
    Height:    600,
    Frameless: true,
})
```

## CSS-Based Drag Regions

Use the `--wails-draggable` CSS property:

```css
/* Draggable area */
.titlebar {
    --wails-draggable: drag;
}

/* Non-draggable elements within draggable area */
.titlebar button {
    --wails-draggable: no-drag;
}
```

### Values

- `drag` - Area is draggable
- `no-drag` - Area is not draggable (even if parent is)

## Resize Handles

Wails provides automatic resize handles for frameless windows:

```css
/* Enable resize on all edges */
body {
    --wails-resize: all;
}

/* Or specific edges */
.resize-top {
    --wails-resize: top;
}
.resize-bottom {
    --wails-resize: bottom;
}
.resize-left {
    --wails-resize: left;
}
.resize-right {
    --wails-resize: right;
}

/* Corners */
.resize-top-left {
    --wails-resize: top-left;
}
.resize-top-right {
    --wails-resize: top-right;
}
.resize-bottom-left {
    --wails-resize: bottom-left;
}
.resize-bottom-right {
    --wails-resize: bottom-right;
}
```

## Platform-Specific Behaviour

### Windows

Windows frameless windows support snap layouts and custom decorations:

```go
window := app.Window.NewWithOptions(application.WebviewWindowOptions{
    Frameless: true,
    Windows: application.WindowsWindow{
        DisableFramelessWindowDecorations: false,
    },
})
```

- Snap layouts support (Windows 11)
- Custom title bar height: Windows automatically detects drag regions from CSS
- Disable decorations:

```go
Windows: application.WindowsWindow{
    DisableFramelessWindowDecorations: true,
},
```

### macOS

macOS supports transparent title bars and full-size content views:

```go
window := app.Window.NewWithOptions(application.WebviewWindowOptions{
    Frameless: true,
    Mac: application.MacWindow{
        TitleBar: application.MacTitleBar{
            AppearsTransparent: true,
        },
        InvisibleTitleBarHeight: 40,
    },
})
```

## Common Patterns

### Modern Title Bar

```html
<div class="titlebar">
    <div class="titlebar-title">My Application</div>
    <div class="titlebar-controls">
        <button class="minimize">-</button>
        <button class="maximize">+</button>
        <button class="close">x</button>
    </div>
</div>
```

```css
.titlebar {
    --wails-draggable: drag;
    display: flex;
    justify-content: space-between;
    align-items: center;
    height: 40px;
    background: #2c3e50;
    color: white;
    padding: 0 10px;
}

.titlebar button {
    --wails-draggable: no-drag;
    background: transparent;
    border: none;
    color: white;
    cursor: pointer;
    width: 30px;
    height: 30px;
}

.titlebar button:hover {
    background: rgba(255, 255, 255, 0.1);
}
```

### Splash Screen

```go
splash := app.Window.NewWithOptions(application.WebviewWindowOptions{
    Title:     "Loading...",
    Width:     400,
    Height:    300,
    Frameless: true,
    AlwaysOnTop: true,
})

// Show splash
splash.Show()

// Initialise application
time.Sleep(2 * time.Second)

// Hide splash, show main window
splash.Close()
mainWindow.Show()
```

### Rounded Window

```css
body {
    --wails-draggable: drag;
    border-radius: 12px;
    overflow: hidden;
}
```

### Overlay Window

```go
overlay := app.Window.NewWithOptions(application.WebviewWindowOptions{
    Title:           "Overlay",
    Width:           300,
    Height:          200,
    Frameless:       true,
    AlwaysOnTop:     true,
    BackgroundType: application.BackgroundTypeTransparent,
    IgnoreMouseEvents: false,
})
```

## Complete Example

```go
package main

import (
    _ "embed"

    "github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed frontend/dist
var assets embed.FS

func main() {
    app := application.New(application.Options{
        Name: "Frameless App",
        Assets: application.AssetOptions{
            Handler: application.AssetFileServerFS(assets),
        },
    })

    window := app.Window.NewWithOptions(application.WebviewWindowOptions{
        Title:     "Frameless Application",
        Width:     1000,
        Height:    700,
        MinWidth:  800,
        MinHeight: 600,
        Frameless: true,
        Mac: application.MacWindow{
            TitleBar: application.MacTitleBar{
                AppearsTransparent: true,
            },
            InvisibleTitleBarHeight: 40,
        },
        Windows: application.WindowsWindow{
            DisableFramelessWindowDecorations: false,
        },
    })

    window.Center()
    window.Show()
    app.Run()
}
```
