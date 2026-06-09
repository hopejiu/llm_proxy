---
title: "Development Setup"
description: "Set up your development environment for Wails v3 development"
---

## Required Tools

### Go Development

1. Install Go 1.25 or later from https://go.dev/dl/
2. Configure Go environment:
   ```
   export GOPATH=$HOME/go
   export PATH=$PATH:$GOPATH/bin
   ```
3. Install useful Go tools:
   ```
   go install golang.org/x/tools/cmd/goimports@latest
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   ```

### Node.js and npm

Install Node.js 20+ and npm.

### Platform-Specific Dependencies

**macOS:** `xcode-select --install`

**Windows:** Install MSYS2 and WebView2 Runtime.

**Linux (Debian/Ubuntu):**
```
sudo apt install build-essential pkg-config libgtk-4-dev libwebkitgtk-6.0-dev
```

## Repository Setup

```
git clone https://github.com/YOUR_USERNAME/wails.git
cd wails
git remote add upstream https://github.com/wailsapp/wails.git

# Build the CLI
cd v3
go build -o ../wails3 ./cmd/wails3
```

## IDE Setup

### VS Code

Install Go, ESLint, Prettier, and MDX extensions.

Configure workspace settings:
```json
{
  "go.useLanguageServer": true,
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "workspace",
  "editor.formatOnSave": true,
  "go.formatTool": "goimports"
}
```

## Verify Your Setup

```
cd v3 && go build ./cmd/wails3
go test ./pkg/...
```

## Running Tests

```
cd v3
go test ./...
go test ./... -race
go test ./... -coverprofile=coverage.out
```

## Working with Documentation

```
cd docs
npm install
npm run dev
```

Documentation will be available at `http://localhost:4321/`.
