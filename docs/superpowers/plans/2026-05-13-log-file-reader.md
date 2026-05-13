# 运行日志文件读取方案 实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 将运行日志从 RingBuffer + Wails 事件推送架构改为直接读取日志文件 + 前端轮询架构

**架构：** slog 写入日志文件，Go 端 LogReader 维护文件偏移量增量读取，前端每秒轮询 GetNewLogs() 获取新日志。启动时归档旧日志文件，自动清理过期归档。

**技术栈：** Go slog, os.File, Wails JS binding, setInterval

---

## 文件结构

| 文件 | 操作 | 职责 |
|------|------|------|
| `internal/logger/logger.go` | 重写 | 简化 Init（移除 RingBuffer 参数），添加归档和清理函数 |
| `internal/logger/reader.go` | 创建 | LogReader：增量读取日志文件，解析 slog 文本格式 |
| `internal/logger/ringbuffer.go` | 删除 | 不再需要 RingBuffer |
| `internal/logger/handler.go` | 删除 | 不再需要 multiHandler / formatLevel |
| `app.go` | 修改 | 移除 RingBuffer/事件推送，添加 GetNewLogs，添加 logReader 字段 |
| `main.go` | 修改 | 移除 RingBuffer 创建，调用归档/清理，传递 logReader |
| `config/config.go` | 修改 | LOG_CLEANUP_DAYS 默认值改为 3，描述改为"日志归档保留天数" |
| `frontend/src/pages/logs.js` | 重写 | 移除 EventsOn，改为 setInterval 轮询 GetNewLogs |
| `vo.go` | 修改 | 无需改动（LogEntryVO 已存在且兼容） |

---

### 任务 1：简化 logger.go，添加归档和清理

**文件：**
- 重写：`internal/logger/logger.go`
- 删除：`internal/logger/handler.go`

- [ ] **步骤 1：重写 logger.go**

移除所有 RingBuffer 相关代码（`globalRingBuffer`、`ringBufferWriter`、`parseSlogTextFormat`、`extractAttr`、`extractExtraAttrs`、`GetRingBuffer`、`GlobalLogFile`、`GlobalLogFileForDebug`），简化 `Init` 签名：

```go
package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// ParseLevel 将字符串解析为 slog.Level
func ParseLevel(levelStr string) slog.Level {
	switch strings.ToLower(levelStr) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// 全局日志文件句柄，用于关闭和复用
var globalLogFile *os.File

// Init 初始化全局 Logger，同时输出到 stdout 和指定日志文件
func Init(logFilePath string, level slog.Level) error {
	if globalLogFile != nil {
		globalLogFile.Close()
	}

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	globalLogFile = logFile

	writers := []io.Writer{logFile}
	if os.Stdout != nil {
		writers = append(writers, os.Stdout)
	}
	baseWriter := io.MultiWriter(writers...)

	handler := slog.NewTextHandler(baseWriter, &slog.HandlerOptions{
		Level: level,
	})

	l := slog.New(handler)
	slog.SetDefault(l)

	return nil
}

// Sync 刷盘日志文件
func Sync() {
	if globalLogFile != nil {
		globalLogFile.Sync()
	}
}

// Close 关闭日志文件
func Close() {
	if globalLogFile != nil {
		globalLogFile.Close()
		globalLogFile = nil
	}
}

// ArchiveLogFile 归档当前日志文件
// 将 logFilePath 的内容追加到 logFilePath 同目录下的 llm-proxy-YYYY-MM-DD.log
// 然后删除原文件，让 Init 创建新的空文件
func ArchiveLogFile(logFilePath string) error {
	info, err := os.Stat(logFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 文件不存在，无需归档
		}
		return err
	}
	if info.Size() == 0 {
		return nil // 空文件，无需归档
	}

	// 读取原文件内容
	data, err := os.ReadFile(logFilePath)
	if err != nil {
		return err
	}

	// 归档文件名：llm-proxy-2026-05-13.log
	dir := dirOfFile(logFilePath)
	archiveName := "llm-proxy-" + nowDate() + ".log"
	archivePath := joinPath(dir, archiveName)

	// 追加到归档文件（同一天多次启动合并）
	f, err := os.OpenFile(archivePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	f.Write(data)
	f.Close()

	// 删除原文件
	return os.Remove(logFilePath)
}

// CleanOldArchives 清理过期的归档日志文件
func CleanOldArchives(logDir string, keepDays int) error {
	if keepDays <= 0 {
		keepDays = 3
	}

	entries, err := os.ReadDir(logDir)
	if err != nil {
		return err
	}

	cutoff := time.Now().AddDate(0, 0, -keepDays)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// 匹配 llm-proxy-YYYY-MM-DD.log 格式
		if !strings.HasPrefix(name, "llm-proxy-") || !strings.HasSuffix(name, ".log") {
			continue
		}
		// 提取日期部分：llm-proxy-2026-05-13.log → 2026-05-13
		dateStr := strings.TrimPrefix(name, "llm-proxy-")
		dateStr = strings.TrimSuffix(dateStr, ".log")
		t, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue // 文件名不匹配日期格式，跳过
		}
		if t.Before(cutoff) {
			os.Remove(joinPath(logDir, name))
		}
	}

	return nil
}
```

