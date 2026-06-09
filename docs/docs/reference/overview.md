---
title: "API Reference Overview"
description: "Conventions, package structure, and general guidance for the Wails API reference."
---

## About This Reference

This section documents the public APIs exposed by Wails. Each package page lists
exported types, functions, and methods with usage notes.

## Go API Conventions

| Convention          | Description                                                  |
|---------------------|--------------------------------------------------------------|
| **Naming**          | Short, idiomatic Go names. Interfaces are noun-shaped.       |
| **Error handling**  | Functions return `(result, error)`. Nil error means success.  |
| **Context**         | Long-running operations accept `context.Context` as the first argument. |
| **Options pattern** | Configuration is passed via `FooOptions` structs.            |

## JavaScript API Conventions

| Convention          | Description                                                  |
|---------------------|--------------------------------------------------------------|
| **Naming**          | camelCase for functions and variables.                       |
| **Async by default**| All bound functions return Promises. Use `await` or `.then()`. |
| **Error handling**  | Rejected promises carry error details.                       |
| **Type safety**     | TypeScript declarations are auto-generated in `bindings/`.   |

## Package Structure

```
github.com/wailsapp/wails/v3/pkg/
├── application     # Core application lifecycle
├── events          # Event bus
├── logger          # Structured logging
├── assets          # Embedded frontend assets
└── ...
```

## Import Paths

```go
import "github.com/wailsapp/wails/v3/pkg/application"
```

All packages are versioned under the `/v3` module path.

## Type Reference

Core types are documented in their respective package pages. Common types you
will encounter:

- `application.Options` — Application configuration.
- `application.WindowOptions` — Window creation parameters.
- `application.Event` — Event payload.
- `application.MacOptions` — macOS-specific settings.
- `application.WindowsOptions` — Windows-specific settings.
- `application.LinuxOptions` — Linux-specific settings.

## Platform Differences

| Feature            | macOS   | Windows | Linux   |
|--------------------|---------|---------|---------|
| System tray        | Yes     | Yes     | Yes     |
| Context menus      | Yes     | Yes     | Yes     |
| Title bar control  | Full    | Partial | Partial |
| GPU acceleration   | Yes     | Yes     | Yes     |
| Universal binary   | Yes     | —       | —       |
| Code signing       | Yes     | Yes     | —       |
| Notarization       | Yes     | —       | —       |

## Versioning

Wails follows [Semantic Versioning](https://semver.org/). Breaking changes
occur only at major version boundaries.

## Deprecation Policy

Deprecated APIs are marked with a `Deprecated:` comment and remain functional
for at least one minor version. Migrate away from deprecated APIs before the
next major release.

## API Stability

- **Stable** — Safe to use in production. Changes follow semver.
- **Unstable** — Experimental. May change without notice between minor versions.

## Getting Help

- **Documentation**: [wails.io/docs](https://wails.io/docs)
- **GitHub**: [github.com/wailsapp/wails](https://github.com/wailsapp/wails)
- **Discord**: [Wails Discord](https://discord.gg/jA668Yr)
- **Issues**: [GitHub Issues](https://github.com/wailsapp/wails/issues)
