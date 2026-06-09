---
title: "Custom Dialogs"
description: "Building custom dialog windows with Wails windows, channels, and events."
---

Custom dialogs give you full control over appearance and behavior by building
them as regular Wails application windows. Use the standard dialog patterns
for confirmation, input, progress, login, settings, and multi-step wizards.

## Creating a Custom Dialog

A custom dialog is a Wails window with specific options that make it behave
like a dialog: centered, modal-like, and closable.

```go
func (a *App) OpenSettingsDialog() {
    window, err := runtime.NewWindow(a.ctx, &options.Window{
        Title:     "Settings",
        Width:     500,
        Height:    400,
        Frameless: true,
        Center:    true,
        Hidden:    true,
    })
    if err != nil {
        return
    }

    window.Load("index.html#/settings")
    window.Show()
}
```

## Patterns

### Confirmation Dialog

A simple yes/no confirmation using a channel to wait for the user's choice.

```go
func (a *App) ConfirmAction(message string) bool {
    ch := make(chan bool)

    window, _ := runtime.NewWindow(a.ctx, &options.Window{
        Title:     "Confirm",
        Width:     350,
        Height:    180,
        Frameless: true,
        Center:    true,
        AlwaysOnTop: true,
    })

    events.On("confirm:yes", func() {
        ch <- true
        window.Close()
    })

    events.On("confirm:no", func() {
        ch <- false
        window.Close()
    })

    window.Load("index.html#/confirm")
    window.Show()

    return <-ch
}
```

Frontend handler:

```js
// renderer/confirm.js
eventsOn("confirm:yes", () => { /* ... */ })
eventsOn("confirm:no", () => { /* ... */ })
```

### Input Dialog

Collect text input from the user and return it through a channel.

```go
func (a *App) PromptInput(title string, defaultValue string) string {
    ch := make(chan string)

    window, _ := runtime.NewWindow(a.ctx, &options.Window{
        Title:     title,
        Width:     400,
        Height:    200,
        Frameless: true,
        Center:    true,
    })

    events.On("input:submit", func(data string) {
        ch <- data
        window.Close()
    })

    events.On("input:cancel", func() {
        ch <- defaultValue
        window.Close()
    })

    window.Load("index.html#/input")
    window.Show()

    return <-ch
}
```

### Progress Dialog

Display a progress bar for long-running operations. Use events to stream
progress updates from Go to the frontend.

```go
func (a *App) ShowProgressDialog(task func(func(int, string))) {
    window, _ := runtime.NewWindow(a.ctx, &options.Window{
        Title:     "Processing",
        Width:     400,
        Height:    150,
        Frameless: true,
        Center:    true,
        AlwaysOnTop: true,
    })

    window.Load("index.html#/progress")
    window.Show()

    go func() {
        task(func(percent int, status string) {
            events.Emit("progress:update", map[string]interface{}{
                "percent": percent,
                "status":  status,
            })
        })

        events.Emit("progress:complete", nil)
        window.Close()
    }()
}

// Usage:
a.ShowProgressDialog(func(update func(int, string)) {
    for i := 0; i <= 100; i += 10 {
        time.Sleep(200 * time.Millisecond)
        update(i, fmt.Sprintf("Processing item %d/10...", i/10+1))
    }
})
```

### Login Dialog

Collect credentials securely using a dedicated dialog window.

```go
type Credentials struct {
    Username string
    Password string
}

func (a *App) ShowLoginDialog() *Credentials {
    ch := make(chan *Credentials)

    window, _ := runtime.NewWindow(a.ctx, &options.Window{
        Title:     "Login",
        Width:     400,
        Height:    280,
        Frameless: true,
        Center:    true,
    })

    events.On("login:submit", func(data string) {
        var creds Credentials
        json.Unmarshal([]byte(data), &creds)
        ch <- &creds
        window.Close()
    })

    events.On("login:cancel", func() {
        ch <- nil
        window.Close()
    })

    window.Load("index.html#/login")
    window.Show()

    return <-ch
}
```

