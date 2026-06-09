---
title: CFN Tracker
description: A desktop application built with Wails
image: ../../../assets/images/og-image.png
---

[Skip to content](#%5Ftop) 

# CFN Tracker

![CFN Tracker](../../../assets/images/cfntracker.DVKFqP-h_1zcGSb.webp)

[CFN Tracker](https://github.com/williamsjokvist/cfn-tracker) \- Track any Street Fighter 6 or V CFN profile's live matches. Check[the website](https://cfn.williamsjokvist.se/) to get started.

## Features

[Section titled "Features"](#features)

* Real-time match tracking
* Storing match logs and statistics
* Support for displaying live stats to OBS via Browser Source
* Support for both SF6 and SFV
* Ability for users to create their own OBS Browser themes with CSS

### Major tech used alongside Wails

[Section titled "Major tech used alongside Wails"](#major-tech-used-alongside-wails)

* [Task](https://github.com/go-task/task) \- wrapping the Wails CLI to make common commands easy to use
* [React](https://github.com/facebook/react) \- chosen for its rich ecosystem (radix, framer-motion)
* [Bun](https://github.com/oven-sh/bun) \- used for its fast dependency resolution and build-time
* [Rod](https://github.com/go-rod/rod) \- headless browser automation for authentication and polling changes
* [SQLite](https://github.com/mattn/go-sqlite3) \- used for storing matches, sessions and profiles
* [Server-sent events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent%5Fevents) \- a http stream to send tracking updates to OBS browser sources
* [i18next](https://github.com/i18next/) \- with backend connector to serve localization objects from the Go layer
* [xstate](https://github.com/statelyai/xstate) \- state machines for auth process and tracking

```json
{"@context":"https://schema.org","@type":"SoftwareApplication","name":"Wails","description":"Build beautiful desktop applications using Go and modern web technologies.","url":"https://v3.wails.io","applicationCategory":"DeveloperApplication","downloadUrl":"https://github.com/wailsapp/wails/releases","softwareVersion":"3.0","operatingSystem":["Windows","macOS","Linux"],"offers":{"@type":"Offer","price":"0","priceCurrency":"USD"},"author":{"@type":"Organization","name":"Wails Contributors","url":"https://github.com/wailsapp/wails"},"provider":{"@type":"Organization","name":"Wails Contributors","url":"https://github.com/wailsapp/wails"}}
{"@context":"https://schema.org","@type":"Organization","name":"Wails","url":"https://v3.wails.io","logo":"../../../assets/images/favicon.svg","sameAs":["https://github.com/wailsapp/wails","https://x.com/wailsapp","https://discord.gg/JDdSxwjhGf"]}
```
