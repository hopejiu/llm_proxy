---
title: "Custom URL Protocols"
description: "Register custom URL schemes to launch your application from links"
---

Custom URL protocols (also called URL schemes) allow your application to be launched when users click links with your custom protocol, such as `myapp://action` or `myapp://open/document`.

## Overview

Custom protocols enable:

* **Deep linking**: Launch your app with specific data
* **Browser integration**: Handle links from web pages
* **Email links**: Open your app from email clients
* **Inter-app communication**: Launch from other applications

## Configuration

Define custom protocols in `build/config.yml`:

```yaml
protocols:
  - scheme: myapp
    description: "My Application Protocol"
```

In Go code, listen for launch-with-URL via the `ApplicationLaunchedWithUrl` event:

```go
package main

import (
    "github.com/wailsapp/wails/v3/pkg/application"
    "github.com/wailsapp/wails/v3/pkg/events"
)

func main() {
    app := application.New(application.Options{
        Name:        "My Application",
        Description: "My awesome application",
    })

    app.Event.OnApplicationEvent(events.Common.ApplicationLaunchedWithUrl, func(e *application.ApplicationEvent) {
        handleCustomURL(e.Context().URL())
    })

    app.Run()
}

func handleCustomURL(url string) {
    println("Received URL:", url)
}
```

## Platform Registration

Custom protocols are registered differently on each platform.

### Windows NSIS Installer

Wails v3 automatically registers custom protocols when using NSIS installers. The NSIS template includes built-in macros that register protocols during installation and remove them during uninstall.

### Windows MSIX Package

Custom protocols are also automatically registered when using MSIX packaging. The generated manifest includes protocol registrations from your `build/config.yml`.

### macOS

On macOS, protocols are registered via your `Info.plist` file. Wails automatically generates the `Info.plist` with your protocols when you build with `wails3 build`.

### Linux

On Linux, protocols are registered via `.desktop` files. Wails generates a desktop entry file with protocol handlers when you build with `wails3 build`.

## Frontend Integration

Handle navigation events in your frontend:

```js
import { Events } from '@wailsio/runtime'

Events.On('navigate', (event) => {
    const { type, id, page, section, user } = event.data

    if (type === 'document') {
        router.push(`/document/${id}`)
    } else if (page === 'settings') {
        router.push(`/settings/${section}`)
    } else if (page === 'user') {
        router.push(`/user/${user}`)
    }
})
```

## Security Considerations

* **Validate all input** — Never trust URLs from external sources
* **Prevent injection attacks** — Never execute URLs directly as code or SQL
* **Use descriptive scheme names** — `mycompany-myapp` instead of `mca`
* **Handle errors gracefully** — Log invalid URLs, don't crash

## Best Practices

* **Use lowercase scheme names**
* **Keep schemes short and memorable**
* **Use hierarchical paths for resources**
* **Include query parameters for optional data**
* **URL-encode special characters**
* **Test on all platforms** — Protocol handling varies
