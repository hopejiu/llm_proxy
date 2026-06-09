---
title: Enums
description: Auto-generated TypeScript enums and JavaScript constants from Go constant types.
sidebar_position: 3
---

# Enums

Wails automatically detects Go constant types and generates corresponding TypeScript enums or JavaScript constant objects. No registration is needed — enums are generated alongside method bindings.

## Quick Start

Define a Go enum type and its constants. They appear in the generated bindings.

```go title="enums.go"
type Status string

const (
    StatusActive   Status = "active"
    StatusInactive Status = "inactive"
    StatusPending  Status = "pending"
)
```

```typescript title="frontend/src/App.ts"
import { Status } from '../bindings/main';

const current: Status = Status.Active;
console.log(current); // "active"
```

## String Enums

```go
type LogLevel string

const (
    LogLevelDebug LogLevel = "debug"
    LogLevelInfo  LogLevel = "info"
    LogLevelWarn  LogLevel = "warn"
    LogLevelError LogLevel = "error"
)
```

Generates:

```typescript
enum LogLevel {
    Debug = "debug",
    Info = "info",
    Warn = "warn",
    Error = "error",
}
```

## Integer Enums

```go
type Priority int

const (
    PriorityLow    Priority = 0
    PriorityMedium Priority = 1
    PriorityHigh   Priority = 2
)
```

Generates:

```typescript
enum Priority {
    Low = 0,
    Medium = 1,
    High = 2,
}
```

## Type Alias Enums

When you use a type alias (rather than a named type), Wails generates a JavaScript const object instead of a TypeScript enum.

```go
type Direction = string

const (
    North Direction = "north"
    South Direction = "south"
    East  Direction = "east"
    West  Direction = "west"
)
```

Generates:

```typescript
const Direction = {
    North: "north",
    South: "south",
    East: "east",
    West: "west",
} as const;

type Direction = typeof Direction[keyof typeof Direction];
```

## Zero Values (`$zero`)

Every enum includes a special `$zero` value representing the zero value of the type. This is useful for detecting unset values.

```go
type Color string

const (
    ColorRed   Color = "red"
    ColorGreen Color = "green"
    ColorBlue  Color = "blue"
)
```

```typescript
enum Color {
    Red = "red",
    Green = "green",
    Blue = "blue",
    $zero = "",  // zero value for string type
}
```

Usage:

```typescript
const color: Color = Color.$zero;
if (color === Color.$zero) {
    console.log("No color selected");
}
```

For integer enums, `$zero` is `0`:

```typescript
enum Priority {
    Low = 0,
    Medium = 1,
    High = 2,
    $zero = 0,
}
```

## Comments and JSDoc

Comments on enum constants become JSDoc documentation.

```go
// Priority levels for task processing
type Priority int

const (
    // Lowest priority — processed last
    PriorityLow Priority = iota
    // Normal priority — processed in order
    PriorityMedium
    // Highest priority — processed first
    PriorityHigh
)
```

Generates:

```typescript
/** Priority levels for task processing */
enum Priority {
    /** Lowest priority — processed last */
    Low = 0,
    /** Normal priority — processed in order */
    Medium = 1,
    /** Highest priority — processed first */
    High = 2,
}
```

## Supported Underlying Types

| Go Underlying Type | Generated Type |
|--------------------|----------------|
| `string` | String enum |
| `int`, `int8`, `int16`, `int32`, `int64` | Integer enum |
| `uint`, `uint8`, `uint16`, `uint32`, `uint64` | Integer enum |

## Using Enums with Services

```go
type TaskService struct{}

type TaskStatus string

const (
    TaskStatusTodo       TaskStatus = "todo"
    TaskStatusInProgress TaskStatus = "in_progress"
    TaskStatusDone       TaskStatus = "done"
)

func (s *TaskService) CreateTask(title string) map[string]any {
    return map[string]any{
        "title":  title,
        "status": TaskStatusTodo,
    }
}

func (s *TaskService) UpdateStatus(id int, status TaskStatus) error {
    // update logic
    return nil
}
```

```typescript
import { TaskStatus } from '../bindings/main';

const task = await TaskService.CreateTask("My task");
console.log(task.status); // "todo"

await TaskService.UpdateStatus(task.id, TaskStatus.Done);
```

## Limitations

- **No generics**: Enum types using Go generics are not supported.
- **No custom marshalers**: Types with `MarshalJSON`/`UnmarshalJSON` methods are treated as their underlying type.
- **No `iota` values**: Only explicitly assigned constant values are detected.
- **Single package**: Enums must be defined in the same package as the service methods.

## See Also

- [Method Bindings](/docs/features/bindings/methods)
- [Data Models](/docs/features/bindings/models)
- [Advanced Bindings](/docs/features/bindings/advanced)
