---
title: Multiple Windows
description: Build multi-window applications with window tracking, communication, and lifecycle management
---

# Multiple Windows

Wails supports creating and managing multiple windows in a single application. Each window has its own lifecycle, events, and webview instance.

## Creating Windows

```go
// Create a window with options
window := app.Window.NewWithOptions(application.WebviewWindowOptions{
    Title:  "My Window",
    Width:  800,
    Height: 600,
    URL:    "http://wails.localhost/",
})

// Show the window
window.Show()
```

## Window Tracking

Keep references to windows you create for later access:

```go
type App struct {
    windows map[string]*application.WebviewWindow
}

func (a *App) CreateWindow(name string) {
    a.windows[name] = app.Window.NewWithOptions(application.WebviewWindowOptions{
        Name:  name,
        Title: name,
        Width: 800,
        Height: 600,
    })
    a.windows[name].Show()
}

func (a *App) GetWindow(name string) *application.WebviewWindow {
    return a.windows[name]
}
```

## Window Communication

Windows can communicate via events:

```go
// In main window
app.Event.Emit("data-updated", map[string]interface{}{
    "value": 42,
})

// In settings window
app.Event.On("data-updated", func(event *application.WailsEvent) {
    data := event.Data.(map[string]interface{})
    value := data["value"].(int)
    fmt.Printf("Received: %d\n", value)
})
```

## Common Patterns

### Singleton Window

Ensure only one instance of a specific window exists:

```go
type App struct {
    settingsWindow *application.WebviewWindow
}

func (a *App) OpenSettings() {
    if a.settingsWindow != nil {
        a.settingsWindow.Show()
        a.settingsWindow.Focus()
        return
    }
    a.settingsWindow = app.Window.NewWithOptions(application.WebviewWindowOptions{
        Title:  "Settings",
        Width:  600,
        Height: 400,
        URL:    "http://wails.localhost/settings",
    })
    a.settingsWindow.Show()
}
```

### Document Windows

Multiple windows for editing different documents:

```go
func (a *App) OpenDocument(docID string) {
    window := app.Window.NewWithOptions(application.WebviewWindowOptions{
        Name:   "doc-" + docID,
        Title:  "Document: " + docID,
        Width:  1024,
        Height: 768,
        URL:    "http://wails.localhost/document?id=" + docID,
    })
    window.Show()
}
```

### Tool Palettes

Small utility windows that float above the main window:

```go
func (a *App) OpenToolPalette() {
    palette := app.Window.NewWithOptions(application.WebviewWindowOptions{
        Title:       "Tools",
        Width:       300,
        Height:      500,
        AlwaysOnTop: true,
        Frameless:   true,
        URL:         "http://wails.localhost/tools",
    })
    palette.Show()
}
```

### Modal Dialogs

Windows that block interaction with the parent:

```go
func (a *App) ShowModalDialog(title, message string) {
    modal := app.Window.NewWithOptions(application.WebviewWindowOptions{
        Title:       title,
        Width:       400,
        Height:      200,
        AlwaysOnTop: true,
        DisableResize: true,
        URL:         "http://wails.localhost/modal?msg=" + message,
    })
    modal.Show()
}
```

## Parent-Child Relationships

```go
func (a *App) OpenChild() {
    child := app.Window.NewWithOptions(application.WebviewWindowOptions{
        Title:  "Child Window",
        Width:  400,
        Height: 300,
        URL:    "http://wails.localhost/child",
    })
    child.Show()

    // Close child when parent closes
    parent.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
        child.Close()
    })
}
```

## Window Lifecycle Management

```go
func (a *App) ManageWindowLifecycle(window *application.WebviewWindow) {
    // Track window state
    window.OnWindowEvent(events.Common.WindowFocus, func(e *application.WindowEvent) {
        fmt.Println("Window focused:", window.Name())
    })

    window.OnWindowEvent(events.Common.WindowLostFocus, func(e *application.WindowEvent) {
        fmt.Println("Window lost focus:", window.Name())
    })

    // Cleanup on close
    window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
        // Save window state
        a.saveWindowState(window)
        // Remove from tracking map
        delete(a.windows, window.Name())
    })
}
```

