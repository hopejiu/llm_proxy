---
title: "Bindings Best Practices"
description: "Patterns and best practices for designing robust, performant, and secure Wails bindings."
---

## API Design Principles

### Single Responsibility

Each binding method should perform one clear action. Avoid methods that do
multiple unrelated things.

```go
// Good: each method has a single purpose
func (s *UserService) GetUser(id int64) (*User, error) {
    return s.db.FindUser(id)
}

func (s *UserService) UpdateEmail(id int64, email string) error {
    return s.db.UpdateUserEmail(id, email)
}

// Bad: method does too many things
func (s *UserService) GetUserAndUpdateEmail(id int64, email string) (*User, error) {
    user, err := s.db.FindUser(id)
    if err != nil {
        return nil, err
    }
    if email != "" {
        _ = s.db.UpdateUserEmail(id, email)
    }
    return user, nil
}
```

### Clear Method Names

Use descriptive names that communicate intent without requiring the caller to
read implementation details.

```go
func (s *FileService) ReadFileContent(path string) (string, error)
func (s *FileService) WriteFileContent(path string, content string) error
func (s *FileService) ListDirectoryContents(path string) ([]string, error)
func (s *FileService) DeleteFile(path string) error
```

### Consistent Return Types

Return types should be predictable. For operations that can fail, always return
an error as the last value. Use structured types for complex responses.

```go
type PagedResult struct {
    Items      []Item `json:"items"`
    Total      int    `json:"total"`
    Page       int    `json:"page"`
    PageSize   int    `json:"pageSize"`
    HasNext    bool   `json:"hasNext"`
}

func (s *ItemService) GetItems(page int, pageSize int) (*PagedResult, error) {
    total, err := s.db.CountItems()
    if err != nil {
        return nil, err
    }

    items, err := s.db.ListItems(page*pageSize, pageSize)
    if err != nil {
        return nil, err
    }

    return &PagedResult{
        Items:    items,
        Total:    total,
        Page:     page,
        PageSize: pageSize,
        HasNext:  (page+1)*pageSize < total,
    }, nil
}
```

### Input Validation

Validate all inputs at the boundary of your binding layer. Return meaningful
errors so the frontend can react appropriately.

```go
func (s *UserService) CreateUser(name string, email string, age int) (*User, error) {
    if strings.TrimSpace(name) == "" {
        return nil, fmt.Errorf("name is required")
    }
    if !strings.Contains(email, "@") {
        return nil, fmt.Errorf("invalid email address")
    }
    if age < 0 || age > 150 {
        return nil, fmt.Errorf("age must be between 0 and 150")
    }

    user := &User{Name: name, Email: email, Age: age}
    if err := s.db.CreateUser(user); err != nil {
        return nil, err
    }
    return user, nil
}
```

## Performance Patterns

### Batch Operations

Prefer batch operations over repeated single-item calls to reduce IPC overhead.

```go
// Good: single call for multiple items
func (s *UserService) GetUsersByIDs(ids []int64) ([]*User, error) {
    return s.db.FindUsersByIDs(ids)
}

// Bad: N separate calls from the frontend
// for (const id of ids) {
//     await GetUser(id);
// }
```

### Pagination

Never return unbounded result sets. Always paginate large collections.

```go
func (s *ProductService) SearchProducts(query string, page int, pageSize int) (*PagedResult, error) {
    if page < 0 {
        page = 0
    }
    if pageSize <= 0 || pageSize > 100 {
        pageSize = 20
    }

    total, err := s.db.CountProducts(query)
    if err != nil {
        return nil, err
    }

    products, err := s.db.SearchProducts(query, page*pageSize, pageSize)
    if err != nil {
        return nil, err
    }

    return &PagedResult{
        Items:    products,
        Total:    total,
        Page:     page,
        PageSize: pageSize,
        HasNext:  (page+1)*pageSize < total,
    }, nil
}
```

### Caching

Cache expensive computations or frequently accessed data to avoid redundant work.

