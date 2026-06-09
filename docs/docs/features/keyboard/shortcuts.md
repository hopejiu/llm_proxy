---
title: Keyboard Shortcuts
description: Register global keyboard shortcuts for quick access to functionality
image: ../../../assets/images/og-image.png
---

[Skip to content](#%5Ftop) 

# Keyboard Shortcuts

Wails provides a powerful key binding system that allows you to register global keyboard shortcuts that work across all windows in your application. This enables users to quickly access functionality without navigating through menus.

## Accessing the Key Binding Manager

[Section titled “Accessing the Key Binding Manager”](#accessing-the-key-binding-manager)

The key binding manager is accessed through the `KeyBindings` property on your application instance:

```

app := application.New(application.Options{

    Name: "Keyboard Shortcuts Demo",

})


// Access the key binding manager

keyBindings := app.KeyBinding


```

## Adding Key Bindings

[Section titled “Adding Key Bindings”](#adding-key-bindings)

### Basic Key Binding

[Section titled “Basic Key Binding”](#basic-key-binding)

Register a simple keyboard shortcut:

```

app.KeyBinding.Add("Ctrl+S", func(window application.Window) {

    // Handle save action

    app.Logger.Info("Save shortcut triggered")

    // Perform save operation...

})


```

### Multiple Key Bindings

[Section titled “Multiple Key Bindings”](#multiple-key-bindings)

Register multiple shortcuts for common operations:

```

// File operations

app.KeyBinding.Add("Ctrl+N", func(window application.Window) {

    // New file

    window.EmitEvent("file:new", nil)

})


app.KeyBinding.Add("Ctrl+O", func(window application.Window) {

    // Open file

    dialog := app.Dialog.OpenFile()

    if file, err := dialog.PromptForSingleSelection(); err == nil {

        window.EmitEvent("file:open", file)

    }

})


app.KeyBinding.Add("Ctrl+S", func(window application.Window) {

    // Save file

    window.EmitEvent("file:save", nil)

})


// Edit operations

app.KeyBinding.Add("Ctrl+Z", func(window application.Window) {

    // Undo

    window.EmitEvent("edit:undo", nil)

})


app.KeyBinding.Add("Ctrl+Y", func(window application.Window) {

    // Redo (Windows/Linux)

    window.EmitEvent("edit:redo", nil)

})


app.KeyBinding.Add("Cmd+Shift+Z", func(window application.Window) {

    // Redo (macOS)

    window.EmitEvent("edit:redo", nil)

})


```

## Key Binding Accelerators

[Section titled “Key Binding Accelerators”](#key-binding-accelerators)

### Accelerator Format

[Section titled “Accelerator Format”](#accelerator-format)

Key bindings use a standard accelerator format with modifiers and keys:

```

// Modifier keys

"Ctrl+S"        // Control + S

"Cmd+S"         // Command + S (macOS)

"Alt+F4"        // Alt + F4

"Shift+Ctrl+Z"  // Shift + Control + Z


// Function keys

"F1"            // F1 key

"Ctrl+F5"       // Control + F5


// Special keys

"Escape"        // Escape key

"Enter"         // Enter key

"Space"         // Spacebar

"Tab"           // Tab key

"Backspace"     // Backspace key

"Delete"        // Delete key


// Arrow keys

"Up"            // Up arrow

"Down"          // Down arrow

"Left"          // Left arrow

"Right"         // Right arrow


```

### Platform-Specific Accelerators

[Section titled “Platform-Specific Accelerators”](#platform-specific-accelerators)

Handle platform differences for common shortcuts:

```

import "runtime"


// Cross-platform save shortcut

if runtime.GOOS == "darwin" {

    app.KeyBinding.Add("Cmd+S", saveHandler)

} else {

    app.KeyBinding.Add("Ctrl+S", saveHandler)

}


// Or register both

app.KeyBinding.Add("Ctrl+S", saveHandler)

app.KeyBinding.Add("Cmd+S", saveHandler)


```

## Managing Key Bindings

[Section titled “Managing Key Bindings”](#managing-key-bindings)

### Removing Key Bindings

[Section titled “Removing Key Bindings”](#removing-key-bindings)

Remove key bindings when they’re no longer needed:

```

// Remove a specific key binding

app.KeyBinding.Remove("Ctrl+S")


// Example: Temporary key binding for a modal

app.KeyBinding.Add("Escape", func(window application.Window) {

    // Close modal

    window.EmitEvent("modal:close", nil)

    // Remove this temporary binding

    app.KeyBinding.Remove("Escape")

})


```

### Getting All Key Bindings

[Section titled “Getting All Key Bindings”](#getting-all-key-bindings)

Retrieve all registered key bindings:

```

allBindings := app.KeyBinding.GetAll()

for _, binding := range allBindings {

    app.Logger.Info("Key binding", "accelerator", binding.Accelerator)

}


```

## Advanced Usage

[Section titled “Advanced Usage”](#advanced-usage)

### Context-Aware Key Bindings

[Section titled “Context-Aware Key Bindings”](#context-aware-key-bindings)

Make key bindings context-aware by checking application state:

```

app.KeyBinding.Add("Ctrl+S", func(window application.Window) {

    // Check current application state

    if isEditMode() {

        // Save document

        saveDocument()

    } else if isInSettings() {

        // Save settings

        saveSettings()

    } else {

        app.Logger.Info("Save not available in current context")

    }

})


```

### Window-Specific Actions

[Section titled “Window-Specific Actions”](#window-specific-actions)

Key bindings receive the active window, allowing window-specific behavior:

```

app.KeyBinding.Add("F11", func(window application.Window) {

    // Toggle fullscreen for the active window

    window.ToggleFullscreen()

})


app.KeyBinding.Add("Ctrl+W", func(window application.Window) {

    // Close the active window

    window.Close()

})


```

### Dynamic Key Binding Management

[Section titled “Dynamic Key Binding Management”](#dynamic-key-binding-management)

Dynamically add and remove key bindings based on application state:

```

func enableEditMode() {

    // Add edit-specific key bindings

    app.KeyBinding.Add("Ctrl+B", func(window application.Window) {

        window.EmitEvent("format:bold", nil)

    })


    app.KeyBinding.Add("Ctrl+I", func(window application.Window) {

        window.EmitEvent("format:italic", nil)

    })


    app.KeyBinding.Add("Ctrl+U", func(window application.Window) {

        window.EmitEvent("format:underline", nil)

    })

}


func disableEditMode() {

    // Remove edit-specific key bindings

    app.KeyBinding.Remove("Ctrl+B")

    app.KeyBinding.Remove("Ctrl+I")

    app.KeyBinding.Remove("Ctrl+U")

}


```

## Platform Considerations

[Section titled “Platform Considerations”](#platform-considerations)

* [  macOS ](#tab-panel-74)
* [  Windows ](#tab-panel-75)
* [  Linux ](#tab-panel-76)

On macOS:

* Use `Cmd` instead of `Ctrl` for standard shortcuts
* `Cmd+Q` is typically reserved for quitting the application
* `Cmd+H` hides the application
* `Cmd+M` minimizes windows
* Consider standard macOS keyboard shortcuts

Common macOS patterns:

```

app.KeyBinding.Add("Cmd+N", newFileHandler)      // New

app.KeyBinding.Add("Cmd+O", openFileHandler)     // Open

app.KeyBinding.Add("Cmd+S", saveFileHandler)     // Save

app.KeyBinding.Add("Cmd+Z", undoHandler)         // Undo

app.KeyBinding.Add("Cmd+Shift+Z", redoHandler)   // Redo

app.KeyBinding.Add("Cmd+C", copyHandler)         // Copy

app.KeyBinding.Add("Cmd+V", pasteHandler)        // Paste


```

On Windows:

* Use `Ctrl` for standard shortcuts
* `Alt+F4` closes applications
* `F1` typically opens help
* Consider Windows keyboard conventions

Common Windows patterns:

```

app.KeyBinding.Add("Ctrl+N", newFileHandler)     // New

app.KeyBinding.Add("Ctrl+O", openFileHandler)    // Open

app.KeyBinding.Add("Ctrl+S", saveFileHandler)    // Save

app.KeyBinding.Add("Ctrl+Z", undoHandler)        // Undo

app.KeyBinding.Add("Ctrl+Y", redoHandler)        // Redo

app.KeyBinding.Add("Ctrl+C", copyHandler)        // Copy

app.KeyBinding.Add("Ctrl+V", pasteHandler)       // Paste

app.KeyBinding.Add("F1", helpHandler)            // Help


```

On Linux:

* Generally follows Windows conventions with `Ctrl`
* May vary by desktop environment
* Consider GNOME/KDE standard shortcuts
* Some desktop environments reserve certain shortcuts

Common Linux patterns:

```

app.KeyBinding.Add("Ctrl+N", newFileHandler)     // New

app.KeyBinding.Add("Ctrl+O", openFileHandler)    // Open

app.KeyBinding.Add("Ctrl+S", saveFileHandler)    // Save

app.KeyBinding.Add("Ctrl+Z", undoHandler)        // Undo

app.KeyBinding.Add("Ctrl+Shift+Z", redoHandler)  // Redo

app.KeyBinding.Add("Ctrl+C", copyHandler)        // Copy

app.KeyBinding.Add("Ctrl+V", pasteHandler)       // Paste


```

## Best Practices

[Section titled “Best Practices”](#best-practices)

1. **Use Standard Shortcuts**: Follow platform conventions for common operations:  
```  
// Cross-platform save  
if runtime.GOOS == "darwin" {  
    app.KeyBinding.Add("Cmd+S", saveHandler)  
} else {  
    app.KeyBinding.Add("Ctrl+S", saveHandler)  
}  
```
2. **Provide Visual Feedback**: Let users know when shortcuts are triggered:  
```  
app.KeyBinding.Add("Ctrl+S", func(window application.Window) {  
    saveDocument()  
    // Show brief notification  
    window.EmitEvent("notification:show", "Document saved")  
})  
```
3. **Handle Conflicts**: Be careful not to override important system shortcuts:  
```  
// Avoid overriding system shortcuts like:  
// Ctrl+Alt+Del (Windows)  
// Cmd+Space (macOS Spotlight)  
// Alt+Tab (Window switching)  
```
4. **Document Shortcuts**: Provide help or documentation for available shortcuts:  
```  
app.KeyBinding.Add("F1", func(window application.Window) {  
    // Show help dialog with available shortcuts  
    showKeyboardShortcutsHelp()  
})  
```
5. **Clean Up**: Remove temporary key bindings when they’re no longer needed:  
```  
func enterEditMode() {  
    app.KeyBinding.Add("Escape", exitEditModeHandler)  
}  
func exitEditModeHandler(window application.Window) {  
    exitEditMode()  
    app.KeyBinding.Remove("Escape") // Clean up temporary binding  
}  
```

## Complete Example

[Section titled “Complete Example”](#complete-example)

Here’s a complete example of a text editor with keyboard shortcuts:

```

package main


import (

    "runtime"

    "github.com/wailsapp/wails/v3/pkg/application"

)


func main() {

    app := application.New(application.Options{

        Name: "Text Editor with Shortcuts",

    })


    // File operations

    if runtime.GOOS == "darwin" {

        app.KeyBinding.Add("Cmd+N", func(window application.Window) {

            window.EmitEvent("file:new", nil)

        })

        app.KeyBinding.Add("Cmd+O", func(window application.Window) {

            openFile(app, window)

        })

        app.KeyBinding.Add("Cmd+S", func(window application.Window) {

            window.EmitEvent("file:save", nil)

        })

    } else {

        app.KeyBinding.Add("Ctrl+N", func(window application.Window) {

            window.EmitEvent("file:new", nil)

        })

        app.KeyBinding.Add("Ctrl+O", func(window application.Window) {

            openFile(app, window)

        })

        app.KeyBinding.Add("Ctrl+S", func(window application.Window) {

            window.EmitEvent("file:save", nil)

        })

    }


    // View operations

    app.KeyBinding.Add("F11", func(window application.Window) {

        window.ToggleFullscreen()

    })


    app.KeyBinding.Add("F1", func(window application.Window) {

        showKeyboardShortcuts(window)

    })


    // Create main window

    window := app.Window.New()

    window.SetTitle("Text Editor")


    err := app.Run()

    if err != nil {

        panic(err)

    }

}


func openFile(app *application.App, window application.Window) {

    dialog := app.Dialog.OpenFile()

    dialog.AddFilter("Text Files", "*.txt;*.md")


    if file, err := dialog.PromptForSingleSelection(); err == nil {

        window.EmitEvent("file:open", file)

    }

}


func showKeyboardShortcuts(window application.Window) {

    shortcuts := `

Keyboard Shortcuts:

- Ctrl/Cmd+N: New file

- Ctrl/Cmd+O: Open file

- Ctrl/Cmd+S: Save file

- F11: Toggle fullscreen

- F1: Show this help

`

    window.EmitEvent("help:show", shortcuts)

}


```

Pro Tip

Test your key bindings on all target platforms to ensure they work correctly and don’t conflict with system shortcuts.

Warning

Be careful not to override critical system shortcuts. Some key combinations are reserved by the operating system and cannot be captured by applications.

```json
{"@context":"https://schema.org","@type":"SoftwareApplication","name":"Wails","description":"Build beautiful desktop applications using Go and modern web technologies.","url":"https://v3.wails.io","applicationCategory":"DeveloperApplication","downloadUrl":"https://github.com/wailsapp/wails/releases","softwareVersion":"3.0","operatingSystem":["Windows","macOS","Linux"],"offers":{"@type":"Offer","price":"0","priceCurrency":"USD"},"author":{"@type":"Organization","name":"Wails Contributors","url":"https://github.com/wailsapp/wails"},"provider":{"@type":"Organization","name":"Wails Contributors","url":"https://github.com/wailsapp/wails"}}
{"@context":"https://schema.org","@type":"Organization","name":"Wails","url":"https://v3.wails.io","logo":"../../../assets/images/favicon.svg","sameAs":["https://github.com/wailsapp/wails","https://x.com/wailsapp","https://discord.gg/JDdSxwjhGf"]}
```