### Settings Dialog

A settings dialog typically loads and saves configuration. Use events to
communicate changes back to Go.

```go
func (a *App) OpenSettings() {
    window, _ := runtime.NewWindow(a.ctx, &options.Window{
        Title:     "Settings",
        Width:     550,
        Height:    450,
        Center:    true,
        AlwaysOnTop: true,
    })

    events.On("settings:save", func(data string) {
        var config map[string]interface{}
        json.Unmarshal([]byte(data), &config)
        a.saveConfig(config)
        window.Close()
    })

    events.On("settings:cancel", func() {
        window.Close()
    })

    window.Load("index.html#/settings")
    window.Show()
}
```

### Wizard Dialog

Multi-step dialogs for guided workflows. Use a state machine to track the
current step.

```go
type WizardStep struct {
    Title    string
    Template string
}

func (a *App) ShowWizard() {
    steps := []WizardStep{
        {Title: "Welcome", Template: "wizard-step-1"},
        {Title: "Configuration", Template: "wizard-step-2"},
        {Title: "Confirmation", Template: "wizard-step-3"},
    }

    window, _ := runtime.NewWindow(a.ctx, &options.Window{
        Title:  "Setup Wizard",
        Width:  500,
        Height: 350,
        Center: true,
    })

    currentStep := 0

    events.On("wizard:next", func() {
        currentStep++
        if currentStep >= len(steps) {
            events.Emit("wizard:complete", nil)
            window.Close()
            return
        }
        events.Emit("wizard:step", steps[currentStep])
    })

    events.On("wizard:prev", func() {
        if currentStep > 0 {
            currentStep--
            events.Emit("wizard:step", steps[currentStep])
        }
    })

    events.On("wizard:cancel", func() {
        window.Close()
    })

    window.Load("index.html#/wizard")
    window.Show()
}
```

## Window Options

Useful window options for dialog behavior:

```go
options.Window{
    Title:       "Dialog",
    Width:       400,
    Height:      300,
    Frameless:   true,   // removes title bar for a clean dialog look
    AlwaysOnTop: true,   // keeps dialog above other windows
    Hidden:      true,   // load content before showing
    Center:      true,   // center on parent or screen
}
```

## Communication Between Go and Frontend

### Using Events

Events provide a simple pub/sub mechanism for cross-language communication.

```go
// Go -> Frontend
events.Emit("dialog:data", payload)

// Frontend -> Go
events.On("dialog:response", func(data string) {
    // handle response
})
```

### Using Channels

Channels allow synchronous waiting for a result in Go.

```go
ch := make(chan string)

events.On("dialog:result", func(data string) {
    ch <- data
})

window.Load("index.html#/dialog")
window.Show()

result := <-ch
window.Close()
```

## Styling with CSS

Style your custom dialogs with CSS loaded through the frontend. Since
frameless windows have no native chrome, you provide your own close buttons
and drag regions.

```css
.dialog-container {
    display: flex;
    flex-direction: column;
    height: 100vh;
    font-family: -apple-system, BlinkMacSystemFont, sans-serif;
}

.dialog-titlebar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 8px 12px;
    background: #f5f5f5;
    -webkit-app-region: drag;
    user-select: none;
}

.dialog-titlebar button {
    -webkit-app-region: no-drag;
}

.dialog-body {
    flex: 1;
    padding: 20px;
    overflow-y: auto;
}

.dialog-footer {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    padding: 12px;
    border-top: 1px solid #e0e0e0;
}

.dialog-footer button {
    padding: 6px 16px;
    border: 1px solid #ccc;
    border-radius: 4px;
    cursor: pointer;
}

.dialog-footer button.primary {
    background: #0066cc;
    color: white;
    border-color: #0066cc;
}
```

Add `-webkit-app-region: drag` to the title bar so the user can move the
dialog, and `-webkit-app-region: no-drag` on interactive elements so they
remain clickable.
