---
title: "Frontend Runtime"
description: "The Wails JavaScript runtime package for frontend integration"
---

The Wails frontend runtime is the standard library for Wails applications. It provides a number of features including window management, dialogs, browser integration, clipboard, menus, system information, events, context menus, screens, and WML.

The runtime is required for integration between Go and the frontend. There are 2 ways to integrate the runtime:

* Using the `@wailsio/runtime` package
* Using a pre-built bundle

## Using the npm package

The `@wailsio/runtime` package is a JavaScript package that provides access to the Wails runtime from the frontend. It is used by all standard templates and is the recommended way to integrate the runtime.

```
npm install --save @wailsio/runtime
```

## Using a pre-built bundle

Some projects will not use a JavaScript bundler and may prefer to use a pre-built bundled version of the runtime:

```
wails3 generate runtime
```

The command will output a `runtime.js` (and `runtime.debug.js`) file in the current directory.

## Initialisation

Make sure to include a side-effect import statement somewhere in your frontend code:

```js
import "@wailsio/runtime";
```

## Vite Plugin for Typed Events

The runtime includes a Vite plugin that enables HMR support for typed events during development.

```ts
import { defineConfig } from 'vite'
import wails from '@wailsio/runtime/plugins/vite'

export default defineConfig({
  plugins: [wails()],
})
```

## API Reference

The runtime is organized into modules:

```js
import { Events, Window, Clipboard } from '@wailsio/runtime'
```

### Events

* `Events.On(eventName, callback)` - Register a callback for an event, returns unsubscribe function
* `Events.Once(eventName, callback)` - Register a callback that runs only once
* `Events.Emit(name, data?)` - Emit an event to Go backend or other windows
* `Events.Off(...eventNames)` - Remove event listeners
* `Events.OffAll()` - Remove all event listeners

### Window

* `Window.SetTitle(title)` - Sets window title
* `Window.Center()` - Centers the window
* `Window.SetSize(width, height)` - Sets window size
* `Window.Size()` - Gets window size
* `Window.SetPosition(x, y)` - Sets absolute position
* `Window.Position()` - Gets absolute position
* `Window.Show()`, `Window.Hide()`, `Window.Close()` - Visibility
* `Window.Minimise()`, `Window.Maximise()`, `Window.Fullscreen()`, `Window.Restore()` - Window state
* `Window.Focus()`, `Window.IsFocused()` - Focus
* `Window.SetZoom()`, `Window.GetZoom()` - Zoom level
* `Window.Print()` - Opens native print dialog

### Clipboard

* `Clipboard.SetText(text)` - Set clipboard text
* `Clipboard.Text()` - Get clipboard text

### System

* `System.invoke(message)` - Sends a raw message directly to the backend

### Application

* `Application.Show()` - Shows all application windows
* `Application.Hide()` - Hides all application windows
* `Application.Quit()` - Quits the application

### Browser

* `Browser.OpenURL(url)` - Opens a URL in the system browser

### Screens

* `Screens.GetAll()` - Gets all screens
* `Screens.GetPrimary()` - Gets the primary screen
* `Screens.GetCurrent()` - Gets the current active screen

### Dialogs

* `Dialogs.Info(options)` - Shows an information dialog
* `Dialogs.Error(options)` - Shows an error dialog
* `Dialogs.Warning(options)` - Shows a warning dialog
* `Dialogs.Question(options)` - Shows a question dialog
* `Dialogs.OpenFile(options)` - Shows a file open dialog
* `Dialogs.SaveFile(options)` - Shows a file save dialog

### WML (Wails Markup Language)

WML provides declarative attributes for common actions on HTML elements:

* `wml-event` - Emits an event when clicked (`<button wml-event="save-clicked">Save</button>`)
* `wml-window` - Calls a window method (`<button wml-window="Close">Close Window</button>`)
* `wml-target-window` - Specifies target window for wml-window
* `wml-openurl` - Opens a URL in the browser
* `wml-confirm` - Shows confirmation dialog before action