## Memory Management

```go
// When a window is closed, clean up references
window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
    // Remove from map
    delete(a.windows, window.Name())
    // Clear reference
    a.mainWindow = nil
})
```

## Advanced Patterns

### Window Pool

Reuse windows for better performance:

```go
type WindowPool struct {
    available []*application.WebviewWindow
    inUse     map[string]*application.WebviewWindow
}

func (p *WindowPool) Acquire(name string) *application.WebviewWindow {
    if len(p.available) > 0 {
        w := p.available[len(p.available)-1]
        p.available = p.available[:len(p.available)-1]
        p.inUse[name] = w
        return w
    }
    // Create new window
    w := app.Window.NewWithOptions(application.WebviewWindowOptions{
        Width:  800,
        Height: 600,
    })
    p.inUse[name] = w
    return w
}

func (p *WindowPool) Release(name string) {
    if w, ok := p.inUse[name]; ok {
        delete(p.inUse, name)
        w.Hide()
        p.available = append(p.available, w)
    }
}
```

### Window Groups

Manage related windows together:

```go
type WindowGroup struct {
    windows []*application.WebviewWindow
}

func (g *WindowGroup) CloseAll() {
    for _, w := range g.windows {
        w.Close()
    }
}

func (g *WindowGroup) ShowAll() {
    for _, w := range g.windows {
        w.Show()
    }
}

func (g *WindowGroup) HideAll() {
    for _, w := range g.windows {
        w.Hide()
    }
}
```

### Workspace Management

Save and restore window positions:

```go
type Workspace struct {
    Windows []WindowState `json:"windows"`
}

type WindowState struct {
    Name      string `json:"name"`
    X         int    `json:"x"`
    Y         int    `json:"y"`
    Width     int    `json:"width"`
    Height    int    `json:"height"`
    Maximized bool   `json:"maximized"`
}

func (a *App) SaveWorkspace() Workspace {
    ws := Workspace{}
    a.windowMap.Range(func(key, value interface{}) bool {
        w := value.(*application.WebviewWindow)
        x, y := w.Position()
        width, height := w.Size()
        ws.Windows = append(ws.Windows, WindowState{
            Name:   key.(string),
            X:      x,
            Y:      y,
            Width:  width,
            Height: height,
        })
        return true
    })
    return ws
}
```

## Complete Example

```go
package main

import (
    "github.com/wailsapp/wails/v3/pkg/application"
)

type App struct {
    windows map[string]*application.WebviewWindow
}

func (a *App) startup() {
    a.windows = make(map[string]*application.WebviewWindow)
}

func (a *App) CreateMainWindow() {
    window := app.Window.NewWithOptions(application.WebviewWindowOptions{
        Title:  "Main Window",
        Width:  1024,
        Height: 768,
        URL:    "http://wails.localhost/",
    })
    a.windows["main"] = window
    window.Show()
}

func (a *App) CreateSecondaryWindow(name string) {
    if _, exists := a.windows[name]; exists {
        a.windows[name].Focus()
        return
    }
    window := app.Window.NewWithOptions(application.WebviewWindowOptions{
        Title:  name,
        Width:  800,
        Height: 600,
        URL:    "http://wails.localhost/secondary",
    })
    a.windows[name] = window
    window.Show()
}

func (a *App) CloseWindow(name string) {
    if w, ok := a.windows[name]; ok {
        w.Close()
        delete(a.windows, name)
    }
}

func main() {
    app := application.New(application.Options{
        Name: "Multi-Window App",
    })

    myApp := &App{}
    app.Lifecycle.OnStartup(myApp.startup)

    // Create main window on startup
    app.Lifecycle.OnShutdown(func() {
        // Cleanup all windows
        for name, w := range myApp.windows {
            w.Close()
            delete(myApp.windows, name)
        }
    })

    app.Run()
}
```