注意：需要添加 `"time"` 到 import。`dirOfFile`、`nowDate`、`joinPath` 是辅助函数：

```go
func dirOfFile(path string) string {
	return filepath.Dir(path)
}

func nowDate() string {
	return time.Now().Format("2006-01-02")
}

func joinPath(dir, name string) string {
	return filepath.Join(dir, name)
}
```

需要 import `"path/filepath"` 和 `"time"`。

- [ ] **步骤 2：删除 handler.go**

删除 `internal/logger/handler.go` 文件。

- [ ] **步骤 3：运行 go vet 验证编译**

```bash
cd d:\teamsun_project\teamsun\code\other\llm_statistic && go vet ./...
```

预期：编译错误，因为 `ringbuffer.go` 还存在但 `logger.go` 不再引用 `RingBuffer`。先忽略，任务 2 会删除 ringbuffer.go。

---

### 任务 2：删除 RingBuffer，创建 LogReader

**文件：**
- 删除：`internal/logger/ringbuffer.go`
- 创建：`internal/logger/reader.go`

- [ ] **步骤 1：删除 ringbuffer.go**

删除 `internal/logger/ringbuffer.go`。

- [ ] **步骤 2：创建 reader.go**

```go
package logger

import (
	"os"
	"strings"
	"sync"
)

// LogEntry 日志条目
type LogEntry struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

// LogReader 从日志文件增量读取日志条目
type LogReader struct {
	filePath string
	offset   int64
	mu       sync.Mutex
}

// NewLogReader 创建 LogReader，offset 初始化为 0
func NewLogReader(filePath string) *LogReader {
	return &LogReader{
		filePath: filePath,
		offset:   0,
	}
}

// ReadAllLogs 从文件头读取全部日志，更新 offset 到文件末尾
func (r *LogReader) ReadAllLogs() []LogEntry {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return nil
	}

	// 更新 offset 到文件末尾
	r.offset = int64(len(data))

	return parseSlogLines(data)
}

// ReadNewLogs 从 offset 位置读取新增日志，更新 offset
func (r *LogReader) ReadNewLogs() []LogEntry {
	r.mu.Lock()
	defer r.mu.Unlock()

	info, err := os.Stat(r.filePath)
	if err != nil {
		return nil
	}

	fileSize := info.Size()
	if fileSize <= r.offset {
		return nil // 没有新内容
	}

	f, err := os.Open(r.filePath)
	if err != nil {
		return nil
	}
	defer f.Close()

	buf := make([]byte, fileSize-r.offset)
	n, err := f.ReadAt(buf, r.offset)
	if err != nil && n == 0 {
		return nil
	}
	buf = buf[:n]

	// 更新 offset
	r.offset += int64(n)

	return parseSlogLines(buf)
}

// ResetOffset 重置 offset 为 0（用于文件归档后重新读取）
func (r *LogReader) ResetOffset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.offset = 0
}

// parseSlogLines 解析 slog TextHandler 输出格式的多行文本
func parseSlogLines(data []byte) []LogEntry {
	text := string(data)
	lines := strings.Split(text, "\n")

	var entries []LogEntry
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		entry := parseSlogLine(line)
		if entry != nil {
			entries = append(entries, *entry)
		}
	}
	return entries
}

// parseSlogLine 解析单行 slog TextHandler 输出
// 格式: time=2026-05-13T10:50:07.481+08:00 level=INFO msg="配置加载成功" db_type=mysql proxy_port=8888
func parseSlogLine(line string) *LogEntry {
	entry := &LogEntry{}

	// 提取 level
	levelStr := extractAttr(line, "level")
	if levelStr == "" {
		levelStr = "info"
	}
	entry.Level = strings.ToLower(levelStr)

	// 提取 msg
	msgStr := extractAttr(line, "msg")
	if msgStr == "" {
		return nil
	}
	msgStr = strings.Trim(msgStr, "\"")
	entry.Message = msgStr

	// 提取 time
	timeStr := extractAttr(line, "time")
	if timeStr != "" {
		timeStr = strings.Trim(timeStr, "\"")
		entry.Time = formatTimeShort(timeStr)
	}

	// 拼接额外属性到 message
	extraAttrs := extractExtraAttrs(line, []string{"time", "level", "msg"})
	if extraAttrs != "" {
		entry.Message += " " + extraAttrs
	}

	return entry
}

// formatTimeShort 将 ISO 时间格式转为 HH:MM:SS.mmm
func formatTimeShort(timeStr string) string {
	if len(timeStr) < 19 {
		return timeStr
	}
	parts := strings.SplitN(timeStr, "T", 2)
	if len(parts) != 2 {
		return timeStr
	}
	timePart := parts[1]
	// 去掉时区偏移
	if idx := strings.Index(timePart, "+"); idx > 0 {
		timePart = timePart[:idx]
	}
	if idx := strings.Index(timePart, "-"); idx > 4 {
		timePart = timePart[:idx]
	}
	// 保留3位毫秒
	if idx := strings.Index(timePart, "."); idx > 0 && idx+4 <= len(timePart) {
		timePart = timePart[:idx+4]
	}
	return timePart
}

// extractAttr 从 slog TextHandler 格式的行中提取指定属性值
func extractAttr(line, key string) string {
	prefix := key + "="
	idx := strings.Index(line, prefix)
	if idx < 0 {
		return ""
	}

	start := idx + len(prefix)
	if start >= len(line) {
		return ""
	}

	if line[start] == '"' {
		end := start + 1
		for end < len(line) {
			if line[end] == '"' && (end+1 >= len(line) || line[end+1] == ' ') {
				return line[start : end+1]
			}
			end++
		}
		return line[start:]
	}

	end := start
	for end < len(line) && line[end] != ' ' {
		end++
	}
	return line[start:end]
}

// extractExtraAttrs 提取除 skipKeys 外的所有属性
func extractExtraAttrs(line string, skipKeys []string) string {
	var result strings.Builder
	remaining := line

	for {
		eqIdx := strings.Index(remaining, "=")
		if eqIdx < 0 {
			break
		}

		keyStart := eqIdx - 1
		for keyStart >= 0 && remaining[keyStart] != ' ' {
			keyStart--
		}
		keyStart++

		key := remaining[keyStart:eqIdx]

		skip := false
		for _, sk := range skipKeys {
			if key == sk {
				skip = true
				break
			}
		}

		valStart := eqIdx + 1
		if valStart >= len(remaining) {
			break
		}

		var valEnd int
		if remaining[valStart] == '"' {
			valEnd = valStart + 1
			for valEnd < len(remaining) {
				if remaining[valEnd] == '"' && (valEnd+1 >= len(remaining) || remaining[valEnd+1] == ' ') {
					valEnd++
					break
				}
				valEnd++
			}
		} else {
			valEnd = valStart
			for valEnd < len(remaining) && remaining[valEnd] != ' ' {
				valEnd++
			}
		}

		if !skip {
			if result.Len() > 0 {
				result.WriteByte(' ')
			}
			result.WriteString(key)
			result.WriteByte('=')
			result.WriteString(remaining[valStart:valEnd])
		}

		if valEnd < len(remaining) && remaining[valEnd] == ' ' {
			remaining = remaining[valEnd+1:]
		} else {
			break
		}
	}

	return result.String()
}
```

