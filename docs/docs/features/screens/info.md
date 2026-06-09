---
title: Screen Information
description: Get information about displays and monitors
image: ../../../assets/images/og-image.png
---

[Skip to content](#%5Ftop) 

# Screen Information

## Screen Information

[Section titled “Screen Information”](#screen-information)

Wails provides a **unified screen API** that works across all platforms. Get screen information, detect multiple monitors, query screen properties (size, position, DPI), identify the primary display, and handle DPI scaling with consistent code.

## Quick Start

[Section titled “Quick Start”](#quick-start)

```

// Get all screens

screens := app.Screen.GetAll()


for _, screen := range screens {

    fmt.Printf("Screen: %s (%dx%d)\n",

        screen.Name, screen.Size.Width, screen.Size.Height)

}


// Get primary screen

primary := app.Screen.GetPrimary()

fmt.Printf("Primary: %s\n", primary.Name)


```

**That’s it!** Cross-platform screen information.

## Getting Screen Information

[Section titled “Getting Screen Information”](#getting-screen-information)

### All Screens

[Section titled “All Screens”](#all-screens)

```

screens := app.Screen.GetAll()


for _, screen := range screens {

    fmt.Printf("ID: %s\n", screen.ID)

    fmt.Printf("Name: %s\n", screen.Name)

    fmt.Printf("Size: %dx%d\n", screen.Size.Width, screen.Size.Height)

    fmt.Printf("Position: %d,%d\n", screen.X, screen.Y)

    fmt.Printf("Scale: %.2f\n", screen.ScaleFactor)

    fmt.Printf("Primary: %v\n", screen.IsPrimary)

    fmt.Println("---")

}


```

### Primary Screen

[Section titled “Primary Screen”](#primary-screen)

```

primary := app.Screen.GetPrimary()


fmt.Printf("Primary screen: %s\n", primary.Name)

fmt.Printf("Resolution: %dx%d\n", primary.Size.Width, primary.Size.Height)

fmt.Printf("Scale factor: %.2f\n", primary.ScaleFactor)


```

### Current Screen

[Section titled “Current Screen”](#current-screen)

Get the screen containing a window:

```

screen, err := window.GetScreen()

if err != nil {

    // handle error

}

fmt.Printf("Window is on: %s\n", screen.Name)


```

### Screen by ID

[Section titled “Screen by ID”](#screen-by-id)

```

screen := app.Screen.GetByID("screen-id")

if screen != nil {

    fmt.Printf("Found screen: %s\n", screen.Name)

}


```

## Screen Properties

[Section titled “Screen Properties”](#screen-properties)

### Screen Structure

[Section titled “Screen Structure”](#screen-structure)

```

type Screen struct {

    ID               string  // Unique identifier

    Name             string  // Display name

    ScaleFactor      float32 // DPI scale (1.0, 1.5, 2.0, etc.)

    X, Y             int     // Position (top-left, logical pixels)

    Size             Size    // Logical size (Width, Height)

    Bounds           Rect    // Logical bounds (X, Y, Width, Height)

    PhysicalBounds   Rect    // Physical bounds (scaled)

    WorkArea         Rect    // Logical work area excluding taskbars/menubars

    PhysicalWorkArea Rect    // Physical work area

    IsPrimary        bool    // Is this the primary screen?

    Rotation         float32 // Screen rotation in degrees (e.g. 0, 90, 180, 270)

}


// Size and Rect are simple value types:

type Size struct{ Width, Height int }

type Rect struct{ X, Y, Width, Height int }


```

Read pixel sizes via `screen.Size.Width` / `screen.Size.Height` — there are no top-level `Width`/`Height` fields.

### Physical vs Logical Pixels

[Section titled “Physical vs Logical Pixels”](#physical-vs-logical-pixels)

```

screen := app.Screen.GetPrimary()


// Logical pixels (what you use)

logicalWidth := screen.Size.Width

logicalHeight := screen.Size.Height


// Physical pixels (actual display)

physicalWidth := int(float32(screen.Size.Width) * screen.ScaleFactor)

physicalHeight := int(float32(screen.Size.Height) * screen.ScaleFactor)


fmt.Printf("Logical: %dx%d\n", logicalWidth, logicalHeight)

fmt.Printf("Physical: %dx%d\n", physicalWidth, physicalHeight)

fmt.Printf("Scale: %.2f\n", screen.ScaleFactor)


```

**Common scale factors:**

* `1.0` \- Standard DPI (96 DPI)
* `1.25` \- 125% scaling (120 DPI)
* `1.5` \- 150% scaling (144 DPI)
* `2.0` \- 200% scaling (192 DPI) - Retina
* `3.0` \- 300% scaling (288 DPI) - 4K/5K

## Window Positioning

[Section titled “Window Positioning”](#window-positioning)

### Centre on Screen

[Section titled “Centre on Screen”](#centre-on-screen)

```

func centreOnScreen(window *application.WebviewWindow, screen *Screen) {

    windowWidth, windowHeight := window.Size()


    x := screen.X + (screen.Size.Width-windowWidth)/2

    y := screen.Y + (screen.Size.Height-windowHeight)/2


    window.SetPosition(x, y)

}


```

### Position on Specific Screen

[Section titled “Position on Specific Screen”](#position-on-specific-screen)

```

func moveToScreen(window *application.WebviewWindow, screenIndex int) {

    screens := app.Screen.GetAll()


    if screenIndex < 0 || screenIndex >= len(screens) {

        return

    }


    screen := screens[screenIndex]


    // Centre on target screen

    centreOnScreen(window, screen)

}


```

### Position Relative to Screen

[Section titled “Position Relative to Screen”](#position-relative-to-screen)

```

// Top-left corner

func positionTopLeft(window *application.WebviewWindow, screen *Screen) {

    window.SetPosition(screen.X+10, screen.Y+10)

}


// Top-right corner

func positionTopRight(window *application.WebviewWindow, screen *Screen) {

    windowWidth, _ := window.Size()

    window.SetPosition(screen.X+screen.Size.Width-windowWidth-10, screen.Y+10)

}


// Bottom-right corner

func positionBottomRight(window *application.WebviewWindow, screen *Screen) {

    windowWidth, windowHeight := window.Size()

    window.SetPosition(

        screen.X+screen.Size.Width-windowWidth-10,

        screen.Y+screen.Size.Height-windowHeight-10,

    )

}


```

## Multi-Monitor Support

[Section titled “Multi-Monitor Support”](#multi-monitor-support)

### Detect Multiple Monitors

[Section titled “Detect Multiple Monitors”](#detect-multiple-monitors)

```

func hasMultipleMonitors() bool {

    return len(app.Screen.GetAll()) > 1

}


func getMonitorCount() int {

    return len(app.Screen.GetAll())

}


```

### List All Monitors

[Section titled “List All Monitors”](#list-all-monitors)

```

func listMonitors() {

    screens := app.Screen.GetAll()


    fmt.Printf("Found %d monitor(s):\n", len(screens))


    for i, screen := range screens {

        primary := ""

        if screen.IsPrimary {

            primary = " (Primary)"

        }


        fmt.Printf("%d. %s%s\n", i+1, screen.Name, primary)

        fmt.Printf("   Resolution: %dx%d\n", screen.Size.Width, screen.Size.Height)

        fmt.Printf("   Position: %d,%d\n", screen.X, screen.Y)

        fmt.Printf("   Scale: %.2fx\n", screen.ScaleFactor)

    }

}


```

### Choose Monitor

[Section titled “Choose Monitor”](#choose-monitor)

```

func chooseMonitor() (*Screen, error) {

    screens := app.Screen.GetAll()


    if len(screens) == 1 {

        return screens[0], nil

    }


    // Show dialog to choose

    var options []string

    for i, screen := range screens {

        primary := ""

        if screen.IsPrimary {

            primary = " (Primary)"

        }

        options = append(options,

            fmt.Sprintf("%d. %s%s - %dx%d",

                i+1, screen.Name, primary, screen.Size.Width, screen.Size.Height))

    }


    // Use dialog to select

    // (Implementation depends on your dialog system)


    return screens[0], nil

}


```

## Complete Examples

[Section titled “Complete Examples”](#complete-examples)

### Multi-Monitor Window Manager

[Section titled “Multi-Monitor Window Manager”](#multi-monitor-window-manager)

```

type MultiMonitorManager struct {

    app     *application.App

    windows map[int]*application.WebviewWindow

}


func NewMultiMonitorManager(app *application.App) *MultiMonitorManager {

    return &MultiMonitorManager{

        app:     app,

        windows: make(map[int]*application.WebviewWindow),

    }

}


func (m *MultiMonitorManager) CreateWindowOnScreen(screenIndex int) error {

    screens := m.app.Screen.GetAll()


    if screenIndex < 0 || screenIndex >= len(screens) {

        return errors.New("invalid screen index")

    }


    screen := screens[screenIndex]


    // Create window

    window := m.app.Window.NewWithOptions(application.WebviewWindowOptions{

        Title:  fmt.Sprintf("Window on %s", screen.Name),

        Width:  800,

        Height: 600,

    })


    // Centre on screen

    x := screen.X + (screen.Size.Width-800)/2

    y := screen.Y + (screen.Size.Height-600)/2

    window.SetPosition(x, y)


    window.Show()


    m.windows[screenIndex] = window

    return nil

}


func (m *MultiMonitorManager) CreateWindowOnEachScreen() {

    screens := m.app.Screen.GetAll()


    for i := range screens {

        m.CreateWindowOnScreen(i)

    }

}


```

### Screen Change Detection

[Section titled “Screen Change Detection”](#screen-change-detection)

```

type ScreenMonitor struct {

    app           *application.App

    lastScreens   []*Screen

    changeHandler func([]*Screen)

}


func NewScreenMonitor(app *application.App) *ScreenMonitor {

    return &ScreenMonitor{

        app:         app,

        lastScreens: app.Screen.GetAll(),

    }

}


func (sm *ScreenMonitor) OnScreenChange(handler func([]*Screen)) {

    sm.changeHandler = handler

}


func (sm *ScreenMonitor) Start() {

    ticker := time.NewTicker(2 * time.Second)


    go func() {

        for range ticker.C {

            sm.checkScreens()

        }

    }()

}


func (sm *ScreenMonitor) checkScreens() {

    current := sm.app.Screen.GetAll()


    if len(current) != len(sm.lastScreens) {

        sm.lastScreens = current

        if sm.changeHandler != nil {

            sm.changeHandler(current)

        }

    }

}


```

### DPI-Aware Window Sizing

[Section titled “DPI-Aware Window Sizing”](#dpi-aware-window-sizing)

```

func createDPIAwareWindow(screen *Screen) *application.WebviewWindow {

    // Base size at 1.0 scale

    baseWidth := 800

    baseHeight := 600


    // Adjust for DPI

    width := int(float32(baseWidth) * screen.ScaleFactor)

    height := int(float32(baseHeight) * screen.ScaleFactor)


    window := app.Window.NewWithOptions(application.WebviewWindowOptions{

        Title:  "DPI-Aware Window",

        Width:  width,

        Height: height,

    })


    // Centre on screen

    x := screen.X + (screen.Size.Width-width)/2

    y := screen.Y + (screen.Size.Height-height)/2

    window.SetPosition(x, y)


    return window

}


```

### Screen Layout Visualiser

[Section titled “Screen Layout Visualiser”](#screen-layout-visualiser)

```

func visualiseScreenLayout() string {

    screens := app.Screen.GetAll()


    var layout strings.Builder

    layout.WriteString("Screen Layout:\n\n")


    for i, screen := range screens {

        primary := ""

        if screen.IsPrimary {

            primary = " [PRIMARY]"

        }


        layout.WriteString(fmt.Sprintf("Screen %d: %s%s\n", i+1, screen.Name, primary))

        layout.WriteString(fmt.Sprintf("  Position: (%d, %d)\n", screen.X, screen.Y))

        layout.WriteString(fmt.Sprintf("  Size: %dx%d\n", screen.Size.Width, screen.Size.Height))

        layout.WriteString(fmt.Sprintf("  Scale: %.2fx\n", screen.ScaleFactor))

        layout.WriteString(fmt.Sprintf("  Physical: %dx%d\n",

            int(float32(screen.Size.Width)*screen.ScaleFactor),

            int(float32(screen.Size.Height)*screen.ScaleFactor)))

        layout.WriteString("\n")

    }


    return layout.String()

}


```

## Best Practices

[Section titled “Best Practices”](#best-practices)

### ✅ Do

[Section titled “✅ Do”](#-do)

* **Check screen count** \- Handle single and multiple monitors
* **Use logical pixels** \- Wails handles DPI automatically
* **Centre windows** \- Better UX than fixed positions
* **Validate positions** \- Ensure windows are visible
* **Handle screen changes** \- Monitors can be added/removed
* **Test on different DPI** \- 100%, 125%, 150%, 200%

### ❌ Don’t

[Section titled “❌ Don’t”](#-dont)

* **Don’t hardcode positions** \- Use screen dimensions
* **Don’t assume primary screen** \- User might have multiple
* **Don’t ignore scale factor** \- Important for DPI awareness
* **Don’t position off-screen** \- Validate coordinates
* **Don’t forget screen changes** \- Laptops dock/undock
* **Don’t use physical pixels** \- Use logical pixels

## Platform Differences

[Section titled “Platform Differences”](#platform-differences)

### macOS

[Section titled “macOS”](#macos)

* Retina displays (2x scale factor)
* Multiple displays common
* Coordinate system: (0,0) at bottom-left
* Spaces (virtual desktops) affect positioning

### Windows

[Section titled “Windows”](#windows)

* Various DPI scaling (100%, 125%, 150%, 200%)
* Multiple displays common
* Coordinate system: (0,0) at top-left
* Per-monitor DPI awareness

### Linux

[Section titled “Linux”](#linux)

* Varies by desktop environment
* X11 vs Wayland differences
* DPI scaling support varies
* Multiple displays supported

## Next Steps

[Section titled “Next Steps”](#next-steps)

Windows 

Learn about window management.

[Learn More →](https://v3.wails.io/features/windows/basics)

Window Options 

Configure window appearance.

[Learn More →](https://v3.wails.io/features/windows/options)

Multiple Windows 

Multi-window patterns.

[Learn More →](https://v3.wails.io/features/windows/multiple)

Bindings 

Call Go functions from JavaScript.

[Learn More →](https://v3.wails.io/features/bindings/methods)

---

**Questions?** Ask in [Discord](https://discord.gg/JDdSxwjhGf) or check the [screen examples](https://github.com/wailsapp/wails/tree/master/v3/examples).

```json
{"@context":"https://schema.org","@type":"SoftwareApplication","name":"Wails","description":"Build beautiful desktop applications using Go and modern web technologies.","url":"https://v3.wails.io","applicationCategory":"DeveloperApplication","downloadUrl":"https://github.com/wailsapp/wails/releases","softwareVersion":"3.0","operatingSystem":["Windows","macOS","Linux"],"offers":{"@type":"Offer","price":"0","priceCurrency":"USD"},"author":{"@type":"Organization","name":"Wails Contributors","url":"https://github.com/wailsapp/wails"},"provider":{"@type":"Organization","name":"Wails Contributors","url":"https://github.com/wailsapp/wails"}}
{"@context":"https://schema.org","@type":"Organization","name":"Wails","url":"https://v3.wails.io","logo":"../../../assets/images/favicon.svg","sameAs":["https://github.com/wailsapp/wails","https://x.com/wailsapp","https://discord.gg/JDdSxwjhGf"]}
```
