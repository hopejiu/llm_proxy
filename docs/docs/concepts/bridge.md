---
title: Go-Frontend Bridge
description: Deep dive into how Wails enables direct communication between Go and JavaScript
---

# Go-Frontend Bridge

## Direct Go-JavaScript Communication

Wails provides a **direct, in-memory bridge** between Go and JavaScript, enabling seamless communication without HTTP overhead, process boundaries, or serialisation bottlenecks.

## The Big Picture

**Key insight:** No HTTP, no IPC, no process boundaries. Just **direct function calls** with **type safety**.

## How It Works: Step by Step

### 1. Service Registration (Startup)

When your application starts, Wails scans your services:

```go
type GreetService struct {
    prefix string
}

func (g *GreetService) Greet(name string) string {
    return g.prefix + name + "!"
}

func (g *GreetService) Add(a, b int) int {
    return a + b
}

// Register service
app := application.New(application.Options{
    Services: []application.Service{
        application.NewService(&GreetService{prefix: "Hello, "}),
    },
})
```

**What Wails does:**

1. **Scans the struct** for exported methods
2. **Extracts type information** (parameters, return types)
3. **Builds a registry** mapping method names to functions
4. **Generates TypeScript bindings** with full type definitions

### 2. Binding Generation (Build Time)

Wails generates TypeScript bindings automatically:

```typescript
// frontend/bindings/GreetService.ts
export function Greet(name: string): Promise<string>
export function Add(a: number, b: number): Promise<number>
```

**Type mapping:**

| Go Type | TypeScript Type |
| --- | --- |
| string | string |
| int, int32, int64 | number |
| float32, float64 | number |
| bool | boolean |
| []T | T[] |
| map[string]T | Record<string, T> |
| struct | interface |
| time.Time | Date |
| error | Exception (thrown) |

### 3. Frontend Call (Runtime)

Developer calls the Go method from JavaScript:

```javascript
import { Greet, Add } from './bindings/GreetService'

// Call Go from JavaScript
const greeting = await Greet("World")
console.log(greeting)  // "Hello, World!"

const sum = await Add(5, 3)
console.log(sum)  // 8
```

**What happens:**

1. **Binding function called** - `Greet("World")`
2. **Message created** - `{ service: "GreetService", method: "Greet", args: ["World"] }`
3. **Sent to bridge** - Via WebView's JavaScript bridge
4. **Promise returned** - Awaits response

### 4. Bridge Processing (Runtime)

The bridge receives the message and processes it:

**Security:** Only registered services and exported methods are callable.

### 5. Go Execution (Runtime)

The Go method executes:

```go
func (g *GreetService) Greet(name string) string {
    // This runs in Go
    return g.prefix + name + "!"
}
```

**Execution context:**

- Runs in a **goroutine** (non-blocking)
- Has access to **all Go features** (file system, network, databases)
- Can call **other Go code** freely
- Returns result or error

### 6. Response (Runtime)

Result is sent back to JavaScript:

```javascript
// Promise resolves with result
const greeting = await Greet("World")
// greeting = "Hello, World!"
```

**Error handling:**

```go
func (g *GreetService) Divide(a, b float64) (float64, error) {
    if b == 0 {
        return 0, errors.New("division by zero")
    }
    return a / b, nil
}
```

```javascript
try {
    const result = await Divide(10, 0)
} catch (error) {
    console.error("Go error:", error)  // "division by zero"
}
```

## Performance Characteristics

### Speed

**Typical call overhead:** <1ms

```
Frontend Call → Bridge → Go Execution → Bridge → Frontend Response
    ↓            ↓           ↓            ↓            ↓
  <0.1ms      <0.1ms      [varies]     <0.1ms      <0.1ms
```

**Compared to alternatives:**

- **HTTP/REST:** 5-50ms (network stack, serialisation)
- **IPC:** 1-10ms (process boundaries, marshalling)
- **Wails Bridge:** <1ms (in-memory, direct call)

### Memory

**Per-call overhead:** ~1KB (message buffer)

