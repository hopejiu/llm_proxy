---
title: How Wails Works
description: Understanding the Wails architecture and how it achieves native performance
---

# How Wails Works

Wails is a framework for building desktop applications using **Go for the backend** and **web technologies for the frontend**. But unlike Electron, Wails doesn't bundle a browser—it uses the **operating system's native WebView**.

**Key differences from Electron:**

| Aspect | Wails | Electron |
| --- | --- | --- |
| **Browser** | OS-provided WebView | Bundled Chromium (~100MB) |
| **Backend** | Go (compiled) | Node.js (interpreted) |
| **Communication** | In-memory bridge | IPC (inter-process) |
| **Bundle Size** | ~15MB | ~150MB |
| **Memory** | ~10MB | ~100MB+ |
| **Startup** | <0.5s | 2-3s |

## Core Components

### 1. Native WebView

Wails uses the operating system's built-in web rendering engine:

**Windows: WebView2** (Microsoft Edge WebView2)

- Based on Chromium (same as Edge browser)
- Pre-installed on Windows 10/11
- Automatic updates via Windows Update
- Full modern web standards support

**macOS: WebKit** (Safari's rendering engine)

- Built into macOS
- Same engine as Safari browser
- Excellent performance and battery life
- Full modern web standards support

**Linux: WebKitGTK** (GTK port of WebKit)

- Installed via package manager
- Same engine as GNOME Web (Epiphany)
- Good standards support
- Lightweight and performant

**Why this matters:**

- **No bundled browser** → Smaller app size
- **OS-native** → Better integration and performance
- **Auto-updates** → Security patches from OS updates
- **Familiar rendering** → Same as system browser

### 2. The Wails Bridge

The bridge is the heart of Wails—it enables **direct communication** between Go and JavaScript.

**How it works:**

1. **Frontend calls a Go method** (via auto-generated binding)
2. **Bridge encodes the call** to JSON (method name + arguments)
3. **Router finds the Go method** in registered services
4. **Go method executes** and returns a value
5. **Bridge decodes the result** and sends back to frontend
6. **Promise resolves** in JavaScript with the result

**Performance characteristics:**

- **In-memory**: No network overhead, no HTTP
- **Zero-copy** where possible (for large data)
- **Async by default**: Non-blocking on both sides
- **Type-safe**: TypeScript definitions auto-generated

### 3. Service System

Services are the recommended way to expose Go functionality to the frontend.

```go
// Define a service (just a regular Go struct)
type GreetService struct {
    prefix string
}

// Methods with exported names are automatically available
func (g *GreetService) Greet(name string) string {
    return g.prefix + name + "!"
}

func (g *GreetService) GetTime() time.Time {
    return time.Now()
}

// Register the service
app := application.New(application.Options{
    Services: []application.Service{
        application.NewService(&GreetService{prefix: "Hello, "}),
    },
})
```

**Service discovery:**

- Wails **scans your struct** at startup
- **Exported methods** become callable from frontend
- **Type information** is extracted for TypeScript bindings
- **Error handling** is automatic (Go errors → JS exceptions)

**Generated TypeScript binding:**

```typescript
// Auto-generated in frontend/bindings/GreetService.ts
export function Greet(name: string): Promise<string>
export function GetTime(): Promise<Date>
```

**Why services?**

- **Type-safe**: Full TypeScript support
- **Auto-discovery**: No manual registration of methods
- **Organised**: Group related functionality
- **Testable**: Services are just Go structs

[Learn more about services →](https://v3.wails.io/features/bindings/services)

### 4. Event System

Events enable **pub/sub communication** between components.

**Use cases:**

- **Window communication**: One window notifies others
- **Background tasks**: Go service notifies UI of progress
- **State synchronisation**: Keep multiple windows in sync
- **Loose coupling**: Components don't need direct references

**Example:**

```go
// Go: Emit an event
app.Event.Emit("user-logged-in", user)
```

```javascript
// JavaScript: Listen for event
import { Events } from '@wailsio/runtime'

Events.On('user-logged-in', (user) => {
    console.log('User logged in:', user)
})
```

[Learn more about events →](https://v3.wails.io/features/events/system)

## Application Lifecycle

Understanding the lifecycle helps you know when to initialise resources and clean up.

**Lifecycle hooks:**

```go
app := application.New(application.Options{
    Name: "My App",

    // Cleanly intercept quit requests (e.g. unsaved changes).
    ShouldQuit: func() bool { return true },

    // Called when the app is confirmed to be quitting — save state, close connections, etc.
    OnShutdown: func() {},
})
```

There is no `OnStartup` field on `application.Options`. Startup work belongs in a service's `ServiceStartup(ctx, options)`, in a callback registered via `app.Event.OnApplicationEvent(events.Common.ApplicationStarted, ...)`, or simply before `app.Run()`.

[Learn more about lifecycle →](https://v3.wails.io/concepts/lifecycle)

## Build Process

Understanding how Wails builds your application:

**Build steps:**

1. **Analyse Go code**
   - Scan services for exported methods
   - Extract parameter and return types
   - Generate method signatures
2. **Generate TypeScript bindings**
   - Create `.ts` files for each service
   - Include full type definitions
   - Add JSDoc comments
3. **Build frontend**
   - Run your bundler (Vite, webpack, etc.)
   - Minify and optimise
   - Output to `frontend/dist/`
4. **Compile Go**
   - Compile with optimisations (`-ldflags="-s -w"`)
   - Include build metadata
   - Platform-specific compilation
5. **Embed assets**
   - Embed frontend files into Go binary
   - Compress assets
   - Create single executable

**Result:** A single native executable with everything embedded.

[Learn more about building →](https://v3.wails.io/guides/build/building)

## Development vs Production

Wails behaves differently in development and production:

**Development (wails3 dev):**

- **Hot reload**: Frontend changes reload instantly
- **Source maps**: Debug with original source
- **DevTools**: Browser DevTools available
- **Logging**: Verbose logging enabled
- **External frontend**: Served from dev server (Vite)

**Benefits:**

- Instant feedback on changes
- Full debugging capabilities
- Faster iteration

**Production (wails3 build):**

- **Embedded assets**: Frontend built into binary
- **Optimised**: Minified, compressed
- **No DevTools**: Disabled by default
- **Minimal logging**: Errors only
- **Single file**: Everything in one executable

**Benefits:**

- Single file distribution
- Smaller size (minified)
- Better performance
- No external dependencies

## Memory Model

Understanding memory usage helps you build efficient applications.

**Memory regions:**

1. **Go Heap**
   - Your services and application state
   - Managed by Go garbage collector
   - Typically 5-10MB for simple apps
2. **WebView Memory**
   - DOM, JavaScript heap, CSS
   - Managed by WebView's engine
   - Typically 10-20MB for simple apps
3. **Bridge Memory**
   - Message buffers for communication
   - Minimal overhead (<1MB)
   - Zero-copy for large data where possible

**Optimisation tips:**

- **Avoid large data transfers**: Pass IDs, fetch details on demand
- **Use events for updates**: Don't poll from frontend
- **Stream large files**: Don't load entirely into memory
- **Clean up listeners**: Remove event listeners when done

[Learn more about performance →](https://v3.wails.io/guides/advanced/performance)

## Security Model

Wails provides a secure-by-default architecture:

**Security features:**

1. **Method whitelisting**
   - Only exported methods are callable
   - Private methods are inaccessible
   - Explicit service registration required
2. **Type validation**
   - Arguments checked against Go types
   - Invalid types rejected
   - Prevents injection attacks
3. **No eval()**
   - Frontend can't execute arbitrary Go code
   - Only predefined methods callable
   - No dynamic code execution
4. **Context isolation**
   - Each window has its own context
   - Services can check caller context
   - Permissions per window possible

**Best practices:**

- **Validate user input** in Go (don't trust frontend)
- **Use context** for authentication/authorisation
- **Sanitise file paths** before file operations
- **Rate limit** expensive operations

[Learn more about security →](https://v3.wails.io/guides/advanced/security)

## Next Steps

- **Application Lifecycle** - Understand startup, shutdown, and lifecycle hooks
  [Learn More →](https://v3.wails.io/concepts/lifecycle)
- **Go-Frontend Bridge** - Deep dive into how the bridge works
  [Learn More →](https://v3.wails.io/concepts/bridge)
- **Build System** - Understand how Wails builds your application
  [Learn More →](https://v3.wails.io/concepts/build-system)
- **Start Building** - Apply what you've learned in a tutorial
  [Tutorials →](https://v3.wails.io/tutorials/03-notes-vanilla)

---

**Questions about architecture?** Ask in [Discord](https://discord.gg/JDdSxwjhGf) or check the [API reference](https://v3.wails.io/reference/overview).
