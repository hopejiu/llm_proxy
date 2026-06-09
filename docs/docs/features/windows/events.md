---
title: Window Events
description: Handle window lifecycle, state-change, and position events in Wails v3
---

# Window Events

Wails dispatches window lifecycle and state-change events through a single API: `OnWindowEvent` for passive listeners, and `RegisterHook` for cancellable hooks that can prevent the default action.

## Listening to Events

```go
import "github.com/wailsapp/wails/v3/pkg/events"

// Listener — observes the event, cannot cancel it.
window.OnWindowEvent(events.Common.WindowDidResize, func(e *application.WindowEvent) {
    // handle resize
})

// Hook — runs before listeners; can call e.Cancel() to suppress the default action.
window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
    if hasUnsavedChanges() {
        e.Cancel() // prevent the window from closing
    }
})
```

Cross-platform events live in `events.Common.*`. Platform-specific events live in `events.Mac.*`, `events.Windows.*`, and `events.Linux.*`. The full list is generated in `v3/pkg/events/events.go`.

## Window Creation Event

Run a callback every time a window is created using `app.Window.OnCreate`:

```go
app.Window.OnCreate(func(window application.Window) {
    fmt.Printf("Window created: %s (ID: %d)\n", window.Name(), window.ID())
    // Configure all new windows
    window.SetMinSize(400, 300)
    // Register a hook for this window
    window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
        if !confirmClose() {
            e.Cancel()
        }
    })
})
```

## Lifecycle Events

### WindowClosing

- Use a hook (`RegisterHook`) — only hooks can cancel the close. Listeners observe the event but cannot prevent it.

```go
window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
    if hasUnsavedChanges() {
        e.Cancel() // prevent the window from closing
    }
})
```

- `WindowClosing` is dispatched on user-initiated close attempts.
- A `RegisterHook` callback can call `e.Cancel()` to keep the window open.
- `OnWindowEvent` callbacks for `WindowClosing` are observers — they fire but cannot cancel.

Programmatic close: `window.Close()`. There is no `window.Destroy()`.

### WindowRuntimeReady

Fired when the in-window runtime is initialised and ready for API calls.

### WindowCreation

Fired when a new window is created.

## Focus Events

### WindowFocus

Fired when a window gains focus.

```go
window.OnWindowEvent(events.Common.WindowFocus, func(e *application.WindowEvent) {
    window.SetTitle("My Application (Active)")
})
```

### WindowLostFocus

Fired when a window loses focus.

```go
window.OnWindowEvent(events.Common.WindowLostFocus, func(e *application.WindowEvent) {
    window.SetTitle("My Application")
})
```

## State Change Events

### WindowMinimise / WindowUnMinimise

Fired when a window is minimised or restored from minimised state.

### WindowMaximise / WindowUnMaximise

Fired when a window is maximised or restored from maximised state.

### WindowFullscreen / WindowUnFullscreen

Fired when a window enters or exits fullscreen.

## Position and Size Events

### WindowDidResize

Fired when a window is resized. Note: `WindowDidResize` and `WindowDidMove` do not carry coordinates on the event itself — query the window via `window.Size()` / `window.Position()` inside the handler.

### WindowDidMove

Fired when a window is moved. Query the new position using `window.Position()`.

## Complete Example