**Zero-copy optimisation:** Large data (>1MB) uses shared memory where possible.

### Concurrency

**Calls are concurrent:**

- Each call runs in its own goroutine
- Multiple calls can execute simultaneously
- No blocking between calls

```javascript
// These run concurrently
const [result1, result2, result3] = await Promise.all([
    SlowOperation1(),
    SlowOperation2(),
    SlowOperation3(),
])
```

## Type System

### Supported Types

#### Primitives

```go
// Go
func Example(
    s string,
    i int,
    f float64,
    b bool,
) (string, int, float64, bool) {
    return s, i, f, b
}
```

```typescript
// TypeScript (auto-generated)
function Example(
    s: string,
    i: number,
    f: number,
    b: boolean,
): Promise<[string, number, number, boolean]>
```

#### Slices and Arrays

```go
// Go
func Sum(numbers []int) int {
    total := 0
    for _, n := range numbers {
        total += n
    }
    return total
}
```

```typescript
// TypeScript
function Sum(numbers: number[]): Promise<number>

// Usage
const total = await Sum([1, 2, 3, 4, 5])  // 15
```

#### Maps

```go
// Go
func GetConfig() map[string]interface{} {
    return map[string]interface{}{
        "theme": "dark",
        "fontSize": 14,
        "enabled": true,
    }
}
```

```typescript
// TypeScript
function GetConfig(): Promise<Record<string, any>>

// Usage
const config = await GetConfig()
console.log(config.theme)  // "dark"
```

#### Structs

```go
// Go
type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func GetUser(id int) (*User, error) {
    return &User{
        ID:    id,
        Name:  "Alice",
        Email: "alice@example.com",
    }, nil
}
```

```typescript
// TypeScript (auto-generated)
interface User {
    id: number
    name: string
    email: string
}

function GetUser(id: number): Promise<User>

// Usage
const user = await GetUser(1)
console.log(user.name)  // "Alice"
```

**JSON tags:** Use `json:` tags to control field names in TypeScript.

#### Time

```go
// Go
func GetTimestamp() time.Time {
    return time.Now()
}
```

```typescript
// TypeScript
function GetTimestamp(): Promise<Date>

// Usage
const timestamp = await GetTimestamp()
console.log(timestamp.toISOString())
```

#### Errors

```go
// Go
func Validate(input string) error {
    if input == "" {
        return errors.New("input cannot be empty")
    }
    return nil
}
```

```typescript
// TypeScript
function Validate(input: string): Promise<void>

// Usage
try {
    await Validate("")
} catch (error) {
    console.error(error)  // "input cannot be empty"
}
```

### Unsupported Types

These types **cannot** be passed across the bridge:

- **Channels** (`chan T`)
- **Functions** (`func()`)
- **Interfaces** (except `interface{}` / `any`)
- **Pointers** (except to structs)
- **Unexported fields** (lowercase)

**Workaround:** Use IDs or handles:

```go
// ❌ Can't pass file handle
func OpenFile(path string) (*os.File, error) {
    return os.Open(path)
}

// ✅ Return file ID instead
var files = make(map[string]*os.File)

func OpenFile(path string) (string, error) {
    file, err := os.Open(path)
    if err != nil {
        return "", err
    }
    id := generateID()
    files[id] = file
    return id, nil
}

func ReadFile(id string) ([]byte, error) {
    file := files[id]
    return io.ReadAll(file)
}

func CloseFile(id string) error {
    file := files[id]
    delete(files, id)
    return file.Close()
}
```

## Advanced Patterns

### Context Passing

Services can access the call context:

```go
type UserService struct{}

func (s *UserService) GetCurrentUser(ctx context.Context) (*User, error) {
    // Access the calling window via the context value
    window, _ := ctx.Value(application.WindowKey).(application.Window)
    _ = window

    // Access the application
    app := application.Get()
    _ = app

    // Your logic
    return getCurrentUser(), nil
}
```

**Context provides:**

- Window that made the call
- Application instance
- Request metadata

