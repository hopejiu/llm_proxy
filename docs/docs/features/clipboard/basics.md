---
title: Clipboard Operations
description: Copy and paste text with the system clipboard
image: ../../../assets/images/og-image.png
---

[Skip to content](#%5Ftop) 

# Clipboard Operations

## Clipboard Operations

[Section titled “Clipboard Operations”](#clipboard-operations)

Wails provides a **unified clipboard API** that works across all platforms. Copy and paste text with simple, consistent methods on Windows, macOS, and Linux.

## Quick Start

[Section titled “Quick Start”](#quick-start)

```

// Copy text to clipboard

app.Clipboard.SetText("Hello, World!")


// Get text from clipboard

text, ok := app.Clipboard.Text()

if ok {

    fmt.Println("Clipboard:", text)

}


```

**That’s it!** Cross-platform clipboard access.

## Copying Text

[Section titled “Copying Text”](#copying-text)

### Basic Copy

[Section titled “Basic Copy”](#basic-copy)

```

success := app.Clipboard.SetText("Text to copy")

if !success {

    app.Logger.Error("Failed to copy to clipboard")

}


```

**Returns:** `bool` \- `true` if successful, `false` otherwise

### Copy from Service

[Section titled “Copy from Service”](#copy-from-service)

```

type ClipboardService struct {

    app *application.App

}


func (c *ClipboardService) CopyToClipboard(text string) bool {

    return c.app.Clipboard.SetText(text)

}


```

**Call from JavaScript:**

```

import { CopyToClipboard } from './bindings/changeme/clipboardservice'


await CopyToClipboard("Text to copy")


```

### Copy with Feedback

[Section titled “Copy with Feedback”](#copy-with-feedback)

```

func copyWithFeedback(text string) {

    if app.Clipboard.SetText(text) {

        app.Dialog.Info().

            SetTitle("Copied").

            SetMessage("Text copied to clipboard!").

            Show()

    } else {

        app.Dialog.Error().

            SetTitle("Copy Failed").

            SetMessage("Failed to copy to clipboard.").

            Show()

    }

}


```

## Pasting Text

[Section titled “Pasting Text”](#pasting-text)

### Basic Paste

[Section titled “Basic Paste”](#basic-paste)

```

text, ok := app.Clipboard.Text()

if !ok {

    app.Logger.Error("Failed to read clipboard")

    return

}


fmt.Println("Clipboard text:", text)


```

**Returns:** `(string, bool)` \- Text and success flag

### Paste from Service

[Section titled “Paste from Service”](#paste-from-service)

```

func (c *ClipboardService) PasteFromClipboard() string {

    text, ok := c.app.Clipboard.Text()

    if !ok {

        return ""

    }

    return text

}


```

**Call from JavaScript:**

```

import { PasteFromClipboard } from './bindings/changeme/clipboardservice'


const text = await PasteFromClipboard()

console.log("Pasted:", text)


```

### Paste with Validation

[Section titled “Paste with Validation”](#paste-with-validation)

```

func pasteText() (string, error) {

    text, ok := app.Clipboard.Text()

    if !ok {

        return "", errors.New("clipboard empty or unavailable")

    }


    // Validate

    if len(text) == 0 {

        return "", errors.New("clipboard is empty")

    }


    if len(text) > 10000 {

        return "", errors.New("clipboard text too large")

    }


    return text, nil

}


```

## Complete Examples

[Section titled “Complete Examples”](#complete-examples)

### Copy Button

[Section titled “Copy Button”](#copy-button)

**Go:**

```

type TextService struct {

    app *application.App

}


func (t *TextService) CopyText(text string) error {

    if !t.app.Clipboard.SetText(text) {

        return errors.New("failed to copy")

    }

    return nil

}


```

**JavaScript:**

```

import { CopyText } from './bindings/changeme/textservice'


async function copyToClipboard(text) {

    try {

        await CopyText(text)

        showNotification("Copied to clipboard!")

    } catch (error) {

        showError("Failed to copy: " + error)

    }

}


// Usage

document.getElementById('copy-btn').addEventListener('click', () => {

    const text = document.getElementById('text').value

    copyToClipboard(text)

})


```

### Paste and Process

[Section titled “Paste and Process”](#paste-and-process)

**Go:**

```

type DataService struct {

    app *application.App

}


func (d *DataService) PasteAndProcess() (string, error) {

    // Get clipboard text

    text, ok := d.app.Clipboard.Text()

    if !ok {

        return "", errors.New("clipboard unavailable")

    }


    // Process text

    processed := strings.TrimSpace(text)

    processed = strings.ToUpper(processed)


    return processed, nil

}


```

**JavaScript:**

```

import { PasteAndProcess } from './bindings/changeme/dataservice'


async function pasteAndProcess() {

    try {

        const result = await PasteAndProcess()

        document.getElementById('output').value = result

    } catch (error) {

        showError("Failed to paste: " + error)

    }

}


```

### Copy Multiple Formats

[Section titled “Copy Multiple Formats”](#copy-multiple-formats)

```

type CopyService struct {

    app *application.App

}


func (c *CopyService) CopyAsPlainText(text string) bool {

    return c.app.Clipboard.SetText(text)

}


func (c *CopyService) CopyAsJSON(data interface{}) bool {

    jsonBytes, err := json.MarshalIndent(data, "", "  ")

    if err != nil {

        return false

    }

    return c.app.Clipboard.SetText(string(jsonBytes))

}


func (c *CopyService) CopyAsCSV(rows [][]string) bool {

    var buf bytes.Buffer

    writer := csv.NewWriter(&buf)


    for _, row := range rows {

        if err := writer.Write(row); err != nil {

            return false

        }

    }


    writer.Flush()

    return c.app.Clipboard.SetText(buf.String())

}


```

### Clipboard Monitor

[Section titled “Clipboard Monitor”](#clipboard-monitor)

```

type ClipboardMonitor struct {

    app          *application.App

    lastText     string

    ticker       *time.Ticker

    stopChan     chan bool

}


func NewClipboardMonitor(app *application.App) *ClipboardMonitor {

    return &ClipboardMonitor{

        app:      app,

        stopChan: make(chan bool),

    }

}


func (cm *ClipboardMonitor) Start() {

    cm.ticker = time.NewTicker(1 * time.Second)


    go func() {

        for {

            select {

            case <-cm.ticker.C:

                cm.checkClipboard()

            case <-cm.stopChan:

                return

            }

        }

    }()

}


func (cm *ClipboardMonitor) Stop() {

    if cm.ticker != nil {

        cm.ticker.Stop()

    }

    cm.stopChan <- true

}


func (cm *ClipboardMonitor) checkClipboard() {

    text, ok := cm.app.Clipboard.Text()

    if !ok {

        return

    }


    if text != cm.lastText {

        cm.lastText = text

        cm.app.Event.Emit("clipboard-changed", text)

    }

}


```

### Copy with History

[Section titled “Copy with History”](#copy-with-history)

```

type ClipboardHistory struct {

    app     *application.App

    history []string

    maxSize int

}


func NewClipboardHistory(app *application.App) *ClipboardHistory {

    return &ClipboardHistory{

        app:     app,

        history: make([]string, 0),

        maxSize: 10,

    }

}


func (ch *ClipboardHistory) Copy(text string) bool {

    if !ch.app.Clipboard.SetText(text) {

        return false

    }


    // Add to history

    ch.history = append([]string{text}, ch.history...)


    // Limit size

    if len(ch.history) > ch.maxSize {

        ch.history = ch.history[:ch.maxSize]

    }


    return true

}


func (ch *ClipboardHistory) GetHistory() []string {

    return ch.history

}


func (ch *ClipboardHistory) RestoreFromHistory(index int) bool {

    if index < 0 || index >= len(ch.history) {

        return false

    }


    return ch.app.Clipboard.SetText(ch.history[index])

}


```

## Frontend Integration

[Section titled “Frontend Integration”](#frontend-integration)

### Using Browser Clipboard API

[Section titled “Using Browser Clipboard API”](#using-browser-clipboard-api)

For simple text, you can use the browser’s clipboard API:

```

// Copy

async function copyText(text) {

    try {

        await navigator.clipboard.writeText(text)

        console.log("Copied!")

    } catch (error) {

        console.error("Copy failed:", error)

    }

}


// Paste

async function pasteText() {

    try {

        const text = await navigator.clipboard.readText()

        return text

    } catch (error) {

        console.error("Paste failed:", error)

        return ""

    }

}


```

**Note:** Browser clipboard API requires HTTPS or localhost, and user permission.

### Using Wails Clipboard

[Section titled “Using Wails Clipboard”](#using-wails-clipboard)

For system-wide clipboard access:

```

import { CopyToClipboard, PasteFromClipboard } from './bindings/changeme/clipboardservice'


// Copy

async function copy(text) {

    const success = await CopyToClipboard(text)

    if (success) {

        console.log("Copied!")

    }

}


// Paste

async function paste() {

    const text = await PasteFromClipboard()

    return text

}


```

## Best Practices

[Section titled “Best Practices”](#best-practices)

### ✅ Do

[Section titled “✅ Do”](#-do)

* **Check return values** \- Handle failures gracefully
* **Provide feedback** \- Let users know copy succeeded
* **Validate pasted text** \- Check format and size
* **Use appropriate method** \- Browser API vs Wails API
* **Handle empty clipboard** \- Check before using
* **Trim whitespace** \- Clean pasted text

### ❌ Don’t

[Section titled “❌ Don’t”](#-dont)

* **Don’t ignore failures** \- Always check success
* **Don’t copy sensitive data** \- Clipboard is shared
* **Don’t assume format** \- Validate pasted data
* **Don’t poll too frequently** \- If monitoring clipboard
* **Don’t copy large data** \- Use files instead
* **Don’t forget security** \- Sanitise pasted content

## Platform Differences

[Section titled “Platform Differences”](#platform-differences)

### macOS

[Section titled “macOS”](#macos)

* Uses NSPasteboard
* Supports rich text (future)
* System-wide clipboard
* Clipboard history (system feature)

### Windows

[Section titled “Windows”](#windows)

* Uses Windows Clipboard API
* Supports multiple formats (future)
* System-wide clipboard
* Clipboard history (Windows 10+)

### Linux

[Section titled “Linux”](#linux)

* Uses X11/Wayland clipboard
* Primary and clipboard selections
* Varies by desktop environment
* May require clipboard manager

## Limitations

[Section titled “Limitations”](#limitations)

### Current Version

[Section titled “Current Version”](#current-version)

* **Text only** \- Images not yet supported
* **No format detection** \- Plain text only
* **No clipboard events** \- Must poll for changes
* **No clipboard history** \- Implement yourself

### Future Features

[Section titled “Future Features”](#future-features)

* Image support
* Rich text support
* Multiple formats
* Clipboard change events
* Clipboard history API

## Next Steps

[Section titled “Next Steps”](#next-steps)

Bindings 

Call Go functions from JavaScript.

[Learn More →](https://v3.wails.io/features/bindings/methods)

Events 

Use events for clipboard notifications.

[Learn More →](https://v3.wails.io/features/events/system)

dialogs 

Show copy/paste feedback.

[Learn More →](https://v3.wails.io/features/dialogs/message)

Services 

Organise clipboard code.

[Learn More →](https://v3.wails.io/features/bindings/services)

---

**Questions?** Ask in [Discord](https://discord.gg/JDdSxwjhGf) or check the [clipboard examples](https://github.com/wailsapp/wails/tree/master/v3/examples).

```json
{"@context":"https://schema.org","@type":"SoftwareApplication","name":"Wails","description":"Build beautiful desktop applications using Go and modern web technologies.","url":"https://v3.wails.io","applicationCategory":"DeveloperApplication","downloadUrl":"https://github.com/wailsapp/wails/releases","softwareVersion":"3.0","operatingSystem":["Windows","macOS","Linux"],"offers":{"@type":"Offer","price":"0","priceCurrency":"USD"},"author":{"@type":"Organization","name":"Wails Contributors","url":"https://github.com/wailsapp/wails"},"provider":{"@type":"Organization","name":"Wails Contributors","url":"https://github.com/wailsapp/wails"}}
{"@context":"https://schema.org","@type":"Organization","name":"Wails","url":"https://v3.wails.io","logo":"../../../assets/images/favicon.svg","sameAs":["https://github.com/wailsapp/wails","https://x.com/wailsapp","https://discord.gg/JDdSxwjhGf"]}
```
