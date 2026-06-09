---
title: Modal File Manager
description: A desktop application built with Wails
image: ../../../assets/images/og-image.png
---

[Skip to content](#%5Ftop) 

# Modal File Manager

![Modal File Manager](../../../assets/images/modalfilemanager.Bj36AulC_Z2o89as.webp)

[Modal File Manager](https://github.com/raguay/ModalFileManager) is a dual pane file manager using web technologies. My original design was based on NW.js and can be found [here](https://github.com/raguay/ModalFileManager-NWjs). This version uses the same Svelte based frontend code (but it has be greatly modified since the departure from NW.js), but the backend is a[Wails 2](https://wails.io/) implementation. By using this implementation, I no longer use command line `rm`, `cp`, etc. commands, but a git install has to be on the system to download themes and extensions. It is fully coded using Go and runs much faster than the previous versions.

This file manager is designed around the same principle as Vim: a state controlled keyboard actions. The number of states isn't fixed, but very programmable. Therefore, an infinite number of keyboard configurations can be created and used. This is the main difference from other file managers. There are themes and extensions available to download from GitHub.

```json
{"@context":"https://schema.org","@type":"SoftwareApplication","name":"Wails","description":"Build beautiful desktop applications using Go and modern web technologies.","url":"https://v3.wails.io","applicationCategory":"DeveloperApplication","downloadUrl":"https://github.com/wailsapp/wails/releases","softwareVersion":"3.0","operatingSystem":["Windows","macOS","Linux"],"offers":{"@type":"Offer","price":"0","priceCurrency":"USD"},"author":{"@type":"Organization","name":"Wails Contributors","url":"https://github.com/wailsapp/wails"},"provider":{"@type":"Organization","name":"Wails Contributors","url":"https://github.com/wailsapp/wails"}}
{"@context":"https://schema.org","@type":"Organization","name":"Wails","url":"https://v3.wails.io","logo":"../../../assets/images/favicon.svg","sameAs":["https://github.com/wailsapp/wails","https://x.com/wailsapp","https://discord.gg/JDdSxwjhGf"]}
```
