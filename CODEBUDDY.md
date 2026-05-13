# CODEBUDDY.md

This file provides guidance to CodeBuddy Code when working with code in this repository.

## Build & Run Commands

```bash
# Development (Wails dev mode with hot reload)
wails dev

# Build (Wails CLI, produces build/bin/llm-proxy.exe)
wails build -ldflags="-s -w"

# Build with version info
wails build -ldflags="-s -w -X main.Version=1.0.0 -X main.BuildTime=2026-05-13"

# Run with SQLite (no MySQL needed, set in .env)
# Edit .env: DB_TYPE=sqlite
```

No test suite exists in this project. No linting configuration is present.

## Architecture

This is a **Wails desktop application** (Go + WebView2) that provides an LLM API proxy service. It simultaneously offers OpenAI, Anthropic, and Ollama compatible interfaces, converting between protocols and forwarding to upstream providers.

### Wails Desktop App

- **Entry point**: `main.go` — initializes DB, DI wiring, starts Wails window with embedded frontend
- **Frontend**: Vite + vanilla JS, embedded via `go:embed all:frontend/dist`
- **Proxy service**: Gin engine on configurable port (default 8888), started/stopped from the desktop UI
- **No separate web server**: The old `web/` directory (Gin templates) is legacy; the active UI is the Wails frontend

### Layered Architecture (Repository → Service → Handler)

```
main.go                       → Entry point, DB init, DI wiring, Wails app startup
app.go                        → App core logic (Wails binding layer, proxy lifecycle)
app_error.go                  → Unified error types
vo.go                         → View objects (frontend data conversion)
internal/config/config.go     → Configuration management (.env loading, validation, persistence)
internal/converter/           → Protocol conversion (Anthropic↔OpenAI, Ollama↔OpenAI)
internal/handler/             → Gin HTTP handlers + active request tracking
  base_handler.go             → Shared base (HTTP client, retry, stream timeout, logging)
  proxy_handler.go            → OpenAI compatible proxy
  anthropic_handler.go        → Anthropic compatible proxy
  anthropic_stream.go         → Anthropic stream state machine
  ollama_handler.go           → Ollama compatible proxy
  active_tracker.go           → Active request tracker (real-time monitoring)
  stream_tracker.go           → Stream content tracker (tool calls)
  web_handler.go              → Web management API (legacy, for Gin template mode)
  response.go                 → Unified response format
internal/logger/logger.go     → Log initialization, archive, cleanup
internal/logger/reader.go     → Incremental log file reader (for UI log viewer)
internal/middleware/cors.go   → CORS middleware
internal/model/models.go      → Core models (ProviderConfig, RequestLog, HourlyStat)
internal/model/anthropic_models.go → Anthropic request/response models
internal/model/ollama_models.go    → Ollama request/response models
internal/repository/          → GORM data access (provider_repo, request_log_repo, hourly_stat_repo)
internal/service/             → Business logic
  proxy_service.go            → Proxy core + Provider cache
  provider_service.go         → Provider management
  stats_service.go            → Stats (hourly_stats + live aggregation)
  cleanup_service.go          → Periodic aggregation and cleanup
  codebuddy_service.go        → CodeBuddy models.json setup
internal/router/router.go     → Proxy route definitions
frontend/                     → Wails frontend (Vite + vanilla JS)
  src/main.js                 → Frontend entry
  src/common.js               → Shared utilities
  src/style.css               → Global styles
  src/pages/                  → Page fragments (providers, stats, logs, realtime, settings)
  src/assets/                 → Fonts, images
  dist/                       → Build output (embedded in Go binary)
```

### Key Design Patterns

- **Wails binding**: `App` struct methods are exposed to frontend via Wails bindings (`frontend/wailsjs/go/main/App.js`)
- **BaseHandler**: Shared base struct for proxy handlers (OpenAI/Anthropic/Ollama), providing HTTP client, retry logic, stream reading with first-byte timeout detection, request logging
- **Protocol conversion**: `internal/converter/` handles bidirectional conversion between Anthropic↔OpenAI and Ollama↔OpenAI formats. All upstream requests are sent in OpenAI format; responses are converted back to the client's protocol
- **Provider caching**: `ProxyService` caches provider list with TTL (default 30s), invalidated on CRUD operations. Model routing: alias match first, then model name match
- **Stream retry**: `ExecuteStreamWithRetry` detects first-byte timeout and retries with exponential backoff. Non-stream requests use `SendRequestWithRetry`
- **Hourly stats aggregation**: `hourly_stats` table aggregates token usage per hour. Stats queries = aggregated hours (from hourly_stats) + current hour (live from request_logs). CleanupService runs periodic aggregation and old data deletion
- **Dual DB support**: MySQL (default) or SQLite (`DB_TYPE=sqlite`). SQLite uses WAL mode, single connection. MySQL uses connection pool (25 max open, 10 idle)
- **Log archive**: On startup, current log file is archived to `llm-proxy-YYYY-MM-DD.log`. Old archives cleaned per `LOG_CLEANUP_DAYS`
- **VO pattern**: `vo.go` converts model structs to view objects with API key masking and date formatting

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

All config via `.env` file (stored in `%APPDATA%/llm-proxy/`) or environment variables (env vars take priority). Key vars: `DB_TYPE`, `DB_PATH`, `DB_HOST/PORT/USER/PASSWORD/NAME`, `PROXY_PORT`, `HTTP_TIMEOUT`, `STREAM_FIRST_BYTE_TIMEOUT`, `STREAM_MAX_RETRIES`, `PROVIDER_CACHE_TTL`, `LOG_CLEANUP_DAYS`, `LOG_LEVEL`, `AUTO_START_PROXY`.

### Frontend

Wails-embedded SPA: Vite + vanilla JS + Tailwind CSS (CDN) + ECharts (CDN). Source in `frontend/src/`, built to `frontend/dist/`, embedded in Go binary via `go:embed`. Pages: providers, stats, logs, realtime, settings.

### Release

GitHub Actions (`release.yml`) builds on `v*` tag push. Uses Wails CLI to build Windows binary, packages as .zip with .env file.

### Legacy

The `web/` directory contains an older Gin-template-based web UI (HTML templates + static JS/CSS). This is not actively used in the Wails desktop app but retained for potential standalone server mode.
