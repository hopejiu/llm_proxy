---
title: "Changelog"
description: "Version history and release notes for Wails v3"
---

Legend:  - macOS, ⊞ - Windows, 🐧 - Linux

## v3.0.0-alpha.96 - 2026-05-25

### Added
* Add Garble obfuscation support

## v3.0.0-alpha.95 - 2026-05-20

### Added
* Added missing project structure page

### Fixed
* Fix `wails3 generate appimage` on the GTK4 default
* Fix `events.Common.ApplicationStarted`, `Common.ThemeChanged`, `Common.SystemWillSleep` and `Common.SystemDidWake` not firing on Linux

## v3.0.0-alpha.93 - 2026-05-17

### Added
* Add `XDG_SESSION_TYPE` to `wails3 doctor` output on Linux

### Fixed
* Fix window menu crash on Wayland
* Fix "not enough memory" error on Windows drag and drop
* Fix race condition in mainthread callback store

## v3.0.0-alpha.91 - 2026-05-12

### Changed
* **BREAKING (macOS):** Normalise the macOS coordinate system

## v3.0.0-alpha.89 - 2026-05-10

### Changed
* Upgrade Vite from 5.x.x to 8.0.0 across all frontend templates

## v3.0.0-alpha.87 - 2026-05-07

### Added
* Add Korean, French, Portuguese documentation

## v3.0.0-alpha.81 - 2026-04-30

### Fixed
* Fix halved Screen Bounds, WorkArea, and Size values on Retina Macs

## Earlier releases

Key additions across earlier alphas include:
* Typed Events
* Multiple windows support
* System tray integration
* Improved bindings generation
* Taskfile-based build system
* Server mode (`-tags server`)
* WML (Wails Markup Language)
* Custom protocols
* File associations
* Single instance
* Autostart
* Updater framework
* GTK4 / WebKitGTK 6.0 support
* Obfuscated builds with Garble