- [ ] **步骤 3：运行 go vet 验证编译**

```bash
cd d:\teamsun_project\teamsun\code\other\llm_statistic && go vet ./...
```

预期：logger 包编译通过。app.go 和 main.go 仍有 RingBuffer 引用错误，任务 3 修复。

---

### 任务 3：更新 main.go 和 app.go

**文件：**
- 修改：`main.go`
- 修改：`app.go`

- [ ] **步骤 1：修改 main.go**

关键变更：
1. 移除 `ringBuffer` 变量和 `logger.NewRingBuffer` 调用
2. 第一次 `logger.Init` 前调用 `logger.ArchiveLogFile` 归档旧日志
3. 第二次 `logger.Init` 不再传 RingBuffer 参数
4. 调用 `logger.CleanOldArchives` 清理过期归档
5. 创建 `logger.NewLogReader` 并传给 App

```go
func main() {
	dataDir := config.DataDir()

	// 归档旧日志文件（启动前执行，确保 Init 创建新文件）
	logFilePath := filepath.Join(dataDir, "llm-proxy.log")
	logger.ArchiveLogFile(logFilePath)

	// 清理过期归档日志
	cfg := config.Load()
	logger.CleanOldArchives(dataDir, cfg.LogCleanupDays)

	// 初始化日志
	if err := logger.Init(logFilePath, slog.LevelInfo); err != nil {
		fmt.Printf("无法创建日志文件: %v\n", err)
		os.Exit(1)
	}

	slog.Info("LLM Proxy 桌面应用启动中...")

	// 根据配置重新设置日志级别
	logger.Init(logFilePath, logger.ParseLevel(cfg.LogLevel))

	slog.Info("配置加载成功", "db_type", cfg.DBType, "proxy_port", cfg.ProxyPort)

	// ... 数据库初始化等不变 ...

	// 创建日志读取器
	logReader := logger.NewLogReader(logFilePath)

	// 创建 App
	app := &App{
		cfg:              cfg,
		providerService:  providerService,
		statsService:     statsService,
		tracker:          tracker,
		logReader:        logReader,
		cleanupSvc:       cleanupSvc,
		proxyHandler:     proxyHandler,
		anthropicHandler: anthropicHandler,
		ollamaHandler:    ollamaHandler,
		proxyState:       proxyState{status: "stopped"},
	}

	slog.Info("正在启动 Wails 窗口...")
	// ... Wails 配置不变 ...
}
```

