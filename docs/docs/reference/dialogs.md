---
title: "Dialogs API"
description: "Complete reference for native dialog APIs"
---

## Overview

The Dialogs API provides methods to show native file dialogs and message dialogs. Access dialogs through the `app.Dialog` manager.

**Dialog Types:**

* **File Dialogs** - Open and Save dialogs
* **Message Dialogs** - Info, Error, Warning, and Question dialogs

All dialogs are **native OS dialogs** that match the platform's look and feel.

## Accessing Dialogs

Dialogs are accessed through the `app.Dialog` manager:

```go
app.Dialog.OpenFile()
app.Dialog.SaveFile()
app.Dialog.Info()
app.Dialog.Question()
app.Dialog.Warning()
app.Dialog.Error()
```

## File Dialogs

### OpenFile()

```go
func (dm *DialogManager) OpenFile() *OpenFileDialogStruct
```

**OpenFileDialogStruct Methods:**

* `SetTitle(title string) *OpenFileDialogStruct`
* `AddFilter(displayName, pattern string) *OpenFileDialogStruct`
* `SetDirectory(directory string) *OpenFileDialogStruct`
* `CanChooseDirectories(canChooseDirectories bool) *OpenFileDialogStruct`
* `CanChooseFiles(canChooseFiles bool) *OpenFileDialogStruct`
* `CanCreateDirectories(canCreateDirectories bool) *OpenFileDialogStruct`
* `ShowHiddenFiles(showHiddenFiles bool) *OpenFileDialogStruct`
* `AttachToWindow(window Window) *OpenFileDialogStruct`
* `PromptForSingleSelection() (string, error)`
* `PromptForMultipleSelection() ([]string, error)`

### SaveFile()

```go
func (dm *DialogManager) SaveFile() *SaveFileDialogStruct
```

**SaveFileDialogStruct Methods:**

* `SetTitle(title string) *SaveFileDialogStruct`
* `SetFilename(filename string) *SaveFileDialogStruct`
* `AddFilter(displayName, pattern string) *SaveFileDialogStruct`
* `SetDirectory(directory string) *SaveFileDialogStruct`
* `AttachToWindow(window Window) *SaveFileDialogStruct`
* `PromptForSingleSelection() (string, error)`

## Message Dialogs

All message dialogs return `*MessageDialog` and share the same methods.

### Info(), Error(), Warning(), Question()

```go
app.Dialog.Info().SetTitle("Success").SetMessage("File saved successfully!").Show()
app.Dialog.Error().SetTitle("Error").SetMessage("Failed to save file").Show()
app.Dialog.Warning().SetTitle("Warning").SetMessage("This action cannot be undone.").Show()
app.Dialog.Question().SetTitle("Confirm").SetMessage("Do you want to save changes?")
```

### MessageDialog Methods

* `SetTitle(title string) *MessageDialog`
* `SetMessage(message string) *MessageDialog`
* `SetIcon(icon []byte) *MessageDialog`
* `AddButton(label string) *Button`
* `SetDefaultButton(button *Button) *MessageDialog`
* `SetCancelButton(button *Button) *MessageDialog`
* `AttachToWindow(window Window) *MessageDialog`
* `Show()`

### Button Methods

* `OnClick(callback func()) *Button`
* `SetAsDefault() *Button`
* `SetAsCancel() *Button`

## Dialog Types by Platform

### macOS

* Dialogs slide down from title bar
* "Sheet" style attached to parent window

### Windows

* Standard Windows dialogs
* Follows Windows design guidelines

### Linux

* GTK dialogs on GTK-based systems
* Qt dialogs on Qt-based systems

#### Linux Dialog Behavior

On Linux, the default GTK4 build uses **xdg-desktop-portal** for file dialogs. The legacy GTK3 path (`-tags gtk3`) retains full programmatic control.

| Option                 | GTK3 (-tags gtk3) | GTK4 (default) | Notes                                                 |
| ---------------------- | ------------------ | -------------- | ----------------------------------------------------- |
| ShowHiddenFiles()      | ✅ Works           | ❌ No effect    | User controls via dialog's UI toggle                  |
| CanCreateDirectories() | ✅ Works           | ❌ No effect    | Always enabled in the portal                          |
| ResolvesAliases()      | ✅ Works           | ❌ No effect    | Portal handles symlink resolution                     |
| SetButtonText()        | ✅ Works           | ✅ Works        | Custom accept button text works                       |
