---
title: "Menu API"
description: "Complete reference for the Menu API"
---

## Overview

The Menu API provides methods to create and manage application menus, context menus, and system tray menus.

**Menu Types:**

* **Application Menus** - Top menu bar (File, Edit, etc.)
* **Context Menus** - Right-click menus
* **System Tray Menus** - Menus in the system tray/notification area

## Creating Menus

### NewMenu()

```go
func (a *App) NewMenu() *Menu
```

## Menu Methods

### Add()

```go
func (m *Menu) Add(label string) *MenuItem
```

### AddSubmenu()

```go
func (m *Menu) AddSubmenu(label string) *Menu
```

### AddSeparator()

```go
func (m *Menu) AddSeparator()
```

### AddCheckbox()

```go
func (m *Menu) AddCheckbox(label string, checked bool) *MenuItem
```

### AddRadio()

```go
func (m *Menu) AddRadio(label string, checked bool) *MenuItem
```

### Update()

```go
func (m *Menu) Update()
```

**Important:** Always call `Update()` after modifying menu item properties.

## Menu Item Methods

### OnClick()

```go
func (mi *MenuItem) OnClick(callback func(ctx *application.Context)) *MenuItem
```

### SetLabel()

```go
func (mi *MenuItem) SetLabel(label string) *MenuItem
```

### SetEnabled()

```go
func (mi *MenuItem) SetEnabled(enabled bool) *MenuItem
```

### SetChecked()

```go
func (mi *MenuItem) SetChecked(checked bool) *MenuItem
```

### Checked()

```go
func (mi *MenuItem) Checked() bool
```

### SetAccelerator()

```go
func (mi *MenuItem) SetAccelerator(accelerator string) *MenuItem
```

### SetTooltip()

```go
func (mi *MenuItem) SetTooltip(tooltip string) *MenuItem
```

### SetHidden()

```go
func (mi *MenuItem) SetHidden(hidden bool) *MenuItem
```

## Application Menu

### app.Menu.Set()

```go
func (mm *MenuManager) Set(menu *Menu)
```

**Platform notes:**

* **macOS:** Menu appears in the top menu bar
* **Windows/Linux:** Menu appears in the window title bar
* **macOS:** Automatically adds application menu with app name

## Context Menus

### app.ContextMenu.New() / app.ContextMenu.Add()

```go
func (cm *ContextMenuManager) New() *ContextMenu
func (cm *ContextMenuManager) Add(name string, menu *ContextMenu)
func (cm *ContextMenuManager) Get(name string) (*ContextMenu, bool)
func (cm *ContextMenuManager) Remove(name string)
```

Context menus are triggered via the `--custom-contextmenu` CSS property on HTML elements.

## System Tray Menu

### app.SystemTray.New()

```go
func (sm *SystemTrayManager) New() *SystemTray
```

### SetIcon(), SetMenu(), SetTooltip(), OnClick()

```go
func (st *SystemTray) SetIcon(icon []byte) *SystemTray
func (st *SystemTray) SetMenu(menu *Menu) *SystemTray
func (st *SystemTray) SetTooltip(tooltip string)
func (st *SystemTray) OnClick(callback func()) *SystemTray
```

## Best Practices

* **Use standard accelerators** - Follow platform conventions (Ctrl+C for copy, etc.)
* **Call Update() after changes** - Menu won't reflect changes otherwise
* **Group related items** - Use separators to organize menu items
* **Disable unavailable actions** - Don't hide, disable with SetEnabled(false)
* **Use clear labels** - Be concise and descriptive
* **Follow platform conventions** - macOS vs Windows/Linux menu patterns
