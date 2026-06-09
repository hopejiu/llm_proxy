---
title: Menu Reference
description: Complete reference for menu item types, properties, and methods
---

## Menu Reference

Complete reference for menu item types, properties, and dynamic behaviour. Build professional, responsive menus with checkboxes, radio groups, separators, and dynamic updates.

## Menu Item Types

### Regular Menu Items

The most common type—displays text and triggers an action:

```
menuItem := menu.Add("Click Me")
menuItem.OnClick(func(ctx *application.Context) {
    fmt.Println("Menu item clicked!")
})
```

**Use for:** Commands, actions, opening windows

### Checkboxes

Toggle-able menu items with checked/unchecked state:

```
checkbox := menu.AddCheckbox("Enable Feature", true)  // true = initially checked
checkbox.OnClick(func(ctx *application.Context) {
    isChecked := ctx.ClickedMenuItem().Checked()
    fmt.Printf("Feature is now: %v\n", isChecked)
})
```

**Use for:** Boolean settings, feature toggles, view options

**Important:** The checked state toggles automatically when clicked.

### Radio Groups

Mutually exclusive options—only one can be selected:

```
menu.AddRadio("Small", true)   // true = initially selected
menu.AddRadio("Medium", false)
menu.AddRadio("Large", false)
```

**Use for:** Mutually exclusive choices (size, theme, mode)

**How grouping works:**

* Adjacent radio items form a group automatically
* Selecting one deselects others in the group
* Separate groups with a separator or regular item

**Example with multiple groups:**

```
// Group 1: Size
menu.AddRadio("Small", true)
menu.AddRadio("Medium", false)
menu.AddRadio("Large", false)

menu.AddSeparator()

// Group 2: Theme
menu.AddRadio("Light", true)
menu.AddRadio("Dark", false)
```

### Submenus

Nested menu structures for organisation:

```
submenu := menu.AddSubmenu("More Options")
submenu.Add("Submenu Item 1").OnClick(func(ctx *application.Context) {
    // Handle click
})
submenu.Add("Submenu Item 2")
```

**Use for:** Grouping related items, reducing clutter

**Nesting limit:** Most platforms support 2-3 levels. Avoid deeper nesting.

### Separators

Visual dividers between menu items:

```
menu.Add("Item 1")
menu.AddSeparator()
menu.Add("Item 2")
```

**Use for:** Grouping related items visually

**Best practice:** Don't start or end menus with separators.

## Menu Item Properties

### Label

The text displayed for the menu item:

```
menuItem := menu.Add("Initial Label")
menuItem.SetLabel("New Label")

// Get current label
label := menuItem.Label()
```

**Dynamic labels:**

```
updateMenuItem := menu.Add("Check for Updates")
updateMenuItem.OnClick(func(ctx *application.Context) {
    updateMenuItem.SetLabel("Checking...")
    menu.Update()  // Important on Windows!

    // Perform update check
    checkForUpdates()

    updateMenuItem.SetLabel("Check for Updates")
    menu.Update()
})
```

### Enabled State

Control whether the menu item can be interacted with:

```
menuItem := menu.Add("Save")
menuItem.SetEnabled(false)  // Greyed out, can't click

// Enable it later
menuItem.SetEnabled(true)
menu.Update()  // Important: Call this after changing enabled state!

// Check current state
isEnabled := menuItem.Enabled()
```

**Windows Menu Behaviour**

On Windows, menus need to be reconstructed when their state changes. **Always call `menu.Update()` after enabling/disabling menu items**, especially if the item was created whilst disabled.

**Why:** Windows menus are rebuilt from scratch when updated. If you don't call `Update()`, click handlers won't fire properly.

**Example: Dynamic enable/disable**

```
var hasSelection bool

cutMenuItem := menu.Add("Cut")
cutMenuItem.SetEnabled(false)  // Initially disabled

copyMenuItem := menu.Add("Copy")
copyMenuItem.SetEnabled(false)

// When selection changes
func onSelectionChanged(selected bool) {
    hasSelection = selected
    cutMenuItem.SetEnabled(hasSelection)
    copyMenuItem.SetEnabled(hasSelection)
    menu.Update()  // Critical on Windows!
}
```

**Common pattern: Enable on condition**

```
saveMenuItem := menu.Add("Save")

func updateSaveMenuItem() {
    canSave := hasUnsavedChanges() && !isSaving()
    saveMenuItem.SetEnabled(canSave)
    menu.Update()
}

// Call whenever state changes
onDocumentChanged(func() {
    updateSaveMenuItem()
})
```

### Checked State

For checkbox and radio items, control or query their checked state:

```
checkbox := menu.AddCheckbox("Feature", false)
checkbox.SetChecked(true)
menu.Update()

// Query state
isChecked := checkbox.Checked()
```

**Auto-toggle:** Checkboxes toggle automatically when clicked. You don't need to call `SetChecked()` in the click handler.

**Manual control:**

```
checkbox := menu.AddCheckbox("Auto-save", false)

// Sync with external state
func syncAutoSave(enabled bool) {
    checkbox.SetChecked(enabled)
    menu.Update()
}
```

### Accelerators (Keyboard Shortcuts)

Add keyboard shortcuts to menu items:

```
saveMenuItem := menu.Add("Save")
saveMenuItem.SetAccelerator("CmdOrCtrl+S")

quitMenuItem := menu.Add("Quit")
quitMenuItem.SetAccelerator("CmdOrCtrl+Q")
```

**Accelerator format:**

* `CmdOrCtrl` - Cmd on macOS, Ctrl on Windows/Linux
* `Shift`, `Alt`, `Option` - Modifier keys
* `A-Z`, `0-9` - Letter/number keys
* `F1-F12` - Function keys
* `Enter`, `Space`, `Backspace`, etc. - Special keys

**Examples:**

```
"CmdOrCtrl+S"           // Save
"CmdOrCtrl+Shift+S"     // Save As
"CmdOrCtrl+W"           // Close Window
"CmdOrCtrl+Q"           // Quit
"F5"                    // Refresh
"CmdOrCtrl+,"           // Preferences (macOS convention)
"Alt+F4"                // Close (Windows convention)
```

**Platform-specific accelerators:**

```
if runtime.GOOS == "darwin" {
    prefsMenuItem.SetAccelerator("Cmd+,")
} else {
    prefsMenuItem.SetAccelerator("Ctrl+P")
}
```

### Tooltip

Add hover text to menu items (platform support varies):

```
menuItem := menu.Add("Advanced Options")
menuItem.SetTooltip("Configure advanced settings")
```

**Platform support:**

* **Windows:** ✅ Supported
* **macOS:** ❌ Not supported (tooltips not standard for menus)
* **Linux:** ⚠️ Varies by desktop environment

### Hidden State

Hide menu items without removing them:

```
debugMenuItem := menu.Add("Debug Mode")
debugMenuItem.SetHidden(true)  // Hidden

// Show in debug builds
if isDebugBuild {
    debugMenuItem.SetHidden(false)
    menu.Update()
}
```

**Use for:** Debug options, feature flags, conditional features

## Event Handling

### OnClick Handler

Execute code when menu item is clicked:

```
menuItem := menu.Add("Click Me")
menuItem.OnClick(func(ctx *application.Context) {
    // Handle click
    fmt.Println("Clicked!")
})
```

**Context provides:**

* `ctx.ClickedMenuItem()` - The menu item that was clicked
* Window context (if from window menu)
* Application context

**Example: Access menu item in handler**

```
checkbox := menu.AddCheckbox("Feature", false)
checkbox.OnClick(func(ctx *application.Context) {
    item := ctx.ClickedMenuItem()
    isChecked := item.Checked()
    fmt.Printf("Feature is now: %v\n", isChecked)
})
```

### Multiple Handlers

You can set multiple handlers (last one wins):

```
menuItem := menu.Add("Action")
menuItem.OnClick(func(ctx *application.Context) {
    fmt.Println("First handler")
})

// This replaces the first handler
menuItem.OnClick(func(ctx *application.Context) {
    fmt.Println("Second handler - this one runs")
})
```

**Best practice:** Set handler once, use conditional logic inside if needed.

## Dynamic Menus

### Updating Menu Items

**The golden rule:** Always call `menu.Update()` after changing menu state.

```
// ✅ Correct
menuItem.SetEnabled(true)
menu.Update()

// ❌ Wrong (especially on Windows)
menuItem.SetEnabled(true)
// Forgot to call Update() - click handlers may not work!
```

**Why this matters:**

* **Windows:** Menus are reconstructed when updated
* **macOS/Linux:** Less critical but still recommended
* **Click handlers:** Won't fire properly without Update()

### Rebuilding Menus

For major changes, rebuild the entire menu:

```
func rebuildFileMenu() {
    menu := app.Menu.New()

    menu.Add("New").OnClick(handleNew)
    menu.Add("Open").OnClick(handleOpen)

    if hasRecentFiles() {
        recentMenu := menu.AddSubmenu("Open Recent")
        for _, file := range getRecentFiles() {
            recentMenu.Add(file).OnClick(func(ctx *application.Context) {
                openFile(file)
            })
        }
    }

    menu.AddSeparator()
    menu.Add("Quit").OnClick(handleQuit)

    // Set the new menu
    window.SetMenu(menu)
}
```