```go
type ConfigService struct {
    mu      sync.RWMutex
    cache   map[string]string
    lastMod time.Time
}

func (s *ConfigService) GetConfig() (map[string]string, error) {
    s.mu.RLock()
    if s.cache != nil {
        defer s.mu.RUnlock()
        return s.cache, nil
    }
    s.mu.RUnlock()

    s.mu.Lock()
    defer s.mu.Unlock()

    info, err := os.Stat(s.configPath)
    if err != nil {
        return nil, err
    }

    if s.cache != nil && !info.ModTime().After(s.lastMod) {
        return s.cache, nil
    }

    data, err := os.ReadFile(s.configPath)
    if err != nil {
        return nil, err
    }

    var config map[string]string
    if err := json.Unmarshal(data, &config); err != nil {
        return nil, err
    }

    s.cache = config
    s.lastMod = info.ModTime()
    return config, nil
}
```

### Streaming with Events

For long-running operations, use events to stream progress updates instead of
blocking until completion.

```go
func (s *DownloadService) DownloadFile(url string, dest string) error {
    resp, err := http.Get(url)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    total := resp.ContentLength
    var downloaded int64

    file, err := os.Create(dest)
    if err != nil {
        return err
    }
    defer file.Close()

    buf := make([]byte, 32*1024)
    for {
        n, err := resp.Body.Read(buf)
        if n > 0 {
            file.Write(buf[:n])
            downloaded += int64(n)

            events.Emit("download:progress", map[string]interface{}{
                "total":      total,
                "downloaded": downloaded,
                "percent":    float64(downloaded) / float64(total) * 100,
            })
        }
        if err == io.EOF {
            break
        }
        if err != nil {
            return err
        }
    }

    events.Emit("download:complete", dest)
    return nil
}
```

## Security Patterns

### Input Sanitisation

Always sanitise user input before using it in file paths, shell commands, or
database queries.

```go
func (s *FileService) ReadFile(path string) (string, error) {
    cleaned := filepath.Clean(path)

    if strings.Contains(cleaned, "..") {
        return "", fmt.Errorf("path traversal not allowed")
    }

    absPath := filepath.Join(s.baseDir, cleaned)
    if !strings.HasPrefix(absPath, s.baseDir) {
        return "", fmt.Errorf("access denied")
    }

    data, err := os.ReadFile(absPath)
    if err != nil {
        return "", err
    }
    return string(data), nil
}
```

### Authentication

Protect sensitive operations with authentication checks.

```go
type AuthService struct {
    sessions map[string]*Session
    mu       sync.RWMutex
}

func (s *AuthService) ValidateSession(token string) (*User, error) {
    s.mu.RLock()
    session, ok := s.sessions[token]
    s.mu.RUnlock()

    if !ok {
        return nil, fmt.Errorf("invalid session")
    }
    if time.Now().After(session.ExpiresAt) {
        return nil, fmt.Errorf("session expired")
    }
    return session.User, nil
}

func (s *DataService) GetSensitiveData(token string) (*SensitiveData, error) {
    user, err := s.auth.ValidateSession(token)
    if err != nil {
        return nil, err
    }
    if !user.HasPermission("read:sensitive") {
        return nil, fmt.Errorf("insufficient permissions")
    }
    return s.db.GetSensitiveData(user.ID)
}
```

### Rate Limiting

Apply rate limits to prevent abuse of expensive operations.

```go
type RateLimiter struct {
    mu       sync.Mutex
    requests map[string][]time.Time
    limit    int
    window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
    return &RateLimiter{
        requests: make(map[string][]time.Time),
        limit:    limit,
        window:   window,
    }
}

func (r *RateLimiter) Allow(key string) bool {
    r.mu.Lock()
    defer r.mu.Unlock()

    now := time.Now()
    cutoff := now.Add(-r.window)

    var valid []time.Time
    for _, t := range r.requests[key] {
        if t.After(cutoff) {
            valid = append(valid, t)
        }
    }

    if len(valid) >= r.limit {
        return false
    }

    r.requests[key] = append(valid, now)
    return true
}
```