- [ ] **步骤 2：修改 app.go**

关键变更：
1. `ringBuffer` 字段改为 `logReader *logger.LogReader`
2. 移除 `startLogEventPusher` 方法
3. `startup` 中移除 `go a.startLogEventPusher()`
4. `GetLogHistory` 改用 `logReader.ReadAllLogs()`
5. 新增 `GetNewLogs` 方法

App 结构体字段变更：
```go
type App struct {
	ctx              context.Context
	providerService  *service.ProviderService
	statsService     *service.StatsService
	tracker          *handler.ActiveRequestTracker
	logReader        *logger.LogReader    // 替换 ringBuffer
	cleanupSvc       *service.CleanupService
	cfg              *config.Config
	proxyState       proxyState
	cleanupCancel    context.CancelFunc
	proxyHandler     *handler.ProxyHandler
	anthropicHandler *handler.AnthropicHandler
	ollamaHandler    *handler.OllamaHandler
}
```

GetLogHistory 和 GetNewLogs：
```go
// GetLogHistory 获取全部日志（首次加载）
func (a *App) GetLogHistory() []LogEntryVO {
	entries := a.logReader.ReadAllLogs()
	result := make([]LogEntryVO, len(entries))
	for i, e := range entries {
		result[i] = LogEntryVO(e)
	}
	return result
}

// GetNewLogs 获取增量日志（轮询）
func (a *App) GetNewLogs() []LogEntryVO {
	entries := a.logReader.ReadNewLogs()
	if entries == nil {
		return []LogEntryVO{}
	}
	result := make([]LogEntryVO, len(entries))
	for i, e := range entries {
		result[i] = LogEntryVO(e)
	}
	return result
}
```

