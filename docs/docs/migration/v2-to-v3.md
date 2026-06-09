---
title: Migrating from v2 to v3
description: Guide for migrating Wails v2 applications to Wails v3.
---

# Migrating from v2 to v3

This guide covers the key changes between Wails v2 and v3 and provides
step-by-step instructions for updating your application.

## Key Changes at a Glance

- **Build system** — Custom build logic replaced with [Task](https://taskfile.dev/)
- **Service system** — New service architecture for platform features
- **Updated APIs** — Simplified and more consistent Go and JS APIs
- **CLI changes** — Commands restructured (`wails3` prefix, Task delegation)

## Updated APIs

### Go Side

Review your imports and function signatures. The `wails3` package reorganized
several modules. Key changes include:

- Application lifecycle methods
- Context handling for frontend callbacks
- Updated event system APIs

### JavaScript Side

The `@wailsio/runtime` package replaces `wails/runtime` from v2. Update your
frontend imports:

```bash
# Remove v2 runtime
npm uninstall wails/runtime

# Install v3 runtime
npm install @wailsio/runtime
```

## New Service System

Wails v3 introduces a service-based architecture. Platform features like system
tray, menus, and clipboard are now services that you register with the
application:

```go
app := wails.CreateApp(&wails.AppConfig{
    // ... config
})

// Register services
app.Services(&service.Options{
    Tray: &tray.Options{
        // tray configuration
    },
})
```

Migrate your v2 platform feature usage to the new service registration pattern.

## Build System Changes

v2 used a custom Go-based build pipeline. v3 delegates to Task files.

| v2 Command           | v3 Equivalent          |
| -------------------- | ---------------------- |
| `wails build`        | `wails3 build`         |
| `wails dev`          | `wails3 dev`           |
| `wails build -upx`   | `wails3 build -upx`   |

The root `Taskfile.yml` and files under `build/` control the build process.
See the [Build Customization](../guides/build/customization.md) guide for
details.

## Breaking Changes

### Project Structure

The expected directory layout has changed:

- `build/` now contains Taskfiles and build config
- `frontend/` structure remains the same
- New `build/config.yml` for dev mode settings

### Configuration

Application configuration fields have been renamed and reorganized. Check the
`wails.AppConfig` struct for the current field names.

### Frontend Runtime

Event subscription and dispatch methods on the JS runtime have new signatures.
Review the runtime API reference for the current method names and parameters.

## Migration Checklist

1. Update Go imports to `v3` package paths
2. Replace `wails/runtime` with `@wailsio/runtime` in frontend code
3. Add `build/taskfiles/` directory with platform Taskfiles
4. Create `build/config.yml` with dev mode settings
5. Register platform features as services instead of configuring them in AppOptions
6. Update CI scripts to use `wails3` commands
7. Test on all target platforms

## Getting Help

If you encounter issues during migration, open a discussion on the
[Wails GitHub repository](https://github.com/wailsapp/wails/issues) with the
`v2-to-v3-migration` label.