### Streaming Data

For large data, use events instead of return values:

```go
func ProcessLargeFile(path string) error {
    file, err := os.Open(path)
    if err != nil {
        return err
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    lineNum := 0

    for scanner.Scan() {
        lineNum++
        // Emit progress events
        app.Event.Emit("file-progress", map[string]interface{}{
            "line": lineNum,
            "text": scanner.Text(),
        })
    }

    return scanner.Err()
}
```

```javascript
import { Events } from '@wailsio/runtime'
import { ProcessLargeFile } from './bindings/FileService'

// Listen for progress
Events.On('file-progress', (data) => {
    console.log(`Line ${data.line}: ${data.text}`)
})

// Start processing
await ProcessLargeFile('/path/to/large/file.txt')
```

### Cancellation

Use context for cancellable operations:

```go
func LongRunningTask(ctx context.Context) error {
    for i := 0; i < 1000; i++ {
        // Check if cancelled
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            // Continue work
            time.Sleep(100 * time.Millisecond)
        }
    }
    return nil
}
```

**Note:** Context cancellation on frontend disconnect is automatic.

### Batch Operations

Reduce bridge overhead by batching:

```go
// ❌ Inefficient: N bridge calls
for _, item := range items {
    await ProcessItem(item)
}

// ✅ Efficient: 1 bridge call
await ProcessItems(items)
```

```go
func ProcessItems(items []Item) ([]Result, error) {
    results := make([]Result, len(items))
    for i, item := range items {
        results[i] = processItem(item)
    }
    return results, nil
}
```

## Debugging the Bridge

### Enable Debug Logging

```go
app := application.New(application.Options{
    Name:     "My App",
    LogLevel: slog.LevelDebug,
})
```

**Output shows:**

- Method calls
- Parameters
- Return values
- Errors
- Timing information

### Inspect Generated Bindings

Check `frontend/bindings/` to see generated TypeScript:

```javascript
// frontend/bindings/<full-go-import-path>/myservice.js
import { Call as $Call } from "/wails/runtime.js";

export function MyMethod($0) {
    return $Call.ByID(1234567890, $0);
}
```

### Test Services Directly

Test Go services without the frontend:

```go
func TestGreetService(t *testing.T) {
    service := &GreetService{prefix: "Hello, "}
    result := service.Greet("Test")
    if result != "Hello, Test!" {
        t.Errorf("Expected 'Hello, Test!', got '%s'", result)
    }
}
```

## Performance Tips

### ✅ Do

- **Batch operations** - Reduce bridge calls
- **Use events for streaming** - Don't return large arrays
- **Keep methods fast** - <100ms ideal
- **Use goroutines** - For long operations
- **Cache on Go side** - Avoid repeated calculations

### ❌ Don't

- **Don't make excessive calls** - Batch when possible
- **Don't return huge data** - Use pagination or streaming
- **Don't block** - Use goroutines for long operations
- **Don't pass complex types** - Keep it simple
- **Don't ignore errors** - Always handle them

## Security

The bridge is secure by default:

1. **Whitelist only** - Only registered services callable
2. **Type validation** - Arguments checked against Go types
3. **No eval()** - Frontend can't execute arbitrary Go code
4. **No reflection abuse** - Only exported methods accessible

**Best practices:**

- **Validate input** in Go (don't trust frontend)
- **Use context** for authentication/authorisation
- **Rate limit** expensive operations
- **Sanitise** file paths and user input

## Next Steps

- **Build System** - Learn how Wails builds and bundles your application
  [Learn More →](https://v3.wails.io/concepts/build-system)
- **Services** - Deep dive into the service system
  [Learn More →](https://v3.wails.io/features/bindings/services)
- **Events** - Use events for pub/sub communication
  [Learn More →](https://v3.wails.io/features/events/system)

---

**Questions about the bridge?** Ask in [Discord](https://discord.gg/JDdSxwjhGf) or check the [binding examples](https://github.com/wailsapp/wails/tree/master/v3/examples/binding).