移除 `startLogEventPusher` 方法，从 `startup` 中移除 `go a.startLogEventPusher()` 调用。

移除 `"log/slog"` import（如果 app.go 中不再有 slog 调用——但 app.go 中其他方法仍有 slog 调用，保留）。

- [ ] **步骤 3：运行 go vet 验证编译**

```bash
cd d:\teamsun_project\teamsun\code\other\llm_statistic && go vet ./...
```

预期：编译通过。

---

### 任务 4：更新配置默认值

**文件：**
- 修改：`internal/config/config.go`

- [ ] **步骤 1：修改 LOG_CLEANUP_DAYS 默认值和描述**

在 `Load()` 函数中，将默认值从 14 改为 3：
```go
LogCleanupDays: getEnvInt("LOG_CLEANUP_DAYS", 3),
```

在 `GetEnvItems()` 中，修改描述和选项：
```go
{Key: "LOG_CLEANUP_DAYS", Label: "日志归档保留天数", Value: getEnv("LOG_CLEANUP_DAYS", "3"), DefaultValue: "3", Type: "select", Group: "其他", Description: "自动清理多少天前的日志归档文件",
    Options: []EnvSelectOption{
        {Value: "1", Label: "1天"}, {Value: "3", Label: "3天"}, {Value: "7", Label: "7天"}, {Value: "14", Label: "14天"}, {Value: "30", Label: "30天"},
    }},
```

- [ ] **步骤 2：运行 go vet 验证编译**

```bash
cd d:\teamsun_project\teamsun\code\other\llm_statistic && go vet ./...
```

---

### 任务 5：重写前端 logs.js

**文件：**
- 重写：`frontend/src/pages/logs.js`

- [ ] **步骤 1：重写 logs.js**

核心变更：
1. 移除 `EventsOn`/`EventsOff` 导入和使用
2. 添加 `setInterval` 每秒轮询 `GetNewLogs`
3. `destroy` 时清除定时器
4. 暂停时停止轮询，继续时恢复并补齐
5. 保留现有的过滤、搜索、清屏、自动滚动逻辑

