---
title: "What's New in Wails v3"
description: "Discover the major improvements and new features in Wails v3"
---

Wails v3 introduces significant changes from v2. It replaces the single-window, declarative API with a more flexible procedural approach.

## Multiple Windows

Wails v3 introduces the ability to create and manage multiple windows within a single application. Each window can be independently configured.

## System Tray Integration

Robust support for system tray functionality, including window attachment, comprehensive menu support, and adaptive icon display.

## Improved Bindings Generation

Binding generation is now done using a sophisticated static analyzer. Requires only a single command: `wails3 generate bindings`.

## Improved Build System

All the heavy lifting that the v2 build system did has been added as tool commands in the CLI, orchestrated via Taskfile.

## Improved Events

Wails now emits events for various runtime operations and system activities. Event hooks can be registered to handle specific events synchronously and can cancel events.

## Wails Markup Language (wml)

An experimental feature to call runtime methods using plain HTML, similar to htmx.

## Examples

More examples available in the [examples](https://github.com/wailsapp/wails/tree/master/v3/examples) directory.
