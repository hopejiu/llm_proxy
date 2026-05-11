# CODEBUDDY.md

This file provides guidance to CodeBuddy Code when working with code in this repository.

## Build & Run Commands

```bash
# Build (CGO_ENABLED=0 for static binary, no SQLite CGO dependency)
CGO_ENABLED=0 go build -ldflags="-s -w" -o server ./cmd/server

# Run
go run ./cmd/server

# Run with SQLite (no MySQL needed)
DB_TYPE=sqlite go run ./cmd/server

# Release build (cross-platform)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o llm-proxy-linux-amd64 ./cmd/server
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o llm-proxy-windows-amd64.exe ./cmd/server
```

No test suite exists in this project. No linting configuration is present.

## Architecture

This is a Go LLM API proxy service that simultaneously provides OpenAI, Anthropic, and Ollama compatible interfaces, converting between protocols and forwarding to upstream providers.

### Two HTTP Servers

- **Web management** (default port 80): Gin engine serving HTML templates + REST API for provider CRUD, stats, and logs
- **Proxy service** (default port 8888): Gin engine handling LLM API proxy requests (OpenAI/Anthropic/Ollama)

### Layered Architecture (Repository → Service → Handler)

```
cmd/server/main.go          → Entry point, DB init, DI wiring, two HTTP servers, graceful shutdown
internal/repository/        → GORM data access (provider_repo, request_log_repo, hourly_stat_repo)
internal/service/           → Business logic (proxy_service, provider_service, stats_service, cleanup_service)
internal/handler/           → Gin HTTP handlers (base_handler, proxy_handler, anthropic_handler, ollama_handler, web_handler)
internal/router/router.go   → Route definitions for both engines
```

### Key Design Patterns

- **BaseHandler**: Shared base struct for proxy handlers (OpenAI/Anthropic/Ollama), providing HTTP client, retry logic, stream reading with first-byte timeout detection, request logging
- **Protocol conversion**: `internal/converter/` handles bidirectional conversion between Anthropic↔OpenAI and Ollama↔OpenAI formats. All upstream requests are sent in OpenAI format; responses are converted back to the client's protocol
- **Provider caching**: `ProxyService` caches provider list with TTL (default 30s), invalidated on CRUD operations. Model routing: alias match first, then model name match
- **Stream retry**: `ExecuteStreamWithRetry` detects first-byte timeout and retries with exponential backoff. Non-stream requests use `SendRequestWithRetry`
- **Hourly stats aggregation**: `hourly_stats` table aggregates token usage per hour. Stats queries = aggregated hours (from hourly_stats) + current hour (live from request_logs). CleanupService runs periodic aggregation and old data deletion
- **Dual DB support**: MySQL (default) or SQLite (`DB_TYPE=sqlite`). SQLite uses WAL mode, single connection. MySQL uses connection pool (25 max open, 10 idle)

### Request Flow (OpenAI example)

```
Client → ProxyHandler.ChatCompletions()
  → ReadBody, parse model/stream
  → ProxyService.GetProviderByModel(model) [alias first, then model name]
  → ProxyService.PrepareRequestBody() [replace model, merge ExtraParams, inject stream_options]
  → BaseHandler.ExecuteStreamWithRetry() or SendRequestWithRetry()
  → Stream SSE lines → client, extract usage tokens
  → SaveRequestLog to request_logs table
```

### Environment Configuration

All config via `.env` file or environment variables (env vars take priority). Key vars: `DB_TYPE`, `DB_PATH`, `DB_HOST/PORT/USER/PASSWORD/NAME`, `WEB_PORT`, `PROXY_PORT`, `HTTP_TIMEOUT`, `STREAM_FIRST_BYTE_TIMEOUT`, `STREAM_MAX_RETRIES`, `PROVIDER_CACHE_TTL`, `LOG_CLEANUP_DAYS`.

### Frontend

Pure static HTML + Tailwind CSS + ECharts, no build step. Templates in `web/templates/`, JS/CSS in `web/static/`. Loaded via Gin's `LoadHTMLGlob` and `Static`.

### Release

GitHub Actions (`release.yml`) builds on `v*` tag push. Produces Windows (.zip) and Linux (.tar.gz) packages containing binary + web directory + .env.