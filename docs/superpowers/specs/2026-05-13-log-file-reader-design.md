# 运行日志：文件读取方案设计

## 背景

当前运行日志采用 RingBuffer + Wails 事件推送架构，但 `slog` 自定义 Handler 在 Wails 环境下拦截不可靠（`multiHandler.Handle()` 未被调用）。改为直接读取日志文件方案。

## 决策记录

| 决策项 | 选择 | 理由 |
|--------|------|------|
| 日志文件区分 | 每次启动创建新文件，旧文件按日期归档 | 避免显示历史启动日志 |
| 归档清理 | 保留最近 N 天（默认 3 天），超期自动删除 | 避免磁盘占用过多 |
| 前端获取方式 | 定时轮询 Go 后端（每秒一次） | 简单可靠 |
| 读取策略 | Go 端维护 offset 增量读取 | 高效，不重复传输 |

## 架构设计

### 数据流

```
slog.Info("xxx") → slog.TextHandler → llm-proxy.log
                                          ↓
前端每秒轮询 → GetNewLogs() → LogReader.ReadNewLogs() → 读取 offset 之后的新内容 → 解析返回
```

### 1. 日志文件管理

**文件命名规则：**
- 当前日志文件：`{dataDir}/llm-proxy.log`（固定名称，slog 始终写入此文件）
- 归档文件：`llm-proxy-2026-05-13.log`（按日期命名）

**启动时归档逻辑：**
1. 检查 `llm-proxy.log` 是否存在且非空
2. 如果存在，将其内容**追加**到 `llm-proxy-{今天日期}.log`（同一天多次启动合并到同一归档文件）
3. 清空或删除 `llm-proxy.log`，让 slog 创建新文件
4. 扫描 `{dataDir}/llm-proxy-*.log` 归档文件，删除超过 N 天的

**自动清理：**
- 启动时执行一次清理
- 清理天数默认 3 天，通过 `LOG_CLEANUP_DAYS` 环境变量配置
- 配置页面可修改此值

### 2. LogReader（新增 `internal/logger/reader.go`）

```go
type LogReader struct {
    filePath string    // 当前日志文件路径
    offset   int64     // 当前读取偏移量
    mu       sync.Mutex
}
```

**方法：**
- `NewLogReader(filePath string) *LogReader`：创建 reader，offset 初始化为 0
- `ReadNewLogs() []LogEntry`：从 offset 位置读取新增内容，按行解析为 LogEntry，更新 offset
- `ReadAllLogs() []LogEntry`：从文件头读取全部内容（用于首次加载），更新 offset 到文件末尾

**解析逻辑：** 复用现有 `parseSlogTextFormat`，按 `\n` 分割后逐行解析。最后一行如果不以 `\n` 结尾，保留在缓冲区中下次继续读取（避免截断不完整的行）。

### 3. App 层 API

| 方法 | 用途 | 调用 LogReader 方法 |
|------|------|---------------------|
| `GetLogHistory() []LogEntryVO` | 页面首次加载 | `ReadAllLogs()` |
| `GetNewLogs() []LogEntryVO` | 定时轮询增量 | `ReadNewLogs()` |

### 4. 前端轮询机制

- 页面初始化时调用 `GetLogHistory()` 加载全部历史
- 之后每秒调用 `GetNewLogs()` 获取增量日志
- **只在日志页面激活时轮询**：页面 `destroy` 时清除定时器，`init` 时启动定时器
- **滚动位置保护**：追加日志时，如果用户不在底部（已向上滚动查看历史），不自动滚动；用户手动滚到底部时恢复自动滚动
- **暂停/继续**：暂停时停止轮询，继续时先调用 `GetNewLogs` 补齐暂停期间的日志
- **清屏**：只清空前端显示数组，不影响后端 offset

### 5. 移除 RingBuffer 和事件推送

删除以下代码：
- `internal/logger/ringbuffer.go` —— 整个文件
- `internal/logger/handler.go` —— `multiHandler` 相关代码
- `logger.go` 中的 `ringBufferWriter`、`parseSlogTextFormat`、`extractAttr`、`extractExtraAttrs` 等解析代码移到 `reader.go`
- `logger.Init` 的 `ringBuf ...*RingBuffer` 参数
- `app.go` 中的 `startLogEventPusher`、`ringBuffer` 字段
- `main.go` 中的 `ringBuffer` 创建和传参
- 前端 `EventsOn('logEvent')` 相关代码

### 6. logger.Init 简化

```go
func Init(logFilePath string, level slog.Level) error {
    // 关闭之前的日志文件
    // 打开/创建日志文件
    // 创建 multiWriter(file + stdout)
    // 创建 slog.TextHandler(writer, opts)
    // slog.SetDefault(slog.New(handler))
}
```

不再需要 RingBuffer 参数，不再需要自定义 handler 或 writer 拦截。

### 7. 配置页面

在 `LOG_CLEANUP_DAYS` 环境变量配置项的描述中标注"日志归档保留天数"，默认值 3。用户可在设置页面修改。