```javascript
import { callGo, escapeHtml, pad, showToast } from '../common.js';

let allLogs = [];
let paused = false;
let maxLogs = 2000;
let pollTimer = null;

async function init() {
  await loadHistory();
  startPolling();

  window.filterLogs = filterLogs;
  window.togglePause = togglePause;
  window.clearLogs = clearLogs;
}

function destroy() {
  stopPolling();
  ['filterLogs', 'togglePause', 'clearLogs'].forEach(fn => delete window[fn]);
}

function startPolling() {
  stopPolling();
  pollTimer = setInterval(pollNewLogs, 1000);
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer);
    pollTimer = null;
  }
}

async function loadHistory() {
  try {
    const entries = await callGo('GetLogHistory');
    allLogs = entries || [];
    renderLogs();
  } catch (e) {
    console.error('[logs] Failed to load log history:', e);
    showToast('加载日志历史失败: ' + e, 'error');
  }
}

async function pollNewLogs() {
  if (paused) return;
  try {
    const entries = await callGo('GetNewLogs');
    if (entries && entries.length > 0) {
      allLogs.push(...entries);
      if (allLogs.length > maxLogs) allLogs = allLogs.slice(-maxLogs);
      renderLogs();
    }
  } catch (e) {
    console.error('[logs] Poll failed:', e);
  }
}

function filterLogs() {
  renderLogs();
}

function togglePause() {
  paused = !paused;
  const btn = document.getElementById('logPauseBtn');
  if (btn) btn.classList.toggle('active', paused);
  showToast(paused ? '日志已暂停' : '日志已继续', 'success');
}

function clearLogs() {
  allLogs = [];
  renderLogs();
}

function renderLogs() {
  const body = document.getElementById('logViewerBody');
  if (!body) return;

  const levelFilter = document.getElementById('logLevelFilter')?.value || 'all';
  const search = (document.getElementById('logSearchInput')?.value || '').toLowerCase();

  let filtered = allLogs;
  if (levelFilter !== 'all') {
    filtered = filtered.filter(l => l.level === levelFilter);
  }
  if (search) {
    filtered = filtered.filter(l => (l.message || '').toLowerCase().includes(search));
  }

  const countEl = document.getElementById('logCount');
  if (countEl) countEl.textContent = `${filtered.length} 条日志`;

  const displayLogs = filtered.slice(-500);

  if (displayLogs.length === 0) {
    body.innerHTML = '<div style="padding:20px;text-align:center;color:#999;">暂无日志数据</div>';
    return;
  }

  const wasAtBottom = body.scrollTop + body.clientHeight >= body.scrollHeight - 20;

  body.innerHTML = displayLogs.map(l => {
    const time = l.time ? formatLogTime(l.time) : '';
    const level = (l.level || 'info').toLowerCase();
    return `<div class="log-line"><span class="log-line-time">${time}</span><span class="log-line-level ${level}">${level.toUpperCase()}</span><span class="log-line-msg">${escapeHtml(l.message)}</span></div>`;
  }).join('');

  if (wasAtBottom) body.scrollTop = body.scrollHeight;
}

function formatLogTime(timeStr) {
  if (!timeStr) return '';
  if (/^\d{2}:\d{2}:\d{2}/.test(timeStr) && !timeStr.includes('T') && !timeStr.includes('-')) {
    return timeStr;
  }
  try {
    const d = new Date(timeStr);
    if (isNaN(d.getTime())) return timeStr;
    return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}.${String(d.getMilliseconds()).padStart(3, '0')}`;
  } catch (e) { return timeStr; }
}

export { init, destroy };
```

- [ ] **步骤 2：构建并验证**

```bash
cd d:\teamsun_project\teamsun\code\other\llm_statistic && powershell -ExecutionPolicy Bypass -File build.ps1
```

---

### 任务 6：端到端验证

- [ ] **步骤 1：运行应用，进入运行日志页面**

启动 `.\dist\llm-proxy.exe`，进入"运行日志"页面，确认：
1. 页面显示本次启动的日志（如"LLM Proxy 桌面应用启动中..."、"配置加载成功"等）
2. 日志每秒自动刷新，新日志实时出现
3. 向上滚动后，新日志不会强制跳到底部
4. 暂停后日志停止刷新，继续后补齐
5. 清屏后日志清空，但新日志继续追加

- [ ] **步骤 2：验证归档功能**

关闭应用，重新启动，检查 `{dataDir}` 目录：
1. 存在 `llm-proxy-YYYY-MM-DD.log` 归档文件
2. `llm-proxy.log` 是新创建的文件，只包含本次启动的日志

- [ ] **步骤 3：验证清理功能**

手动创建一个过期的归档文件（如 `llm-proxy-2020-01-01.log`），重启应用，确认该文件被自动删除。
