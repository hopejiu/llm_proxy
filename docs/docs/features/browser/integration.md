---
title: Browser Integration
description: Open URLs and files in the user's default web browser
image: ../../../assets/images/og-image.png
---

[Skip to content](#%5Ftop) 

# Browser Integration

Wails provides simple browser integration through the BrowserManager API, allowing your application to open URLs and files in the user’s default web browser. This is useful for opening external links, documentation, or files that should be handled by the browser.

## Accessing the Browser Manager

[Section titled “Accessing the Browser Manager”](#accessing-the-browser-manager)

The browser manager is accessed through the `Browser` property on your application instance:

```

app := application.New(application.Options{

    Name: "Browser Integration Demo",

})


// Access the browser manager

browser := app.Browser


```

## Opening URLs

[Section titled “Opening URLs”](#opening-urls)

### Open Web URLs

[Section titled “Open Web URLs”](#open-web-urls)

Open URLs in the user’s default web browser:

```

// Open a website

err := app.Browser.OpenURL("https://wails.io")

if err != nil {

    app.Logger.Error("Failed to open URL", "error", err)

}


// Open specific pages

err = app.Browser.OpenURL("https://github.com/wailsapp/wails")

if err != nil {

    app.Logger.Error("Failed to open GitHub", "error", err)

}


```

### Open Local URLs

[Section titled “Open Local URLs”](#open-local-urls)

Open local development servers or local network resources:

```

// Open local development server

err := app.Browser.OpenURL("http://localhost:3000")

if err != nil {

    app.Logger.Error("Failed to open local server", "error", err)

}


// Open network resource

err = app.Browser.OpenURL("http://192.168.1.100:8080")

if err != nil {

    app.Logger.Error("Failed to open network resource", "error", err)

}


```

## Opening Files

[Section titled “Opening Files”](#opening-files)

### Open HTML Files

[Section titled “Open HTML Files”](#open-html-files)

Open local HTML files in the browser:

```

// Open an HTML file

err := app.Browser.OpenFile("/path/to/documentation.html")

if err != nil {

    app.Logger.Error("Failed to open HTML file", "error", err)

}


// Open generated reports

reportPath := "/tmp/report.html"

err = app.Browser.OpenFile(reportPath)

if err != nil {

    app.Logger.Error("Failed to open report", "error", err)

}


```

### Open Other File Types

[Section titled “Open Other File Types”](#open-other-file-types)

Open various file types that browsers can handle:

```

// Open PDF files

err := app.Browser.OpenFile("/path/to/document.pdf")

if err != nil {

    app.Logger.Error("Failed to open PDF", "error", err)

}


// Open image files

err = app.Browser.OpenFile("/path/to/image.png")

if err != nil {

    app.Logger.Error("Failed to open image", "error", err)

}


// Open text files

err = app.Browser.OpenFile("/path/to/readme.txt")

if err != nil {

    app.Logger.Error("Failed to open text file", "error", err)

}


```

## Common Use Cases

[Section titled “Common Use Cases”](#common-use-cases)

### Help and Documentation

[Section titled “Help and Documentation”](#help-and-documentation)

Provide easy access to help resources:

```

// Create help menu

func setupHelpMenu(app *application.App) {

    menu := app.Menu.New()

    helpMenu := menu.AddSubmenu("Help")


    helpMenu.Add("Online Documentation").OnClick(func(ctx *application.Context) {

        err := app.Browser.OpenURL("https://docs.yourapp.com")

        if err != nil {

            app.Dialog.Error().

                SetTitle("Error").

                SetMessage("Could not open documentation").

                Show()

        }

    })


    helpMenu.Add("GitHub Repository").OnClick(func(ctx *application.Context) {

        app.Browser.OpenURL("https://github.com/youruser/yourapp")

    })


    helpMenu.Add("Report Issue").OnClick(func(ctx *application.Context) {

        app.Browser.OpenURL("https://github.com/youruser/yourapp/issues/new")

    })

}


```

### External Links in Content

[Section titled “External Links in Content”](#external-links-in-content)

Handle external links from your application content:

```

func handleExternalLink(app *application.App, url string) {

    // Validate the URL before opening

    if !isValidURL(url) {

        app.Logger.Warn("Invalid URL", "url", url)

        return

    }


    // Optionally confirm with user

    dialog := app.Dialog.Question()

    dialog.SetTitle("Open External Link")

    dialog.SetMessage(fmt.Sprintf("Open %s in your browser?", url))


    dialog.AddButton("Open").OnClick(func() {

        err := app.Browser.OpenURL(url)

        if err != nil {

            app.Logger.Error("Failed to open URL", "url", url, "error", err)

        }

    })


    dialog.AddButton("Cancel")

    dialog.Show()

}


func isValidURL(url string) bool {

    parsed, err := url.Parse(url)

    return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https")

}


```

### Export and View Reports

[Section titled “Export and View Reports”](#export-and-view-reports)

Generate and open reports in the browser:

```

import (

    "html/template"

    "os"

    "path/filepath"

)


func generateAndOpenReport(app *application.App, data interface{}) error {

    // Create temporary file for the report

    tmpDir := os.TempDir()

    reportPath := filepath.Join(tmpDir, "report.html")


    // Generate HTML report

    tmpl := `

<!DOCTYPE html>

<html>

<head>

    <title>Application Report</title>

    <style>

        body { font-family: Arial, sans-serif; margin: 40px; }

        .header { border-bottom: 2px solid #333; padding-bottom: 10px; }

        .data { margin-top: 20px; }

    </style>

</head>

<body>

    <div class="header">

        <h1>Application Report</h1>

        <p>Generated: {{.Timestamp}}</p>

    </div>

    <div class="data">

        <!-- Report content here -->

        {{range .Items}}

        <p>{{.}}</p>

        {{end}}

    </div>

</body>

</html>`


    // Write report to file

    file, err := os.Create(reportPath)

    if err != nil {

        return err

    }

    defer file.Close()


    t, err := template.New("report").Parse(tmpl)

    if err != nil {

        return err

    }


    err = t.Execute(file, data)

    if err != nil {

        return err

    }


    // Open in browser

    return app.Browser.OpenFile(reportPath)

}


```

### Development Tools

[Section titled “Development Tools”](#development-tools)

Open development resources during development:

```

func setupDevelopmentMenu(app *application.App) {

    if !app.Env.Info().Debug {

        return // Only show in debug mode

    }


    menu := app.Menu.New()

    devMenu := menu.AddSubmenu("Development")


    devMenu.Add("Open DevTools").OnClick(func(ctx *application.Context) {

        // This would open browser devtools if available

        window := app.Window.Current()

        if window != nil {

            window.OpenDevTools()

        }

    })


    devMenu.Add("View Source").OnClick(func(ctx *application.Context) {

        // Open source code repository

        app.Browser.OpenURL("https://github.com/youruser/yourapp")

    })


    devMenu.Add("API Documentation").OnClick(func(ctx *application.Context) {

        // Open local API docs

        app.Browser.OpenURL("http://localhost:8080/docs")

    })

}


```

## Error Handling

[Section titled “Error Handling”](#error-handling)

### Graceful Error Handling

[Section titled “Graceful Error Handling”](#graceful-error-handling)

Always handle potential errors when opening URLs or files:

```

func openURLWithFallback(app *application.App, url string, fallbackMessage string) {

    err := app.Browser.OpenURL(url)

    if err != nil {

        app.Logger.Error("Failed to open URL", "url", url, "error", err)


        // Show fallback dialog with URL

        dialog := app.Dialog.Info()

        dialog.SetTitle("Unable to Open Link")

        dialog.SetMessage(fmt.Sprintf("%s\n\nURL: %s", fallbackMessage, url))

        dialog.Show()

    }

}


// Usage

openURLWithFallback(app,

    "https://docs.example.com",

    "Please open the following URL manually in your browser:")


```

### User Feedback

[Section titled “User Feedback”](#user-feedback)

Provide feedback when operations succeed or fail:

```

func openURLWithFeedback(app *application.App, url string) {

    err := app.Browser.OpenURL(url)

    if err != nil {

        // Show error dialog

        app.Dialog.Error().

            SetTitle("Browser Error").

            SetMessage(fmt.Sprintf("Could not open URL: %s", err.Error())).

            Show()

    } else {

        // Optionally show success notification

        app.Logger.Info("URL opened successfully", "url", url)

    }

}


```

## Platform Considerations

[Section titled “Platform Considerations”](#platform-considerations)

* [  macOS ](#tab-panel-62)
* [  Windows ](#tab-panel-63)
* [  Linux ](#tab-panel-64)

On macOS:

* Uses the `open` command to launch the default browser
* Respects user’s default browser setting in System Preferences
* May prompt for permission if the application is sandboxed
* Handles `file://` URLs correctly for local files

On Windows:

* Uses Windows Shell API to open URLs
* Respects default browser setting in Windows Settings
* Handles Windows path formats correctly
* May show security warnings for untrusted URLs

On Linux:

* Attempts to use `xdg-open` first, falls back to other methods
* Behavior varies by desktop environment
* Respects `BROWSER` environment variable if set
* May require additional packages in minimal installations

## Best Practices

[Section titled “Best Practices”](#best-practices)

1. **Always Handle Errors**: Browser operations can fail for various reasons:  
```  
if err := app.Browser.OpenURL(url); err != nil {  
    app.Logger.Error("Failed to open browser", "error", err)  
    // Provide fallback or user notification  
}  
```
2. **Validate URLs**: Ensure URLs are well-formed before opening:  
```  
func isValidHTTPURL(str string) bool {  
    u, err := url.Parse(str)  
    return err == nil && (u.Scheme == "http" || u.Scheme == "https")  
}  
```
3. **User Confirmation**: For external links, consider asking user permission:  
```  
// Show confirmation dialog before opening external links  
confirmAndOpen(app, "https://external-site.com")  
```
4. **Secure File Paths**: When opening files, ensure paths are safe:  
```  
func openSafeFile(app *application.App, filename string) error {  
    // Ensure file exists and is readable  
    if _, err := os.Stat(filename); err != nil {  
        return err  
    }  
    return app.Browser.OpenFile(filename)  
}  
```

## Complete Example

[Section titled “Complete Example”](#complete-example)

Here’s a complete example showing various browser integration patterns:

```

package main


import (

    "fmt"

    "os"

    "path/filepath"

    "github.com/wailsapp/wails/v3/pkg/application"

)


func main() {

    app := application.New(application.Options{

        Name: "Browser Integration Demo",

    })


    // Setup menu with browser actions

    setupMenu(app)


    // Create main window

    window := app.Window.New()

    window.SetTitle("Browser Integration")


    err := app.Run()

    if err != nil {

        panic(err)

    }

}


func setupMenu(app *application.App) {

    menu := app.Menu.New()


    // File menu

    fileMenu := menu.AddSubmenu("File")

    fileMenu.Add("Generate Report").OnClick(func(ctx *application.Context) {

        generateHTMLReport(app)

    })


    // Help menu

    helpMenu := menu.AddSubmenu("Help")

    helpMenu.Add("Documentation").OnClick(func(ctx *application.Context) {

        openWithConfirmation(app, "https://docs.example.com")

    })

    helpMenu.Add("Support").OnClick(func(ctx *application.Context) {

        openWithConfirmation(app, "https://support.example.com")

    })


    app.Menu.Set(menu)

}


func openWithConfirmation(app *application.App, url string) {

    dialog := app.Dialog.Question()

    dialog.SetTitle("Open External Link")

    dialog.SetMessage(fmt.Sprintf("Open %s in your browser?", url))


    dialog.AddButton("Open").OnClick(func() {

        if err := app.Browser.OpenURL(url); err != nil {

            showError(app, "Failed to open URL", err)

        }

    })


    dialog.AddButton("Cancel")

    dialog.Show()

}


func generateHTMLReport(app *application.App) {

    // Create temporary HTML file

    tmpDir := os.TempDir()

    reportPath := filepath.Join(tmpDir, "demo_report.html")


    html := `

<!DOCTYPE html>

<html>

<head>

    <title>Demo Report</title>

    <style>

        body { font-family: Arial, sans-serif; margin: 40px; }

        .header { color: #333; border-bottom: 1px solid #ccc; }

    </style>

</head>

<body>

    <div class="header">

        <h1>Application Report</h1>

        <p>This is a sample report generated by the application.</p>

    </div>

    <div class="content">

        <h2>Report Details</h2>

        <p>This report was generated to demonstrate browser integration.</p>

    </div>

</body>

</html>`


    err := os.WriteFile(reportPath, []byte(html), 0644)

    if err != nil {

        showError(app, "Failed to create report", err)

        return

    }


    // Open in browser

    err = app.Browser.OpenFile(reportPath)

    if err != nil {

        showError(app, "Failed to open report", err)

    }

}


func showError(app *application.App, message string, err error) {

    app.Dialog.Error().

        SetTitle("Error").

        SetMessage(fmt.Sprintf("%s: %v", message, err)).

        Show()

}


```

Pro Tip

Consider providing fallback mechanisms for when browser operations fail, such as copying URLs to clipboard or showing them in a dialog for manual opening.

Warning

Always validate URLs and file paths before opening them to prevent security issues. Be cautious about opening user-provided URLs without validation.

```json
{"@context":"https://schema.org","@type":"SoftwareApplication","name":"Wails","description":"Build beautiful desktop applications using Go and modern web technologies.","url":"https://v3.wails.io","applicationCategory":"DeveloperApplication","downloadUrl":"https://github.com/wailsapp/wails/releases","softwareVersion":"3.0","operatingSystem":["Windows","macOS","Linux"],"offers":{"@type":"Offer","price":"0","priceCurrency":"USD"},"author":{"@type":"Organization","name":"Wails Contributors","url":"https://github.com/wailsapp/wails"},"provider":{"@type":"Organization","name":"Wails Contributors","url":"https://github.com/wailsapp/wails"}}
{"@context":"https://schema.org","@type":"Organization","name":"Wails","url":"https://v3.wails.io","logo":"../../../assets/images/favicon.svg","sameAs":["https://github.com/wailsapp/wails","https://x.com/wailsapp","https://discord.gg/JDdSxwjhGf"]}
```