## Error Handling Patterns

### Descriptive Errors

Return errors that clearly describe what went wrong and how to fix it.

```go
func (s *FileService) SaveFile(path string, content string) error {
    if path == "" {
        return &BindingError{
            Code:    "EMPTY_PATH",
            Message: "File path cannot be empty",
        }
    }

    if err := s.validatePath(path); err != nil {
        return &BindingError{
            Code:    "INVALID_PATH",
            Message: fmt.Sprintf("Invalid file path: %v", err),
        }
    }

    if err := os.WriteFile(path, []byte(content), 0644); err != nil {
        return &BindingError{
            Code:    "WRITE_FAILED",
            Message: fmt.Sprintf("Failed to write file: %v", err),
        }
    }
    return nil
}
```

### Error Types

Define structured error types that the frontend can handle programmatically.

```go
type BindingError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
}

func (e *BindingError) Error() string {
    return e.Message
}

func (e *BindingError) Unwrap() error {
    return nil
}

// Frontend usage:
// try {
//   await SaveFile(path, content);
// } catch (e) {
//   if (e.code === "INVALID_PATH") {
//     showPathError(e.message);
//   } else {
//     showGenericError(e.message);
//   }
// }
```

### Error Recovery

Design operations to be recoverable. Clean up partial state on failure.

```go
func (s *TransactionService) TransferFunds(fromID, toID int64, amount float64) error {
    tx, err := s.db.Begin()
    if err != nil {
        return fmt.Errorf("failed to start transaction: %w", err)
    }

    if err := tx.Withdraw(fromID, amount); err != nil {
        tx.Rollback()
        return fmt.Errorf("withdrawal failed: %w", err)
    }

    if err := tx.Deposit(toID, amount); err != nil {
        tx.Rollback()
        return fmt.Errorf("deposit failed: %w", err)
    }

    if err := tx.Commit(); err != nil {
        tx.Rollback()
        return fmt.Errorf("commit failed: %w", err)
    }

    return nil
}
```

## Testing Patterns

### Unit Testing

Test binding methods in isolation with mocked dependencies.

```go
func TestUserService_GetUser(t *testing.T) {
    db := &MockDB{
        FindUserFunc: func(id int64) (*User, error) {
            if id == 1 {
                return &User{ID: 1, Name: "Alice"}, nil
            }
            return nil, fmt.Errorf("not found")
        },
    }

    service := NewUserService(db)

    user, err := service.GetUser(1)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if user.Name != "Alice" {
        t.Errorf("expected Alice, got %s", user.Name)
    }
}

func TestUserService_GetUser_NotFound(t *testing.T) {
    db := &MockDB{}
    service := NewUserService(db)

    _, err := service.GetUser(999)
    if err == nil {
        t.Fatal("expected error for missing user")
    }
}
```

### Integration Testing

Test binding methods against real dependencies in a controlled environment.

```go
func TestFileService_Integration(t *testing.T) {
    dir := t.TempDir()
    service := NewFileService(dir)

    content := "Hello, World!"
    if err := service.SaveFile("test.txt", content); err != nil {
        t.Fatalf("save failed: %v", err)
    }

    got, err := service.ReadFile("test.txt")
    if err != nil {
        t.Fatalf("read failed: %v", err)
    }
    if got != content {
        t.Errorf("expected %q, got %q", content, got)
    }

    if err := service.DeleteFile("test.txt"); err != nil {
        t.Fatalf("delete failed: %v", err)
    }
}
```

### Mock Services

Create lightweight mock services for testing frontend integration without real
backends.

```go
type MockNotificationService struct {
    Notifications []Notification
}

func (m *MockNotificationService) Send(title, message string) error {
    m.Notifications = append(m.Notifications, Notification{
        Title:   title,
        Message: message,
        SentAt:  time.Now(),
    })
    return nil
}

func (m *MockNotificationService) GetLast() *Notification {
    if len(m.Notifications) == 0 {
        return nil
    }
    return &m.Notifications[len(m.Notifications)-1]
}
```
