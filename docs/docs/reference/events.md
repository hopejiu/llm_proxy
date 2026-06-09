---
title: "Events API"
description: "Complete reference for the Events API"
---

## Overview

The Events API provides methods to emit and listen to events, enabling communication between different parts of your application.

**Event Types:**

* **Application Events** - App lifecycle events (startup, shutdown)
* **Window Events** - Window state changes (focus, blur, resize)
* **Custom Events** - User-defined events for app-specific communication

**Communication Patterns:**

* **Go to Frontend** - Emit events from Go, listen in JavaScript
* **Frontend to Go** - Not directly (use service bindings instead)
* **Frontend to Frontend** - Via Go or local runtime events
* **Window to Window** - Target specific windows or broadcast to all

## Event Methods (Go)

### app.Event.Emit()

```go
func (em *EventManager) Emit(name string, data ...any) bool
```

### app.Event.On()

```go
func (em *EventManager) On(name string, callback func(*CustomEvent)) func()
```

Returns a cleanup function to remove the event listener.

### Window-Specific Events

```go
window.EmitEvent("notification", "Hello from Go!")
app.Event.Emit("global-update", data)
```

## Event Methods (Frontend)

```js
import { Events } from '@wailsio/runtime'

Events.On(eventName, callback)          // Listen for events
Events.Once(eventName, callback)        // Listen for a single event occurrence
Events.Off(...eventNames)               // Remove all listeners for given events
Events.OffAll()                         // Remove every event listener
Events.OnMultiple(eventName, callback, max) // Listen up to max times, then auto-unsubscribe
```

## Application Events

### app.Event.OnApplicationEvent()

```go
func (em *EventManager) OnApplicationEvent(
    eventType events.ApplicationEventType,
    callback func(*ApplicationEvent),
) func()
```

**Common application events:**

* `events.Common.ApplicationStarted` - Application has finished launching.
* `events.Common.ThemeChanged` - System theme switched between light and dark.
* `events.Common.ApplicationOpenedWithFile` - Launched via a file association.
* `events.Common.ApplicationLaunchedWithUrl` - Launched via a URL scheme.

## Window Events

### OnWindowEvent()

```go
func (w *WebviewWindow) OnWindowEvent(
    eventType events.WindowEventType,
    callback func(*WindowEvent),
) func()
```

**Common window events:**

* `events.Common.WindowFocus` / `WindowLostFocus` - Window focus changes
* `events.Common.WindowClosing` - Window is about to close (cancellable via RegisterHook)
* `events.Common.WindowDidResize` / `WindowDidMove` - Window geometry changes
* `events.Common.WindowMinimise` / `WindowUnMinimise` / `WindowMaximise` / `WindowUnMaximise` / `WindowFullscreen` / `WindowUnFullscreen`
* `events.Common.WindowRuntimeReady` - Safe to emit events to the in-window runtime
* `events.Common.WindowFilesDropped` / `WindowDropZoneFilesDropped` - File drop events

## Event Naming Conventions

```
// Good - descriptive and specific
app.Event.Emit("user:logged-in", user)
app.Event.Emit("data:fetch:complete", results)

// Bad - vague and unclear
app.Event.Emit("event1", data)
app.Event.Emit("update", stuff)
```

## Performance Considerations

### Debouncing High-Frequency Events

In Go:

```go
type Service struct {
    app            *application.App
    lastEmit       time.Time
    debounceWindow time.Duration
}

func (s *Service) EmitWithDebounce(event string, data interface{}) {
    now := time.Now()
    if now.Sub(s.lastEmit) < s.debounceWindow {
        return
    }
    s.app.Event.Emit(event, data)
    s.lastEmit = now
}
```

In JavaScript:

```js
let lastUpdate = 0
const throttleMs = 100

Events.On('high-frequency-event', (data) => {
    const now = Date.now()
    if (now - lastUpdate < throttleMs) return
    processUpdate(data)
    lastUpdate = now
})
```
