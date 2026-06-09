---
title: Snippet Expander
description: A desktop application built with Wails
image: ../../../assets/images/og-image.png
---

[Skip to content](#%5Ftop) 

# Snippet Expander

![Snippet Expander Screenshot](../../../assets/images/snippetexpandergui-select-snippet.DMXGnGdj_Z1gOrVR.webp)

Screenshot of Snippet Expander's Select Snippet window

![Snippet Expander Screenshot](../../../assets/images/snippetexpandergui-add-snippet.D_sdBq9P_Z1cwI6n.webp)

Screenshot of Snippet Expander's Add Snippet screen

![Snippet Expander Screenshot](../../../assets/images/snippetexpandergui-search-and-paste.BqVVnETz_rzAXO.webp)

Screenshot of Snippet Expander's Search & Paste window

[Snippet Expander](https://snippetexpander.org) is "Your little expandable text snippets helper", for Linux.

Snippet Expander comprises of a GUI application built with Wails for managing snippets and settings, with a Search & Paste window mode for quickly selecting and pasting a snippet.

The Wails based GUI, go-lang CLI and vala-lang auto expander daemon all communicate with a go-lang daemon via D-Bus. The daemon does the majority of the work, managing the database of snippets and common settings, and providing services for expanding and pasting snippets etc.

Check out the[source code](https://git.sr.ht/~ianmjones/snippetexpander/tree/trunk/item/cmd/snippetexpandergui/app.go#L38)to see how the Wails app sends messages from the UI to the backend that are then sent to the daemon, and subscribes to a D-Bus event to monitor changes to snippets via another instance of the app or CLI and show them instantly in the UI via a Wails event.

```json
{"@context":"https://schema.org","@type":"SoftwareApplication","name":"Wails","description":"Build beautiful desktop applications using Go and modern web technologies.","url":"https://v3.wails.io","applicationCategory":"DeveloperApplication","downloadUrl":"https://github.com/wailsapp/wails/releases","softwareVersion":"3.0","operatingSystem":["Windows","macOS","Linux"],"offers":{"@type":"Offer","price":"0","priceCurrency":"USD"},"author":{"@type":"Organization","name":"Wails Contributors","url":"https://github.com/wailsapp/wails"},"provider":{"@type":"Organization","name":"Wails Contributors","url":"https://github.com/wailsapp/wails"}}
{"@context":"https://schema.org","@type":"Organization","name":"Wails","url":"https://v3.wails.io","logo":"../../../assets/images/favicon.svg","sameAs":["https://github.com/wailsapp/wails","https://x.com/wailsapp","https://discord.gg/JDdSxwjhGf"]}
```
