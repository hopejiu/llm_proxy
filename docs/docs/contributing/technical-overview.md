---
title: Technical Overview
description: High-level architecture and internal design of Wails v3.
---

# Technical Overview

This document provides a high-level view of how Wails v3 works internally. It
is intended for contributors and advanced users who want to understand the
codebase before diving into specific subsystems.

## Architecture

Wails v3 is composed of four layers:

```
┌─────────────────────────────────────────────┐
│               CLI & Tooling                 │
├─────────────────────────────────────────────┤
│            Bridging Layer                   │
├──────────────────┬──────────────────────────┤
│   Go Backend     │    Web Frontend          │
└──────────────────┴──────────────────────────┘
```

### Go Backend

The Go backend runs your application code, manages the application lifecycle,
and exposes functions to the frontend through the binding system. It also hosts
services (system tray, menus, events, etc.) and handles OS-level interactions.

### Web Frontend

The frontend is a standard web application (React, Vue, Svelte, etc.) that
runs inside the native webview. It communicates with the Go backend through
the binding layer and uses the Wails runtime library to access platform
features.

### Bridging Layer

The bridging layer connects Go and JavaScript. It handles:

- Function binding (Go → JS)
- Event dispatching (Go ↔ JS)
- Runtime method exposure (JS → Go)
- Type serialization and marshaling

### CLI & Tooling

The `wails3` CLI provides commands for creating, building, and developing
applications. It delegates build tasks to Task and interacts with the Wails
runtime libraries.

## End-to-End Flow

1. User runs `wails3 dev` or `wails3 build`
2. The CLI parses the command and loads the Taskfile
3. Frontend assets are built (Vite/npm)
4. Go code is compiled with build tags and injected metadata
5. The application starts — the Go backend initializes services and the webview
6. The frontend loads and connects to the backend through the binding layer
7. In dev mode, file watchers trigger rebuilds and hot reloads

## What This Documentation Covers

The technical documentation is organized into the following sections:

| Section                        | Description                                      |
| ------------------------------ | ------------------------------------------------ |
| **Codebase Layout**            | Repository structure and module organization      |
| **Runtime Internals**          | Application lifecycle and runtime system          |
| **Asset & Dev Server**         | Frontend serving, hot reload, and Vite integration |
| **Build & Packaging Pipeline** | Compilation, platform packaging, and distribution |
| **Binding System**             | Go ↔ JavaScript function binding and type mapping  |
| **Template System**            | Project templates and scaffolding                 |
| **Testing & CI**               | Test infrastructure and continuous integration    |
| **Extending Wails**            | Writing plugins and custom services               |

Each section covers a specific subsystem in detail with code references and
internal design notes.
