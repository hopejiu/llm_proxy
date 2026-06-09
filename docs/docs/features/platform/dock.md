---
title: Dock &#38; Taskbar
description: Manage dock icon visibility and display badges on macOS and Windows
image: ../../../assets/images/og-image.png
---

[Skip to content](#%5Ftop) 

# Dock & Taskbar

## Introduction

[Section titled “Introduction”](#introduction)

Wails provides a cross-platform Dock service for desktop applications. This service allows you to:

* Hide and show the application icon in the macOS Dock
* Display badges on your application tile or dock/taskbar icon (macOS and Windows)

## Basic Usage

[Section titled “Basic Usage”](#basic-usage)

### Creating the Service

[Section titled “Creating the Service”](#creating-the-service)

First, initialize the dock service:

```

import "github.com/wailsapp/wails/v3/pkg/application"

import "github.com/wailsapp/wails/v3/pkg/services/dock"


// Create a new Dock service

dockService := dock.New()


// Register the service with the application

app := application.New(application.Options{

    Services: []application.Service{

        application.NewService(dockService),

    },

})


```

### Creating the Service with Custom Badge Options (Windows Only)

[Section titled “Creating the Service with Custom Badge Options (Windows Only)”](#creating-the-service-with-custom-badge-options-windows-only)

On Windows, you can customize the badge appearance with various options:

```

import "github.com/wailsapp/wails/v3/pkg/application"

import "github.com/wailsapp/wails/v3/pkg/services/dock"

import "image/color"


// Create a dock service with custom badge options

options := dock.BadgeOptions{

    TextColour:       color.RGBA{255, 255, 255, 255}, // White text

    BackgroundColour: color.RGBA{0, 0, 255, 255},     // Blue background

    FontName:         "consolab.ttf",                 // Bold Consolas font

    FontSize:         20,                             // Font size for single character

    SmallFontSize:    14,                             // Font size for multiple characters

}


dockService := dock.NewWithOptions(options)


// Register the service with the application

app := application.New(application.Options{

    Services: []application.Service{

        application.NewService(dockService),

    },

})


```

## Dock Operations

[Section titled “Dock Operations”](#dock-operations)

### Hiding the dock app icon

[Section titled “Hiding the dock app icon”](#hiding-the-dock-app-icon)

Hide the app icon from the macOS Dock:

```

// Hide the app icon

dockService.HideAppIcon()


```

### Showing the dock app icon

[Section titled “Showing the dock app icon”](#showing-the-dock-app-icon)

Show the app icon in the macOS Dock:

```

// Show the app icon

dockService.ShowAppIcon()


```

## Badge Operations

[Section titled “Badge Operations”](#badge-operations)

### Setting a Badge

[Section titled “Setting a Badge”](#setting-a-badge)

Set a badge on the application tile/dock icon:

```

// Set a default badge

dockService.SetBadge("")


// Set a numeric badge

dockService.SetBadge("3")


// Set a text badge

dockService.SetBadge("New")


```

### Setting a Custom Badge (Windows Only)

[Section titled “Setting a Custom Badge (Windows Only)”](#setting-a-custom-badge-windows-only)

Set a badge with one-off options applied:

```

options := dock.BadgeOptions{

    BackgroundColour: color.RGBA{0, 255, 255, 255},

    FontName:         "arialb.ttf", // System font

    FontSize:         16,

    SmallFontSize:    10,

    TextColour:       color.RGBA{0, 0, 0, 255},

}


// Set a default badge

dockService.SetCustomBadge("", options)


// Set a numeric badge

dockService.SetCustomBadge("3", options)


// Set a text badge

dockService.SetCustomBadge("New", options)


```

### Removing a Badge

[Section titled “Removing a Badge”](#removing-a-badge)

Remove the badge from the application icon:

```

dockService.RemoveBadge()


```

### Getting the set badge

[Section titled “Getting the set badge”](#getting-the-set-badge)

```

dockService.GetBadge()


```

## Platform Considerations

[Section titled “Platform Considerations”](#platform-considerations)

* [  macOS ](#tab-panel-92)
* [  Windows ](#tab-panel-93)
* [  Linux ](#tab-panel-94)

On macOS:

* The dock icon can be **hidden** and **shown**
* Badges are displayed directly on the dock icon
* Badge options are **not customizable** (any options passed to `NewWithOptions`/`SetCustomBadge` are ignored)
* The standard macOS dock badge styling is used and automatically adapts to appearance
* Label overflow is handled by the system
* Providing an empty label displays a default badge of ”●”

On Windows:

* Hiding/showing the taskbar icon is not currently supported by this service
* Badges are displayed as an overlay icon in the taskbar
* Badges support text values
* Badge appearance can be customized via `BadgeOptions`
* The application must have a window for badges to display
* A smaller font size is automatically used for multi-character labels
* Label overflow is not handled
* Customization options:  
   * **TextColour**: Text color (default: white)  
   * **BackgroundColour**: Badge background color (default: red)  
   * **FontName**: Font file name (default: “segoeuib.ttf”)  
   * **FontSize**: Font size for single character (default: 18)  
   * **SmallFontSize**: Font size for multiple characters (default: 14)

On Linux:

* Dock icon visibility and badge functionality are not available

## Best Practices

[Section titled “Best Practices”](#best-practices)

1. **When hiding the dock icon (macOS):**  
   * Ensure users can still access your app (e.g., via [system tray](https://v3.wails.io/learn/systray/))  
   * Include a “Quit” option in your alternative UI  
   * The app won’t appear in Command+Tab switcher  
   * Open windows remain visible and functional  
   * Closing all windows may not quit the app (macOS behavior varies)  
   * Users lose the standard way to quit via Dock right-click
2. **Use badges sparingly:**  
   * Too many badge updates can distract users  
   * Reserve badges for important notifications
3. **Keep badge text short:**  
   * Numeric badges are most effective  
   * On macOS, text badges should be brief
4. **For Windows badge customization:**  
   * Ensure high contrast between text and background colors  
   * Test with different text lengths as font size decreases with length  
   * Use common system fonts to ensure availability

## API Reference

[Section titled “API Reference”](#api-reference)

### Service Management

[Section titled “Service Management”](#service-management)

| Method                               | Description                                                                                                 |
| ------------------------------------ | ----------------------------------------------------------------------------------------------------------- |
| New()                                | Creates a new dock service                                                                                  |
| NewWithOptions(options BadgeOptions) | Creates a new dock service with custom badge options (Windows only; options are ignored on macOS and Linux) |

### Dock Operations

[Section titled “Dock Operations”](#dock-operations-1)

| Method        | Description                                         |
| ------------- | --------------------------------------------------- |
| HideAppIcon() | Hides the app icon from the macOS Dock (macOS only) |
| ShowAppIcon() | Shows the app icon in the macOS Dock (macOS only)   |

### Badge Operations

[Section titled “Badge Operations”](#badge-operations-1)

| Method                                                   | Description                                                                     |
| -------------------------------------------------------- | ------------------------------------------------------------------------------- |
| SetBadge(label string) error                             | Sets a badge with the specified label                                           |
| SetCustomBadge(label string, options BadgeOptions) error | Sets a badge with the specified label and custom styling options (Windows only) |
| RemoveBadge() error                                      | Removes the badge from the application icon                                     |
| GetBadge() \*string                                      | Gets the current badge                                                          |

### Structs and Types

[Section titled “Structs and Types”](#structs-and-types)

```

// Options for customizing badge appearance (Windows only)

type BadgeOptions struct {

    TextColour       color.RGBA  // Color of the badge text

    BackgroundColour color.RGBA  // Color of the badge background

    FontName         string      // Font file name (e.g., "segoeuib.ttf")

    FontSize         int         // Font size for single character

    SmallFontSize    int         // Font size for multiple characters

}


```

```json
{"@context":"https://schema.org","@type":"SoftwareApplication","name":"Wails","description":"Build beautiful desktop applications using Go and modern web technologies.","url":"https://v3.wails.io","applicationCategory":"DeveloperApplication","downloadUrl":"https://github.com/wailsapp/wails/releases","softwareVersion":"3.0","operatingSystem":["Windows","macOS","Linux"],"offers":{"@type":"Offer","price":"0","priceCurrency":"USD"},"author":{"@type":"Organization","name":"Wails Contributors","url":"https://github.com/wailsapp/wails"},"provider":{"@type":"Organization","name":"Wails Contributors","url":"https://github.com/wailsapp/wails"}}
{"@context":"https://schema.org","@type":"Organization","name":"Wails","url":"https://v3.wails.io","logo":"../../../assets/images/favicon.svg","sameAs":["https://github.com/wailsapp/wails","https://x.com/wailsapp","https://discord.gg/JDdSxwjhGf"]}
```
