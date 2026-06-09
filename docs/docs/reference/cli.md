---
title: "CLI Reference"
description: "Complete reference for the Wails CLI commands"
---

## Overview

The Wails CLI (`wails3`) is the command-line entry point for creating, developing, building, signing, packaging, and inspecting Wails 3 applications. Most build orchestration is delegated to per-project Taskfiles (under `build/` in your project).

## Project lifecycle

| Command              | Description |
| -------------------- | ----------- |
| wails3 init          | Create a new project from a template. |
| wails3 dev           | Run the application in development mode with frontend hot reload. |
| wails3 build         | Build the project. |
| wails3 package       | Run the platform-specific package Taskfile task. |
| wails3 task [name]   | Run any Taskfile task; with no name, --list shows every registered task. |
| wails3 doctor        | Print a diagnostic report of your environment. |
| wails3 doctor-ng     | Newer TUI variant of doctor. |
| wails3 version       | Print the CLI version. |
| wails3 releasenotes  | Print recent release notes. |
| wails3 docs          | Open the docs site in your browser. |
| wails3 sponsor       | Open the sponsor page. |

## Generation

`wails3 generate <subcommand>`:

| Subcommand                    | Description |
| ----------------------------- | ----------- |
| generate bindings             | Generate Go-to-frontend bindings. |
| generate icons                | Convert a source PNG into the platform icon formats. |
| generate build-assets         | Generate the build/ directory contents from build/config.yml. |
| generate runtime              | Regenerate the pre-built /wails/runtime.js. |
| generate syso                 | Generate the Windows .syso resource file. |
| generate webview2bootstrapper | Generate a WebView2 bootstrap installer for Windows. |
| generate constants            | Generate JS event-name constants from Go event types. |
| generate template             | Scaffold a new project template. |
| generate .desktop             | Generate a Linux .desktop file. |
| generate appimage             | Generate the AppImage build directory. |

## Update

`wails3 update <subcommand>`:

| Subcommand          | Description |
| ------------------- | ----------- |
| update build-assets | Refresh the build/ directory from build/config.yml. |
| update cli          | Self-update the wails3 binary. |

## Code signing & packaging

| Command                   | Description |
| ------------------------- | ----------- |
| wails3 setup signing      | Interactive wizard that configures signing for the platforms it detects in build/. |
| wails3 setup entitlements | Interactive wizard for macOS entitlements. |
| wails3 sign [GOOS=...]    | Wrapper that runs the platform-specific *:sign Taskfile task. |
| wails3 tool sign          | Low-level direct signing entry point. |

## Tools

`wails3 tool <subcommand>`:

| Subcommand        | Description |
| ----------------- | ----------- |
| tool checkport    | Check whether a TCP port is open. |
| tool watcher      | Run a command whenever watched files change. |
| tool cp           | Cross-platform file copy. |
| tool buildinfo    | Print the embedded Go build info from a binary. |
| tool package      | Build a Linux package from build/linux/nfpm. |
| tool version      | Bump a project's semantic version. |
| tool lipo         | Combine multiple macOS architecture binaries into a universal binary. |
| tool capabilities | Probe the system for GTK3/GTK4 / WebKit availability. |
| tool sign         | Low-level signing entry point. |

## Services

`wails3 service <subcommand>`:

| Subcommand   | Description |
| ------------ | ----------- |
| service init | Scaffold a new service package. |

## iOS

`wails3 ios <subcommand>`:

| Subcommand      | Description |
| --------------- | ----------- |
| ios overlay:gen | Generate the Go overlay for the iOS bridge shim. |
| ios xcode:gen   | Generate an Xcode project in the output directory. |

## Build output paths

* Native binaries land in `bin/<APP_NAME>` (or `bin/<APP_NAME>.exe` on Windows).
* Packaged outputs (`.app`, `.dmg`, NSIS installer, MSIX, DEB/RPM/AppImage) likewise land in `bin/`.

## Global flags

| Flag         | Applies to   | Description                        |
| ------------ | ------------ | ---------------------------------- |
| --no-colour | All commands | Disable ANSI colour in CLI output. |
