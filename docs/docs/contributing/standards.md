---
title: "Coding Standards"
description: "Code style, conventions, and best practices for Wails v3"
---

## Go Code Standards

### Code Formatting

Use standard Go formatting tools:

```
gofmt -w .
goimports -w .
```

### Naming Conventions

**Packages:** Lowercase, single word when possible (e.g., `package application`).

**Exported Names:** PascalCase for types, functions, constants.

**Unexported Names:** camelCase for internal types, functions, variables.

**Interfaces:** Name by behavior with `-er` suffix (e.g., `Closer`, `Reader`).

### Error Handling

Always check errors and use error wrapping:

```go
result, err := doSomething()
if err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}
```

### Comments and Documentation

Use package comments for packages and doc comments for exported declarations.

### Testing

Use table-driven tests:

```go
func TestValidate(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"empty input", "", true},
        {"valid input", "hello", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validate(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## JavaScript/TypeScript Standards

### Code Formatting

Use Prettier with:
```json
{
  "semi": false,
  "singleQuote": true,
  "tabWidth": 2,
  "trailingComma": "es5"
}
```

### Naming Conventions

* **Variables and functions:** camelCase
* **Classes and types:** PascalCase
* **Constants:** UPPER_SNAKE_CASE

## Commit Message Format

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <subject>

<body>

<footer>
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`
