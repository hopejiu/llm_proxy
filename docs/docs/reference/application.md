---
title: "Application API"
description: "Core APIs for creating and managing Wails applications."
---

## Creating an Application

```go
import "github.com/wailsapp/wails/v3/pkg/application"

app := application.New(application.Options{})
```

`application.Options` accepts configuration for logging, assets, build
settings, and more.

## Core Methods

### Run

Starts the application event loop. This call blocks until the application
quits.

```go
app.Run()
```

### Quit

Gracefully shuts down the application.

```go
app.Quit()
```

### Config

Returns the application configuration.

```go
cfg := app.Config()
```

## Window Management

### New

Create a new window with default options:

```go
window := app.New("My Window")
```

### NewWithOptions

Create a window with explicit options:

```go
window := app.NewWithOptions("My Window", application.WindowOptions{
    Width:  800,
    Height: 600,
    Title:  "My Application",
})
```

### GetByName

Retrieve a window by its registered name:

```go
window := app.GetByName("main")
```

### GetAll

Returns all open windows:

```go
windows := app.GetAll()
```

## Managers

Wails provides manager objects for subsystems. Access them through the
application instance:

| Manager      | Access              | Purpose                     |
|--------------|----------------------|-----------------------------|
| Window       | `app.Window`        | Window lifecycle             |
| Menu         | `app.Menu`          | Application menus            |
| Dialog       | `app.Dialog`        | Native dialogs               |
| Event        | `app.Event`         | Event bus                    |
| Clipboard    | `app.Clipboard`     | Clipboard access             |
| Screen       | `app.Screen`        | Display information          |
| SystemTray   | `app.SystemTray`    | System tray icon             |
| Browser      | `app.Browser`       | Open URLs in default browser |
| Env          | `app.Env`           | Environment variables        |
| ContextMenu  | `app.ContextMenu`   | Context menus                |
| KeyBinding   | `app.KeyBinding`    | Global key bindings          |
| Logger       | `app.Logger`        | Structured logging           |

## Service Management

Register services that receive lifecycle events:

```go
app.RegisterService(&MyService{})
```

Services implement optional interfaces (`OnStartup`, `OnShutdown`, etc.) that
Wails calls at the appropriate lifecycle points.

## Event Management

### Emit

Send an event to the frontend:

```go
app.Event.Emit("myEvent", payload)
```

### On

Listen for events from the frontend:

```go
app.Event.On("frontendEvent", func(data ...interface{}) {
    // handle event
})
```

### OnApplicationEvent

Listen for application-level lifecycle events:

```go
app.Event.OnApplicationEvent(application.EventShutdown, func(data ...interface{}) {
    // clean up
})
```

## Dialog Methods

Open native dialogs:

```go
// Open file dialog
app.Dialog.OpenFile("Select a file")

// Save file dialog
app.Dialog.SaveFile("Save as")

// Message dialog
app.Dialog.Message("Hello, world!").Title("Info").Info()

// Question dialog
app.Dialog.Question("Continue?").Title("Confirm").YesNo()
```

## Logger

Write structured log entries:

```go
app.Logger.Info("Application started")
app.Logger.Error("Something went wrong", "error", err)
```

## Raw Message Handling

Process raw messages from the frontend:

```go
window.On("rawMessage", func(message string) {
    // handle raw JSON or text
})
```

## Platform-Specific Options

Use platform build tags to conditionally configure options:

```go
//go:build darwin

app := application.New(application.Options{
    Mac: application.MacOptions{
        TitleBar: application.MacTitleBar(),
    },
})
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

    window := app.NewWithOptions("main", application.WindowOptions{
        Width:  1024,
        Height: 768,
        Title:  "My Application",
    })

    app.Event.On("greet", func(data ...interface{}) {
        window.Emit("greetResponse", "Hello from Go!")
    })

    app.Run()
}
```