```go
package main

import (
    "encoding/json"
    "fmt"
    "os"

    "github.com/wailsapp/wails/v3/pkg/application"
    "github.com/wailsapp/wails/v3/pkg/events"
)

type WindowState struct {
    X          int  `json:"x"`
    Y          int  `json:"y"`
    Width      int  `json:"width"`
    Height     int  `json:"height"`
    Maximised  bool `json:"maximised"`
    Fullscreen bool `json:"fullscreen"`
}

type ManagedWindow struct {
    app    *application.App
    window *application.WebviewWindow
    state  WindowState
    dirty  bool
}

func main() {
    app := application.New(application.Options{
        Name: "Event Demo",
    })
    mw := &ManagedWindow{app: app}
    mw.CreateWindow()
    mw.LoadState()
    mw.SetupEventHandlers()
    app.Run()
}

func (mw *ManagedWindow) CreateWindow() {
    mw.window = mw.app.Window.NewWithOptions(application.WebviewWindowOptions{
        Name:   "main",
        Title:  "Event Demo",
        Width:  800,
        Height: 600,
    })
}

func (mw *ManagedWindow) SetupEventHandlers() {
    // Focus events
    mw.window.OnWindowEvent(events.Common.WindowFocus, func(e *application.WindowEvent) {
        mw.window.EmitEvent("focus-state", true)
    })
    mw.window.OnWindowEvent(events.Common.WindowLostFocus, func(e *application.WindowEvent) {
        mw.window.EmitEvent("focus-state", false)
    })

    // State change events
    mw.window.OnWindowEvent(events.Common.WindowMaximise, func(e *application.WindowEvent) {
        mw.state.Maximised = true
        mw.dirty = true
    })
    mw.window.OnWindowEvent(events.Common.WindowUnMaximise, func(e *application.WindowEvent) {
        mw.state.Maximised = false
        mw.dirty = true
    })
    mw.window.OnWindowEvent(events.Common.WindowFullscreen, func(e *application.WindowEvent) {
        mw.state.Fullscreen = true
        mw.dirty = true
    })
    mw.window.OnWindowEvent(events.Common.WindowUnFullscreen, func(e *application.WindowEvent) {
        mw.state.Fullscreen = false
        mw.dirty = true
    })

    // Position and size events
    mw.window.OnWindowEvent(events.Common.WindowDidMove, func(e *application.WindowEvent) {
        mw.state.X, mw.state.Y = mw.window.Position()
        mw.dirty = true
    })
    mw.window.OnWindowEvent(events.Common.WindowDidResize, func(e *application.WindowEvent) {
        mw.state.Width, mw.state.Height = mw.window.Size()
        mw.dirty = true
    })

    // Cancellable close — use a hook, not a listener.
    mw.window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
        if mw.dirty {
            mw.SaveState()
        }
    })
}

func (mw *ManagedWindow) LoadState() {
    data, err := os.ReadFile("window-state.json")
    if err != nil {
        return
    }
    if err := json.Unmarshal(data, &mw.state); err != nil {
        return
    }
    // Restore window state
    mw.window.SetPosition(mw.state.X, mw.state.Y)
    mw.window.SetSize(mw.state.Width, mw.state.Height)
    if mw.state.Maximised {
        mw.window.Maximise()
    }
    if mw.state.Fullscreen {
        mw.window.Fullscreen()
    }
}

func (mw *ManagedWindow) SaveState() {
    data, err := json.Marshal(mw.state)
    if err != nil {
        return
    }
    os.WriteFile("window-state.json", data, 0644)
    mw.dirty = false
    fmt.Println("Window state saved")
}
```

## Event Coordination

### Cross-Window Communication

Coordinate between multiple windows using the application event bus:

```go
// In main window
mainWindow.OnWindowEvent(events.Common.WindowFocus, func(e *application.WindowEvent) {
    app.Event.Emit("main-window-focused", nil)
})

// In other windows
app.Event.On("main-window-focused", func(event *application.CustomEvent) {
    updateRelativeToMain()
})
```

## Best Practices

- **Use hooks for cancellation** — only `RegisterHook` callbacks can call `e.Cancel()`.
- **Save state on close** — restore window position/size next launch.
- **Debounce frequent events** — `WindowDidResize` and `WindowDidMove` fire rapidly.
- **Handle focus changes** — update UI appropriately.
- **Coordinate via events** — `app.Event.Emit` for cross-window messaging.
- **Unsubscribe when handlers are no longer needed** — `OnWindowEvent` / `RegisterHook` both return an unsubscribe `func()`.
- **Don't block event handlers** — keep them fast.
- **Don't try to cancel from `OnWindowEvent`** — use `RegisterHook`.
- **Don't use `window.Destroy()`** — it doesn't exist; use `window.Close()`.
- **Don't save on every event** — debounce position/size saves.
- **Don't assume coordinates on the event** — call `window.Position()` / `window.Size()`.
