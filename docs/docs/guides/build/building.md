---
title: "Building Applications"
description: "How to build, develop, and package Wails applications."
---

## Build System

Wails uses [Task](https://taskfile.dev/) as its build system. All build
commands are invoked through the `wails3` CLI or directly via `task`.

## Building for the Current Platform

```bash
wails3 build
```

Produces a binary in the `bin/` directory for your current OS and architecture.

## Cross-Platform Builds

Use `GOOS` and `GOARCH` to target a different platform:

```bash
GOOS=linux GOARCH=amd64 wails3 build
GOOS=windows GOARCH=amd64 wails3 build
GOOS=darwin GOARCH=arm64 wails3 build
```

## Development Mode

Start a development server with hot reload:

```bash
wails3 dev
```

Features:
- **Hot reload** — frontend changes are reflected immediately.
- **Custom port** — specify with `--port`:

  ```bash
  wails3 dev --port 8080
  ```

- **HTTPS** — enable with `--https`:

  ```bash
  wails3 dev --https
  ```

## Packaging

Create a distributable package (installer, app bundle, etc.):

```bash
wails3 package
```

This produces a platform-specific artifact in `bin/`:
- **macOS** — `.app` bundle
- **Windows** — installer (NSIS)
- **Linux** — `.deb`, `.rpm`, or `.AppImage`

## Build Tags

Control optional features at compile time with Go build tags:

```bash
wails3 build -tags gtk3
wails3 build -tags server
```

## Using Task Directly

If you prefer calling Task without the `wails3` wrapper:

```bash
task build
task dev
task package
```

## Generating Assets

Generate platform icons and other assets from a single source image:

```bash
wails3 generate icons
```

Place your source icon at `build/appicon.png` (at least 512×512 pixels). The
command generates optimized icons for every target platform under `build/`.