**When to rebuild:**

* Recent files list changes
* Plugin menus change
* Major state transitions

**When to update:**

* Enable/disable items
* Change labels
* Toggle checkboxes

### Context-Sensitive Menus

Adjust menus based on application state:

```
func updateEditMenu() {
    cutMenuItem.SetEnabled(hasSelection())
    copyMenuItem.SetEnabled(hasSelection())
    pasteMenuItem.SetEnabled(hasClipboardContent())
    undoMenuItem.SetEnabled(canUndo())
    redoMenuItem.SetEnabled(canRedo())
    menu.Update()
}

// Call whenever state changes
onSelectionChanged(updateEditMenu)
onClipboardChanged(updateEditMenu)
onUndoStackChanged(updateEditMenu)
```

## Platform Differences

### Menu Bar Location

| Platform    | Location      | Notes                |
| ----------- | ------------- | -------------------- |
| **macOS**   | Top of screen | Global menu bar      |
| **Windows** | Top of window | Per-window menu      |
| **Linux**   | Top of window | Per-window (usually) |

### Standard Menus

**macOS:**

* Has "Application" menu (with app name)
* "Preferences" in Application menu
* "Quit" in Application menu

**Windows/Linux:**

* No Application menu
* "Preferences" in Edit or Tools menu
* "Exit" in File menu

**Example: Platform-appropriate structure**

```
menu := app.Menu.New()

// macOS gets Application menu
if runtime.GOOS == "darwin" {
    menu.AddRole(application.AppMenu)
}

// File menu
fileMenu := menu.AddSubmenu("File")
fileMenu.Add("New")
fileMenu.Add("Open")

// Preferences location varies
if runtime.GOOS == "darwin" {
    // On macOS, preferences are in Application menu (added by AppMenu role)
} else {
    // On Windows/Linux, add to Edit or Tools menu
    editMenu := menu.AddSubmenu("Edit")
    editMenu.Add("Preferences")
}
```

### Accelerator Conventions

**macOS:**

* `Cmd+` for most shortcuts
* `Cmd+,` for Preferences
* `Cmd+Q` for Quit

**Windows:**

* `Ctrl+` for most shortcuts
* `Ctrl+P` or `Ctrl+,` for Preferences
* `Alt+F4` for Exit (or `Ctrl+Q`)

**Linux:**

* Generally follows Windows conventions
* Desktop environment may override

## Best Practices

### ✅ Do

* **Call menu.Update()** after changing menu state (especially on Windows)
* **Use radio groups** for mutually exclusive options
* **Use checkboxes** for toggleable features
* **Add accelerators** to common actions
* **Group related items** with separators
* **Test on all platforms** - behaviour varies

### ❌ Don't

* **Don't forget menu.Update()** - Click handlers won't work properly
* **Don't nest too deeply** - 2-3 levels maximum
* **Don't start/end with separators** - Looks unprofessional
* **Don't use tooltips on macOS** - Not supported
* **Don't hardcode platform shortcuts** - Use `CmdOrCtrl`

## Troubleshooting

### Menu Items Not Responding

**Symptom:** Click handlers don't fire

**Cause:** Forgot to call `menu.Update()` after enabling item

**Solution:**

```
menuItem.SetEnabled(true)
menu.Update()  // Add this!
```

### Menu Items Greyed Out

**Symptom:** Can't click menu items

**Cause:** Items are disabled

**Solution:**

```
menuItem.SetEnabled(true)
menu.Update()
```

### Accelerators Not Working

**Symptom:** Keyboard shortcuts don't trigger menu items

**Causes:**

1. Accelerator format incorrect
2. Conflict with system shortcuts
3. Window doesn't have focus

**Solution:**

```
// Check format
menuItem.SetAccelerator("CmdOrCtrl+S")  // ✅ Correct
menuItem.SetAccelerator("Ctrl+S")       // ❌ Wrong (macOS uses Cmd)

// Avoid conflicts
// ❌ Cmd+H (Hide Window on macOS - system shortcut)
// ✅ Cmd+Shift+H (Custom shortcut)
```

## Next Steps

* [Application Menus](https://v3.wails.io/features/menus/application) - Create application menu bars
* [Context Menus](https://v3.wails.io/features/menus/context) - Right-click context menus
* [System Tray Menus](https://v3.wails.io/features/menus/systray) - System tray/menu bar menus
* [Menu Patterns](https://v3.wails.io/guides/patterns/menus) - Common menu patterns and best practices
