---
title: Context Menus
description: Creating right-click context menus with CSS properties, context data, and platform behaviour.
sidebar_position: 1
---

# Context Menus

Context menus provide right-click functionality for your application's UI elements. Wails v3 makes it easy to create, customise, and manage context menus using CSS properties and Go APIs.

## Quick Start

1. Create a context menu in Go.
2. Add menu items.
3. Register it with a unique ID.
4. Use CSS to associate it with HTML elements.

```go title="main.go"
package main

import (
    "fmt"

    "github.com/wailsapp/wails/v3/pkg/application"
)

func main() {
    app := application.New(application.Options{})

    ctxMenu := app.NewContextMenu("my-context-menu")
    ctxMenu.Add("Open", func(ctx *application.ContextMenu) {
        fmt.Println("Open clicked")
    })
    ctxMenu.Add("Save", func(ctx *application.ContextMenu) {
        fmt.Println("Save clicked")
    })
    ctxMenu.AddSeparator()
    ctxMenu.Add("Delete", func(ctx *application.ContextMenu) {
        fmt.Println("Delete clicked")
    })

    app.Run()
}
```

```html title="index.html"
<div class="my-item" style="--custom-contextmenu: my-context-menu;">
  Right-click me
</div>
```

## CSS Association

Associate a context menu with any element using the `--custom-contextmenu` CSS property.

```css
.my-item {
    --custom-contextmenu: my-context-menu;
}
```

The value must match the ID passed to `NewContextMenu`.

## Menu Items

### Standard Items

```go
ctxMenu := app.NewContextMenu("my-menu")

// Simple label
ctxMenu.Add("Click Me", func(ctx *application.ContextMenu) {
    fmt.Println("clicked")
})

// With icon
ctxMenu.AddWithIcon("Open", icon, func(ctx *application.ContextMenu) {
    fmt.Println("open")
})

// With shortcut
ctxMenu.AddWithShortcut("Save", "CmdOrCtrl+S", func(ctx *application.ContextMenu) {
    fmt.Println("saved")
})
```

### Separators

```go
ctxMenu.AddSeparator()
```

### Disabled and Checked Items

```go
item := ctxMenu.Add("Copy", handler)
item.SetDisabled(true)

checkItem := ctxMenu.Add("Show Toolbar", handler)
checkItem.SetChecked(true)
```

## Context Data

Pass data to the context menu handler using `--custom-contextmenu-data`.

```go
ctxMenu := app.NewContextMenu("my-menu")
ctxMenu.Add("Copy", func(ctx *application.ContextMenu) {
    data := ctx.GetData("itemId")
    fmt.Println("Copy item:", data)
})
```

```html
<div style="--custom-contextmenu: my-menu; --custom-contextmenu-data: 42;">
  Right-click me
</div>
```

## Submenus

```go
subMenu := ctxMenu.NewSubmenu("Share")
subMenu.Add("Email", func(ctx *application.ContextMenu) {
    fmt.Println("email")
})
subMenu.Add("Slack", func(ctx *application.ContextMenu) {
    fmt.Println("slack")
})

ctxMenu.AddSubmenu(subMenu)
```

## Checkbox and Radio Groups

```go
// Checkbox items
ctxMenu.Add("Show Grid", func(ctx *application.ContextMenu) {
    item := ctx.GetItem("Show Grid")
    item.SetChecked(!item.IsChecked())
})

// Radio group
radioGroup := ctxMenu.NewRadioGroup("theme")
radioGroup.Add("Light", func(ctx *application.ContextMenu) {})
radioGroup.Add("Dark", func(ctx *application.ContextMenu) {})
ctxMenu.AddRadioGroup(radioGroup)
```

## Validation

Add validation logic to control when menu items are enabled or visible.

```go
item := ctxMenu.Add("Delete", func(ctx *application.ContextMenu) {
    fmt.Println("deleting")
})

item.SetValidation(func() bool {
    return selectedItems != nil
})
```

## Platform Behaviour

| Platform | Notes |
|----------|-------|
| **macOS** | Native context menu appearance. Supports menu icons. |
| **Windows** | Standard Windows context menu. Keyboard shortcuts displayed. |
| **Linux** | GTK-based menus. Appearance varies by desktop environment. |

## Dynamic Updates

Update context menu contents at runtime.

```go
ctxMenu := app.NewContextMenu("dynamic-menu")

func refreshMenu() {
    ctxMenu.Clear()
    for _, item := range getItems() {
        name := item
        ctxMenu.Add(name, func(ctx *application.ContextMenu) {
            fmt.Println("Selected:", name)
        })
    }
}
```

## See Also

- [Application Menus](/docs/features/menus/application)
- [Menu API Reference](/docs/reference/apis/menu)
