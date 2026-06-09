---
title: "Services"
description: "Organizing Go code with services, lifecycle management, and dependency injection."
---

Services provide a structured way to organize your Go backend code. Each
service encapsulates a specific domain of functionality and is registered with
the application through the service manager.

## Creating a Service

Define a service as a struct with methods that will be exposed to the frontend.

```go
type Greeter struct {
    name string
}

func NewGreeter(name string) *Greeter {
    return &Greeter{name: name}
}

func (g *Greeter) Greet(name string) string {
    return fmt.Sprintf("Hello %s, my name is %s!", name, g.name)
}
```

## Registering Services

Register services with the application using `app.NewService()`.

```go
func main() {
    app := application.New(application.Options{
        Title: "My App",
    })

    greeter := NewGreeter("Wails")
    app.NewService(greeter)

    err := app.Run()
    if err != nil {
        log.Fatal(err)
    }
}
```

## Service Lifecycle

Implement lifecycle hooks to manage initialization and cleanup.

```go
type DatabaseService struct {
    db     *sql.DB
    connStr string
}

func NewDatabaseService(connStr string) *DatabaseService {
    return &DatabaseService{connStr: connStr}
}

func (s *DatabaseService) OnStartup(ctx context.Context) error {
    var err error
    s.db, err = sql.Open("postgres", s.connStr)
    if err != nil {
        return fmt.Errorf("failed to open database: %w", err)
    }
    return s.db.PingContext(ctx)
}

func (s *DatabaseService) OnBeforeClose(ctx context.Context) bool {
    if err := s.db.Close(); err != nil {
        log.Printf("error closing database: %v", err)
        return false
    }
    return true
}
```

Available lifecycle hooks:

| Hook | When it runs |
|------|-------------|
| `OnStartup` | When the application starts, before the window is shown |
| `OnBeforeClose` | When the user requests to close the window. Return `false` to prevent closing |
| `OnShutdown` | When the application is shutting down |
| `OnSecondInstanceLaunch` | When a second instance of the application is launched |

## Dependency Injection

Pass dependencies to services through constructor parameters. This keeps
services decoupled and testable.

```go
type UserService struct {
    db     *DatabaseService
    logger *LogService
}

func NewUserService(db *DatabaseService, logger *LogService) *UserService {
    return &UserService{db: db, logger: logger}
}

func (s *UserService) GetUser(id int64) (*User, error) {
    s.logger.Info("Fetching user", id)
    return s.db.FindUser(id)
}

// Registration order matters:
func main() {
    app := application.New(application.Options{
        Title: "My App",
    })

    logService := NewLogService()
    app.NewService(logService)

    dbService := NewDatabaseService("postgres://localhost/mydb")
    app.NewService(dbService)

    userService := NewUserService(dbService, logService)
    app.NewService(userService)

    app.Run()
}
```

## Thread Safety

Services are shared between the main thread and goroutines. Protect shared
state with mutexes.

```go
type CacheService struct {
    mu    sync.RWMutex
    items map[string]*Item
}

func NewCacheService() *CacheService {
    return &CacheService{
        items: make(map[string]*Item),
    }
}

func (s *CacheService) Get(key string) (*Item, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    item, ok := s.items[key]
    return item, ok
}

func (s *CacheService) Set(key string, item *Item) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.items[key] = item
}

func (s *CacheService) Delete(key string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    delete(s.items, key)
}
```

For read-heavy workloads, prefer `sync.RWMutex` over `sync.Mutex` to allow
concurrent reads.

## Complete Example

A multi-service application with a file service, notification service, and
a combined app service.

```go
// services/fileservice.go
type FileService struct {
    baseDir string
    mu      sync.RWMutex
    watchers map[string]func(string)
}

func NewFileService(baseDir string) *FileService {
    return &FileService{
        baseDir:  baseDir,
        watchers: make(map[string]func(string)),
    }
}

func (s *FileService) ReadFile(path string) (string, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    clean := filepath.Clean(path)
    fullPath := filepath.Join(s.baseDir, clean)

    data, err := os.ReadFile(fullPath)
    if err != nil {
        return "", err
    }
    return string(data), nil
}

func (s *FileService) WriteFile(path string, content string) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    clean := filepath.Clean(path)
    fullPath := filepath.Join(s.baseDir, clean)

    if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
        return err
    }

    if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
        return err
    }

    s.notifyWatchers(path)
    return nil
}

func (s *FileService) OnWatch(path string, callback func(string)) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.watchers[path] = callback
}

func (s *FileService) notifyWatchers(path string) {
    if cb, ok := s.watchers[path]; ok {
        cb(path)
    }
}

// services/notificationservice.go
type NotificationService struct {
    mu           sync.Mutex
    notifications []Notification
}

type Notification struct {
    ID      int       `json:"id"`
    Title   string    `json:"title"`
    Message string    `json:"message"`
    Read    bool      `json:"read"`
    Time    time.Time `json:"time"`
}

func NewNotificationService() *NotificationService {
    return &NotificationService{}
}

func (s *NotificationService) Send(title, message string) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    n := Notification{
        ID:      len(s.notifications) + 1,
        Title:   title,
        Message: message,
        Read:    false,
        Time:    time.Now(),
    }
    s.notifications = append(s.notifications, n)
    events.Emit("notification:new", n)
    return nil
}

func (s *NotificationService) GetAll() []Notification {
    s.mu.Lock()
    defer s.mu.Unlock()

    result := make([]Notification, len(s.notifications))
    copy(result, s.notifications)
    return result
}

func (s *NotificationService) MarkRead(id int) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    for i := range s.notifications {
        if s.notifications[i].ID == id {
            s.notifications[i].Read = true
            return nil
        }
    }
    return fmt.Errorf("notification %d not found", id)
}

// main.go
func main() {
    app := application.New(application.Options{
        Title: "Multi-Service App",
    })

    fileService := NewFileService("./data")
    app.NewService(fileService)

    notifService := NewNotificationService()
    app.NewService(notifService)

    app.Run()
}
```

Frontend usage:

```javascript
import { ReadFile, WriteFile, GetAll, Send, MarkRead } from '../wailsjs/go/main';

// Read a file
const content = await ReadFile("config.json");

// Write a file
await WriteFile("output.txt", "Hello from Wails");

// Get all notifications
const notifications = await GetAll();

// Send a notification
await Send("Update Available", "Version 2.0 is ready to install");

// Mark a notification as read
await MarkRead(1);
```
