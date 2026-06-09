---
title: Method Bindings
description: Type-safe Go-JavaScript method bindings for calling Go functions from your frontend.
sidebar_position: 1
---

# Method Bindings

Wails generates type-safe JavaScript/TypeScript bindings for your Go methods, allowing you to call Go functions directly from your frontend code with full type safety and automatic serialisation.

## Quick Start

1. Write a Go service or struct with exported methods.
2. Register it with `application.NewService()`.
3. Generate bindings with `wails3 generate bindings`.
4. Import and use in JavaScript/TypeScript.

```go title="main.go"
package main

import (
    "github.com/wailsapp/wails/v3/pkg/application"
)

type GreetingService struct{}

func (g *GreetingService) Greet(name string) string {
    return "Hello, " + name + "!"
}

func main() {
    app := application.New(application.Options{})
    app.NewService(&GreetingService{})
    app.Run()
}
```

```typescript title="frontend/src/App.ts"
import { GreetingService } from '../bindings/main';

const greeting = await GreetingService.Greet("World");
console.log(greeting); // "Hello, World!"
```

## Registering Services

```go
// Register a struct
app.NewService(&MyService{})

// Register with a custom name
app.NewService(&MyService{}, application.ServiceOptions{
    Name: "customName",
})
```

## Type Mapping

### Primitives

| Go Type | JavaScript/TypeScript Type |
|---------|---------------------------|
| `string` | `string` |
| `bool` | `boolean` |
| `int`, `int8`, `int16`, `int32`, `int64` | `number` |
| `uint`, `uint8`, `uint16`, `uint32`, `uint64` | `number` |
| `float32`, `float64` | `number` |
| `byte` | `number` |
| `complex64`, `complex128` | `[number, number]` |

### Complex Types

| Go Type | JavaScript/TypeScript Type |
|---------|---------------------------|
| `[]T` | `T[]` |
| `[N]T` | `T[]` |
| `map[K]V` | `Record<K, V>` |
| `struct` | Generated class (see [Models](/docs/features/bindings/models)) |
| `*T` | `T \| null` |
| `interface{}` | `any` |
| `time.Time` | `string` (ISO 8601) |
| `time.Duration` | `number` (nanoseconds) |

### Unsupported Types

The following types cannot be bound:

- `func` types
- `chan` types
- `complex` types (in struct fields)
- Generics (parameterised types)

## Error Handling

Go methods that return an error will have the error propagated to JavaScript as a rejected promise.

```go
func (s *MyService) Save(data string) error {
    if data == "" {
        return fmt.Errorf("data cannot be empty")
    }
    // save logic
    return nil
}
```

```typescript
try {
    await MyService.Save("data");
} catch (error) {
    console.error("Save failed:", error);
}
```

## Multiple Return Values

```go
func (s *MyService) Divide(a, b float64) (float64, error) {
    if b == 0 {
        return 0, fmt.Errorf("division by zero")
    }
    return a / b, nil
}
```

```typescript
try {
    const result = await MyService.Divide(10, 3);
    console.log(result); // 3.333...
} catch (error) {
    console.error(error);
}
```

## Performance

Method bindings use efficient serialisation with typical call times under 1ms for small payloads. For large data transfers, consider:

- Batching multiple calls
- Using streaming for large datasets
- Minimising data serialisation overhead

## Complete Example

```go title="service.go"
package main

import (
    "fmt"
    "sync"
)

type TodoService struct {
    mu    sync.RWMutex
    todos []Todo
    nextID int
}

type Todo struct {
    ID        int    `json:"id"`
    Title     string `json:"title"`
    Completed bool   `json:"completed"`
}

func NewTodoService() *TodoService {
    return &TodoService{nextID: 1}
}

func (s *TodoService) Add(title string) Todo {
    s.mu.Lock()
    defer s.mu.Unlock()
    todo := Todo{ID: s.nextID, Title: title}
    s.nextID++
    s.todos = append(s.todos, todo)
    return todo
}

func (s *TodoService) List() []Todo {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.todos
}

func (s *TodoService) Complete(id int) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    for i := range s.todos {
        if s.todos[i].ID == id {
            s.todos[i].Completed = true
            return nil
        }
    }
    return fmt.Errorf("todo %d not found", id)
}

func (s *TodoService) Delete(id int) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    for i := range s.todos {
        if s.todos[i].ID == id {
            s.todos = append(s.todos[:i], s.todos[i+1:]...)
            return nil
        }
    }
    return fmt.Errorf("todo %d not found", id)
}
```

```go title="main.go"
func main() {
    app := application.New(application.Options{})
    app.NewService(NewTodoService{})
    app.Run()
}
```

```typescript title="frontend/src/App.ts"
import { TodoService } from '../bindings/main';

async function demo() {
    const todo1 = await TodoService.Add("Buy groceries");
    const todo2 = await TodoService.Add("Write documentation");

    await TodoService.Complete(todo1.id);

    const allTodos = await TodoService.List();
    console.log(allTodos);
    // [{id: 1, title: "Buy groceries", completed: true},
    //  {id: 2, title: "Write documentation", completed: false}]
}
```

## See Also

- [Data Models](/docs/features/bindings/models)
- [Enums](/docs/features/bindings/enums)
- [Advanced Bindings](/docs/features/bindings/advanced)
