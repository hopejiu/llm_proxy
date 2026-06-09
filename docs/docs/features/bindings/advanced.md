---
title: Advanced Binding
description: Customising generated code with directives, working with complex types, and optimising performance.
sidebar_position: 4
---

# Advanced Binding

Wails provides directives for fine-grained control over generated bindings, support for complex Go types, and options for optimising output size and performance.

## Directives

Directives are special comments placed above functions or types to control binding generation.

### `//wails:inject`

Inject custom code into the generated binding file. Useful for adding helper functions or custom runtime code.

```go
//wails:inject
// function customHelper(data: string) { return data.trim(); }

func (s *MyService) Process(input string) string {
    return input
}
```

The injected code appears at the top of the generated JavaScript/TypeScript file.

### `//wails:include`

Include additional files in the generated output. Useful for sharing type definitions or utility modules.

```go
//wails:include ../types/common.ts

func (s *MyService) GetData() CommonType {
    return CommonType{}
}
```

### `//wails:internal`

Mark a method as internal. Internal methods are included in the generated code but not exported in the public API.

```go
//wails:internal
func (s *MyService) helperMethod() string {
    return "internal"
}
```

### `//wails:ignore`

Skip a method entirely during binding generation.

```go
//wails:ignore
func (s *MyService) DebugInfo() string {
    return "debug data"
}
```

### `//wails:id`

Assign a custom method ID to control the generated method name.

```go
//wails:id fetchUserData
func (s *MyService) GetUserByID(id int) User {
    // ...
}
```

Generates `fetchUserData` instead of `getUserByID`.

## Working with Complex Types

### Nested Structs

```go
type Metadata struct {
    CreatedAt string `json:"createdAt"`
    UpdatedAt string `json:"updatedAt"`
    Tags      []string `json:"tags"`
}

type Document struct {
    ID       int      `json:"id"`
    Title    string   `json:"title"`
    Content  string   `json:"content"`
    Meta     Metadata `json:"meta"`
}
```

```typescript
const doc = await DocumentService.Get(1);
console.log(doc.meta.tags); // full type safety
```

### Maps with Complex Values

```go
func (s *CacheService) GetAll() map[string]CacheEntry {
    return map[string]CacheEntry{
        "user:1": {Value: "Alice", Expires: time.Now()},
        "user:2": {Value: "Bob", Expires: time.Now()},
    }
}
```

```typescript
const entries = await CacheService.GetAll();
// Record<string, CacheEntry>
Object.entries(entries).forEach(([key, entry]) => {
    console.log(key, entry.value, entry.expires);
});
```

### Slices of Structs

```go
func (s *ProductService) Search(query string) []Product {
    // ...
}
```

```typescript
const products = await ProductService.Search("widget");
products.forEach(p => console.log(p.name, p.price));
```

### Generics

Go generics have limited support. Methods using generic types fall back to `any` for type parameters.

```go
// This compiles but the binding uses `any`
func Transform[T any](input T) T {
    return input
}
```

For full type safety, use concrete types in exported methods.

### Interfaces

Interface parameters and return types are serialised as `any`. For type safety, prefer concrete struct types.

```go
// Accepts any implementation — serialised as `any` on JS side
func (s *Service) Process(handler Handler) error {
    return handler.Handle()
}
```

## Optimising Output

### `-names` Flag

Generate bindings with short method names instead of fully qualified names.

```bash
wails3 generate bindings -names
```

Before: `UserService.GetUserName`
After: `getUserName`

### `-b` Flag

Bundle the Wails runtime into the output file for self-contained distribution.

```bash
wails3 generate bindings -b
```

### `-noindex` Flag

Skip generating the index file that re-exports all bindings.

```bash
wails3 generate bindings -noindex
```

Useful when importing individual service bindings directly.

## Real-World Examples

### Event Emitter Pattern

```go
type EventService struct {
    listeners map[string][]func()
}

//wails:internal
func (s *EventService) emit(event string) {
    for _, fn := range s.listeners[event] {
        fn()
    }
}

func (s *EventService) On(event string, callback func()) {
    s.listeners[event] = append(s.listeners[event], callback)
}
```

### CRUD Service

```go
type TodoService struct {
    mu    sync.RWMutex
    items []Todo
    next  int
}

func (s *TodoService) Create(title string) Todo {
    s.mu.Lock()
    defer s.mu.Unlock()
    t := Todo{ID: s.next, Title: title, Done: false}
    s.next++
    s.items = append(s.items, t)
    return t
}

func (s *TodoService) List() []Todo {
    s.mu.RLock()
    defer s.mu.RUnlock()
    result := make([]Todo, len(s.items))
    copy(result, s.items)
    return result
}

func (s *TodoService) Update(id int, done bool) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    for i := range s.items {
        if s.items[i].ID == id {
            s.items[i].Done = done
            return nil
        }
    }
    return fmt.Errorf("todo %d not found", id)
}

func (s *TodoService) Delete(id int) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    for i := range s.items {
        if s.items[i].ID == id {
            s.items = append(s.items[:i], s.items[i+1:]...)
            return nil
        }
    }
    return fmt.Errorf("todo %d not found", id)
}
```

### Configuration Service with Validation

```go
type AppConfig struct {
    Theme    string `json:"theme"`
    Language string `json:"language"`
    FontSize int    `json:"fontSize"`
}

type ConfigService struct {
    config AppConfig
}

func (s *ConfigService) Get() AppConfig {
    return s.config
}

func (s *ConfigService) Update(cfg AppConfig) error {
    if cfg.FontSize < 8 || cfg.FontSize > 72 {
        return fmt.Errorf("font size must be between 8 and 72")
    }
    if cfg.Theme != "light" && cfg.Theme != "dark" {
        return fmt.Errorf("theme must be 'light' or 'dark'")
    }
    s.config = cfg
    return nil
}
```

```typescript
// Full type safety on the frontend
const config = await ConfigService.Get();
await ConfigService.Update({
    ...config,
    fontSize: 14,
});
```

## See Also

- [Method Bindings](/docs/features/bindings/methods)
- [Data Models](/docs/features/bindings/models)
- [Enums](/docs/features/bindings/enums)
