---
title: "Manager API"
description: "Organized API structure with focused manager interfaces"
---

The Wails v3 Manager API provides an organised and discoverable way to access application functionality through focused manager structs grouped under public fields on `*application.App`.

## Overview

The Manager API organises application functionality into twelve focused areas:

* **`app.Window`** - Window creation, management, and callbacks
* **`app.ContextMenu`** - Context menu registration and management
* **`app.KeyBinding`** - Global key binding management
* **`app.Browser`** - Browser integration (opening URLs and files)
* **`app.Env`** - Environment information and system state
* **`app.Dialog`** - File and message dialog operations
* **`app.Event`** - Custom event handling and application events
* **`app.Menu`** - Application menu management
* **`app.Screen`** - Screen management and coordinate transformations
* **`app.Clipboard`** - Clipboard text operations
* **`app.SystemTray`** - System tray icon creation and management
* **`app.Autostart`** - Register the application to launch at user login

## Benefits

* **Better discoverability** - IDE autocomplete shows organized API surface
* **Improved code organization** - Related methods grouped together
* **Enhanced maintainability** - Separation of concerns across managers
* **Future extensibility** - Easier to add new features to specific areas

## Usage

```go
// Events and custom event handling
app.Event.Emit("custom", data)
app.Event.On("custom", func(e *CustomEvent) { ... })

// Window management
window, _ := app.Window.GetByName("main")
app.Window.OnCreate(func(window Window) { ... })

// Browser integration
app.Browser.OpenURL("https://wails.io")

// Menu management
menu := app.Menu.New()
app.Menu.Set(menu)

// System tray
systray := app.SystemTray.New()
```

## Manager Reference

### Window Manager

```go
// Create windows
window := app.Window.New()
window := app.Window.NewWithOptions(options)
current := app.Window.Current()

// Find windows
window, exists := app.Window.GetByName("main")
windows := app.Window.GetAll()

// Window callbacks
app.Window.OnCreate(func(window Window) {
    // Handle window creation
})
```

### Event Manager

```go
// Custom events
app.Event.Emit("userAction", data)
cancelFunc := app.Event.On("userAction", func(e *CustomEvent) { ... })
app.Event.Off("userAction")
app.Event.Reset()

// Application events
app.Event.OnApplicationEvent(events.Common.ThemeChanged, func(e *ApplicationEvent) { ... })
```

### Browser Manager

```go
err := app.Browser.OpenURL("https://wails.io")
err := app.Browser.OpenFile("/path/to/document.pdf")
```

### Environment Manager

```go
env := app.Env.Info()
if app.Env.IsDarkMode() { ... }
err := app.Env.OpenFileManager("/path/to/folder", false)
```

### Dialog Manager

```go
result, err := app.Dialog.OpenFile().AddFilter("Text Files", "*.txt").PromptForSingleSelection()
result, err = app.Dialog.SaveFile().SetFilename("document.txt").PromptForSingleSelection()
app.Dialog.Info().SetTitle("Information").SetMessage("Operation completed successfully").Show()
```

### Menu Manager

```go
menu := app.Menu.New()
fileMenu := menu.AddSubmenu("File")
fileMenu.Add("New").OnClick(func(ctx *Context) { ... })
app.Menu.Set(menu)
```

### Key Binding Manager

```go
app.KeyBinding.Add("ctrl+n", func(window application.Window) { ... })
app.KeyBinding.Add("ctrl+q", func(window application.Window) { app.Quit() })
app.KeyBinding.Remove("ctrl+n")
bindings := app.KeyBinding.GetAll()
```

### Context Menu Manager

```go
menu := app.ContextMenu.New()
app.ContextMenu.Add("myMenu", menu)
menu, exists := app.ContextMenu.Get("myMenu")
app.ContextMenu.Remove("myMenu")
```

### Screen Manager

```go
screens := app.Screen.GetAll()
primary := app.Screen.GetPrimary()
physicalPoint := app.Screen.DipToPhysicalPoint(logicalPoint)
```

### Clipboard Manager

```go
success := app.Clipboard.SetText("Hello World")
text, ok := app.Clipboard.Text()
```

### SystemTray Manager

```go
systray := app.SystemTray.New()
systray.SetLabel("My App")
systray.SetIcon(iconBytes)
systray.Destroy()
```

### Autostart Manager

```go
err := app.Autostart.Enable()
err = app.Autostart.EnableWithOptions(application.AutostartOptions{
    Identifier: "com.example.myapp",
    Arguments:  []string{"--hidden"},
})
enabled, err := app.Autostart.IsEnabled()
err = app.Autostart.Disable()
```
