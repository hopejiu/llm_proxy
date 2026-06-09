---
title: Syso files on macOS
description: Troubleshoot Syso file build errors on macOS
image: ../../assets/images/og-image.png
---

[Skip to content](#%5Ftop) 

# Syso files on macOS

## Problem

[Section titled “Problem”](#problem)

When trying to build a Wails application on macOS, the build fails with an error similar to the following:

```

Error: Users/runner/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.1.darwin-arm64/pkg/tool/darwin_arm64/link: running clang failed: exit status 1

ld: unknown file type in '/private/var/folders/ml/x_tvfgn50_s7p67dm1ypcqqm0000gn/T/go-link-774134794/000000.o'

clang: error: linker command failed with exit code 1 (use -v to see invocation)


```

## Why is this happening?

[Section titled “Why is this happening?”](#why-is-this-happening)

When building for Windows, Wails will generate `.syso` files for the application icon, window icon, and menu icon. These files are required for the application to build on Windows. These `.syso` files can be found in the project’s root directory and can cause issues when building for macOS.

## Solution

[Section titled “Solution”](#solution)

To fix this issue, you can remove the syso files from the project’s root directory or if you are generating them yourself, name them `wails_windows_<arch>.syso` e.g. `wails_windows_arm64.syso`.

```json
{"@context":"https://schema.org","@type":"SoftwareApplication","name":"Wails","description":"Build beautiful desktop applications using Go and modern web technologies.","url":"https://v3.wails.io","applicationCategory":"DeveloperApplication","downloadUrl":"https://github.com/wailsapp/wails/releases","softwareVersion":"3.0","operatingSystem":["Windows","macOS","Linux"],"offers":{"@type":"Offer","price":"0","priceCurrency":"USD"},"author":{"@type":"Organization","name":"Wails Contributors","url":"https://github.com/wailsapp/wails"},"provider":{"@type":"Organization","name":"Wails Contributors","url":"https://github.com/wailsapp/wails"}}
{"@context":"https://schema.org","@type":"Organization","name":"Wails","url":"https://v3.wails.io","logo":"../../assets/images/favicon.svg","sameAs":["https://github.com/wailsapp/wails","https://x.com/wailsapp","https://discord.gg/JDdSxwjhGf"]}
```
