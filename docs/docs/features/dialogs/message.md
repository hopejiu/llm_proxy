---
title: "Message Dialogs"
description: "Using the app.Dialog manager to show info, warning, error, and question dialogs."
---

Message dialogs provide a simple way to communicate with users through the
application's built-in dialog manager. Wails wraps the native OS dialog APIs,
so dialogs look and feel native on each platform.

## The Dialog Manager

Access the dialog manager through `app.Dialog`.

```go
err := app.Dialog.Message("File saved successfully.").Info()
```

## Dialog Types

### Info

Show informational messages to the user.

```go
func (a *App) ShowInfo() {
    app.Dialog.Message("Export complete.").Info()
}
```

### Warning

Show warning messages that require attention.

```go
func (a *App) ShowWarning() {
    app.Dialog.Message("Disk space is running low.").Warning()
}
```

### Error

Show error messages when something goes wrong.

```go
func (a *App) ShowError() {
    app.Dialog.Message("Failed to save file.").Error()
}
```

### Question

Ask the user for confirmation or a decision. Question dialogs support buttons,
default/cancel configuration, and click handlers.

```go
func (a *App) AskQuestion() bool {
    result := app.Dialog.Message("Do you want to proceed?").
        Question().
        SetDefaultButton("Yes").
        SetCancelButton("No").
        AddButton("Yes", func(result dialog.MessageDialogResult) {
            // handle yes
        }).
        AddButton("No", func(result dialog.MessageDialogResult) {
            // handle no
        })

    _ = result
    return false
}
```

## Question Dialog Features

### AddButton

Add custom buttons to a question dialog. Each button can have its own click
handler.

```go
app.Dialog.Message("How would you like to proceed?").
    Question().
    AddButton("Save", func(r dialog.MessageDialogResult) {
        a.saveWork()
    }).
    AddButton("Discard", func(r dialog.MessageDialogResult) {
        a.discardWork()
    }).
    AddButton("Cancel", func(r dialog.MessageDialogResult) {
        // do nothing
    })
```

### OnClick

Handle the raw dialog response. The callback receives the button label that
was clicked.

```go
app.Dialog.Message("Delete this item?").
    Question().
    AddButton("Delete", nil).
    AddButton("Cancel", nil).
    OnClick(func(result dialog.MessageDialogResult) {
        if result.Button == "Delete" {
            a.deleteItem()
        }
    })
```

### SetDefaultButton

Mark a button as the default action, triggered by pressing Enter.

```go
app.Dialog.Message("Save changes?").
    Question().
    AddButton("Save", nil).
    AddButton("Cancel", nil).
    SetDefaultButton("Save")
```

### SetCancelButton

Mark a button as the cancel action, triggered by pressing Escape.

```go
app.Dialog.Message("Save changes?").
    Question().
    AddButton("Save", nil).
    AddButton("Cancel", nil).
    SetCancelButton("Cancel")
```

## Custom Icons

Override the default dialog icon by providing a path to an image file.

```go
app.Dialog.Message("Important notification").
    Question().
    SetIcon("/path/to/custom-icon.png").
    AddButton("OK", nil)
```

## Window Attachment

Attach a dialog to a specific window so it appears as a sheet on macOS and
stays above its parent on other platforms.

```go
func (a *App) ShowAttachedDialog() {
    window := a.GetMainWindow()
    app.Dialog.Message("Operation complete.").
        Info().
        SetWindow(window)
}
```

## Complete Examples

### Confirm Destructive Action

```go
func (a *App) ConfirmDelete(itemName string) {
    app.Dialog.Message(fmt.Sprintf("Are you sure you want to delete '%s'?", itemName)).
        Question().
        SetDefaultButton("Cancel").
        SetCancelButton("Cancel").
        AddButton("Delete", func(r dialog.MessageDialogResult) {
            a.performDelete(itemName)
        }).
        AddButton("Cancel", nil)
}
```

### Quit Confirmation

```go
func (a *App) ConfirmQuit() {
    app.Dialog.Message("Save changes before quitting?").
        Question().
        AddButton("Save", func(r dialog.MessageDialogResult) {
            a.saveAll()
            runtime.Quit()
        }).
        AddButton("Don't Save", func(r dialog.MessageDialogResult) {
            runtime.Quit()
        }).
        AddButton("Cancel", nil).
        SetDefaultButton("Save").
        SetCancelButton("Cancel")
}
```

### Update Dialog

```go
func (a *App) CheckForUpdates() {
    hasUpdate, version, err := a.checkUpdateServer()
    if err != nil {
        app.Dialog.Message("Failed to check for updates.").Error()
        return
    }

    if !hasUpdate {
        app.Dialog.Message("You are running the latest version.").Info()
        return
    }

    app.Dialog.Message(fmt.Sprintf("Update available: v%s. Download now?", version)).
        Question().
        AddButton("Download", func(r dialog.MessageDialogResult) {
            a.downloadUpdate(version)
        }).
        AddButton("Later", nil).
        SetDefaultButton("Download")
}
```

## Platform Differences

| Platform | Behavior |
|----------|----------|
| **macOS** | Dialogs appear as sheets attached to the active window when a window is specified. Standalone dialogs appear as modal panels. |
| **Windows** | Uses the TaskDialog API with native styling. Supports advanced features like command links and expanded content. |
| **Linux** | Uses GTK dialog widgets. Appearance depends on the desktop environment and GTK theme. |

Some platform-specific features may not be available everywhere. For example,
custom icons on Windows TaskDialog are handled differently than on macOS. Always
test dialog behavior on each target platform.
