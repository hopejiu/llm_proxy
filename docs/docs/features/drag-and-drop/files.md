---
title: File Drop
description: Accept files dragged from the operating system into your application
image: ../../../assets/images/og-image.png
---

[Skip to content](#%5Ftop) 

# File Drop

Wails lets users drag files from the operating system (file manager, desktop) into your application. Unlike HTML5 drag-and-drop which only works within the browser, this gives you access to actual file paths on disk.

## Enable File Drop

[Section titled “Enable File Drop”](#enable-file-drop)

File drop is disabled by default. To enable it, set `EnableFileDrop: true` in your window options:

```

window := app.Window.NewWithOptions(application.WebviewWindowOptions{

    Title:          "My App",

    Width:          800,

    Height:         600,

    EnableFileDrop: true,

})


```

When `EnableFileDrop` is `false` (the default), files dragged from the OS are blocked - they won’t open in the webview or trigger any events. This prevents accidental navigation when users drag files over your app.

## Define Drop Zones

[Section titled “Define Drop Zones”](#define-drop-zones)

Drop zones tell Wails which elements should accept files. Files dropped outside a drop zone are ignored.

Add the `data-file-drop-target` attribute to any element:

```

<div id="upload" class="drop-zone" data-file-drop-target>

    Drop files here

</div>


```

You can have multiple drop zones. The element’s `id` and CSS classes are passed to your Go code, so you can handle drops differently depending on where files land.

## Style Drag Hover

[Section titled “Style Drag Hover”](#style-drag-hover)

When files are dragged over a drop zone, Wails adds the `file-drop-target-active` class. This lets you provide visual feedback so users know where they can drop:

```

.drop-zone {

    border: 2px dashed #ccc;

    padding: 40px;

    text-align: center;

    transition: all 0.2s ease;

}


.drop-zone.file-drop-target-active {

    border-color: #007bff;

    background-color: rgba(0, 123, 255, 0.1);

}


```

The class is removed automatically when files leave the zone or are dropped.

## Detect Dropped Files

[Section titled “Detect Dropped Files”](#detect-dropped-files)

When files are dropped on a valid drop zone, Wails fires a `WindowFilesDropped` event. The event context contains the full filesystem paths of all dropped files:

```

import "github.com/wailsapp/wails/v3/pkg/events"


window.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {

    files := event.Context().DroppedFiles()


    for _, file := range files {

        fmt.Println("Dropped:", file)

    }

})


```

The paths are absolute, like `/home/user/documents/report.pdf` or `C:\Users\Name\Documents\report.pdf`.

## Get Drop Target Info

[Section titled “Get Drop Target Info”](#get-drop-target-info)

When you have multiple drop zones, you can find out which one received the files using `DropTargetDetails()`:

```

window.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {

    files := event.Context().DroppedFiles()

    details := event.Context().DropTargetDetails()


    fmt.Printf("Dropped on element: id=%s, classes=%v\n",

        details.ElementID, details.ClassList)

    fmt.Printf("Position: x=%d, y=%d\n", details.X, details.Y)

})


```

This lets you route files to different handlers:

```

switch details.ElementID {

case "images":

    handleImageUpload(files)

case "documents":

    handleDocumentUpload(files)

}


```

## Complete Example

[Section titled “Complete Example”](#complete-example)

**Go:**

```

window := app.Window.NewWithOptions(application.WebviewWindowOptions{

    Title:          "File Uploader",

    EnableFileDrop: true,

})


window.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {

    files := event.Context().DroppedFiles()

    details := event.Context().DropTargetDetails()


    // Send to frontend

    app.Event.Emit("files-dropped", map[string]any{

        "files":   files,

        "target":  details.ElementID,

    })

})


```

**HTML:**

```

<div id="images" class="drop-zone" data-file-drop-target>

    Drop images here

</div>


<div id="documents" class="drop-zone" data-file-drop-target>

    Drop documents here

</div>


<style>

    .drop-zone {

        border: 2px dashed #ccc;

        border-radius: 8px;

        padding: 40px;

        text-align: center;

        margin: 20px;

        transition: all 0.2s ease;

    }


    .drop-zone.file-drop-target-active {

        border-color: #007bff;

        background-color: rgba(0, 123, 255, 0.1);

    }

</style>


```

## Full Window Drop

[Section titled “Full Window Drop”](#full-window-drop)

If you want files to be droppable anywhere in your app, add the attribute to the body element:

```

<body data-file-drop-target>

    <!-- Your app content -->

</body>


```

You can use a CSS overlay to indicate the entire window is a drop target:

```

body.file-drop-target-active::after {

    content: "Drop files anywhere";

    position: fixed;

    inset: 0;

    display: flex;

    align-items: center;

    justify-content: center;

    font-size: 24px;

    color: #007bff;

    background: rgba(255, 255, 255, 0.9);

    pointer-events: none;

}


```

## Combining with HTML Drag & Drop

[Section titled “Combining with HTML Drag & Drop”](#combining-with-html-drag--drop)

You can use both external file drops and internal HTML drag-and-drop in the same application. When `EnableFileDrop` is `true`, Wails intercepts external file drags but lets internal HTML5 drags pass through normally.

To distinguish between them in your HTML drop zone handlers, check if the drag contains files:

```

zone.addEventListener('dragenter', (e) => {

    // Skip external file drags - Wails handles these

    if (e.dataTransfer?.types.includes('Files')) {

        return;

    }

    // Handle internal HTML5 drags

    zone.classList.add('drag-over');

});


zone.addEventListener('drop', (e) => {

    // Skip external file drops - Wails handles these

    if (e.dataTransfer?.types.includes('Files')) {

        return;

    }

    e.preventDefault();

    zone.classList.remove('drag-over');

    // Handle internal drop

});


```

This ensures your HTML drop handlers only respond to internal drags (like moving list items), while Wails handles external file drops separately via the `WindowFilesDropped` event.

## Next Steps

[Section titled “Next Steps”](#next-steps)

* [HTML Drag & Drop](https://v3.wails.io/features/drag-and-drop/html) \- Drag elements within your app
* [Window Options](https://v3.wails.io/features/windows/options) \- All window configuration options

```json
{"@context":"https://schema.org","@type":"SoftwareApplication","name":"Wails","description":"Build beautiful desktop applications using Go and modern web technologies.","url":"https://v3.wails.io","applicationCategory":"DeveloperApplication","downloadUrl":"https://github.com/wailsapp/wails/releases","softwareVersion":"3.0","operatingSystem":["Windows","macOS","Linux"],"offers":{"@type":"Offer","price":"0","priceCurrency":"USD"},"author":{"@type":"Organization","name":"Wails Contributors","url":"https://github.com/wailsapp/wails"},"provider":{"@type":"Organization","name":"Wails Contributors","url":"https://github.com/wailsapp/wails"}}
{"@context":"https://schema.org","@type":"Organization","name":"Wails","url":"https://v3.wails.io","logo":"../../../assets/images/favicon.svg","sameAs":["https://github.com/wailsapp/wails","https://x.com/wailsapp","https://discord.gg/JDdSxwjhGf"]}
```
