# LLM Proxy - 大语言模型中转服务

## 项目简介

LLM Proxy 是一个轻量级的大语言模型（LLM）API 中转代理服务，基于 **Wails** 构建为 Windows 桌面应用，同时提供 **OpenAI**、**Anthropic**、**Ollama** 三种兼容接口，支持多服务商管理、模型别名路由、请求日志记录和 Token 使用统计。通过桌面界面可视化管理 Provider 配置，实现一键切换不同的 LLM 服务商。

## 核心功能

- **桌面应用**：基于 Wails 构建的 Windows 原生桌面应用，内置 WebView2 前端界面，无需浏览器即可管理
- **多协议兼容代理**：同时提供 OpenAI、Anthropic、Ollama 三种 API 接口，无缝对接各类应用
- **多服务商管理**：支持配置多个 LLM 服务商（如 DeepSeek、OpenAI、Claude 等），按模型名自动路由
- **模型别名路由**：支持为 Provider 设置别名，请求时按模型名自动路由到对应 Provider
- **自动模型替换**：代理请求时自动将请求体中的 model 替换为当前激活服务商的模型
- **自定义请求参数**：支持通过 ExtraParams 注入额外请求参数（如 temperature、max_tokens 等）
- **流式/非流式支持**：完整支持 SSE 流式响应和非流式响应，流式自动注入 `stream_options` 确保 usage 返回
- **流式超时保护**：首字节超时检测 + 流中途卡住超时检测，自动重试，防止请求挂起
- **请求日志记录**：详细记录每次请求的输入输出 Token、缓存 Token、推理内容、响应内容、耗时等
- **实时请求监控**：实时查看活跃请求列表，包括请求状态、已返回内容、工具调用等
- **统计报表**：提供 30 天 Token 使用趋势、今日分时统计、请求统计等可视化报表
- **Provider 导入导出**：支持 JSON 格式批量导入导出 Provider 配置
- **CodeBuddy 集成**：一键配置 CodeBuddy 的 models.json，快速接入代理服务
- **双数据库支持**：支持 MySQL 和 SQLite，SQLite 模式零依赖开箱即用
- **数据自动清理**：自动汇总和清理过期请求记录，汇总表永久保留统计数据
- **运行日志查看**：应用内实时查看运行日志，支持日志级别过滤

## 适用场景

- **开发测试**：在开发过程中快速切换不同的 LLM 服务商进行测试
- **成本优化**：根据不同场景选择性价比最优的模型
- **统一入口**：为多个应用提供统一的 LLM API 入口，便于管理和监控
- **Token 用量监控**：追踪和分析 API 调用成本
- **多协议适配**：同一服务同时支持 OpenAI、Anthropic、Ollama 客户端接入

---

## 运行环境

### 操作系统

- Windows 10/11（当前主要支持平台）

### 编程语言

- **Go**: 1.25+
- **Node.js**: 16+（前端构建所需）

### 数据库（二选一）

- **MySQL**: 5.7+ 或 8.0+
- **SQLite**: 无需额外安装（默认使用 MySQL，通过 `DB_TYPE=sqlite` 切换）

---

## 依赖库

### 核心依赖

| 依赖库                       | 版本      | 说明                            |
| ---------------------------- | --------- | ------------------------------- |
| github.com/wailsapp/wails/v2 | v2.12.0   | 桌面应用框架（Go + WebView2）   |
| github.com/gin-gonic/gin     | v1.12.0   | HTTP 框架（代理服务路由）       |
| gorm.io/gorm                 | v1.31.1   | ORM 框架                        |
| gorm.io/driver/mysql         | v1.6.0    | MySQL 驱动                      |
| github.com/gleberez/sqlite   | v1.11.0   | 纯 Go SQLite 驱动（无需 CGO）   |
| github.com/joho/godotenv     | v1.5.1    | .env 文件加载                   |

### 前端

| 依赖库   | 版本    | 说明          |
| -------- | ------- | ------------- |
| Vite     | ^3.0.7  | 前端构建工具  |

---

## 快速开始

### 克隆项目

```bash
git clone <repository-url>
cd llm_statistic
```

### 安装依赖

```bash
# Go 依赖
go mod download

# 前端依赖
cd frontend && npm install && cd ..
```

### 配置环境变量

在 `%APPDATA%/llm-proxy/` 目录下创建 `.env` 文件，或在工作目录下创建：

**MySQL 模式（默认）：**

```bash
DB_TYPE=mysql
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=your_password
DB_NAME=llm_proxy
```

**SQLite 模式（零依赖）：**

```bash
DB_TYPE=sqlite
DB_PATH=llm_proxy.db
```

### 开发模式运行

```bash
wails dev
```

### 构建发布

```bash
# 使用构建脚本（推荐）
.\build.ps1

# 或直接使用 Wails CLI
wails build -ldflags="-s -w"
```

构建产物输出到 `dist/` 目录，包含 `llm-proxy.exe` 可执行文件。

---

## 自动构建

项目使用 GitHub Actions 实现自动编译和发布。当推送 `v*` 格式的 tag 时，会自动构建 Windows 二进制文件并发布到 GitHub Release。

### 触发方式

```bash
git tag v1.0.0
git push origin v1.0.0
```

---

## 配置说明

### 全部配置项

| 配置项                 | 环境变量                      | 默认值           | 说明                       |
| ---------------------- | ----------------------------- | ---------------- | -------------------------- |
| DBType                 | `DB_TYPE`                   | `mysql`        | 数据库类型（mysql/sqlite） |
| DBPath                 | `DB_PATH`                   | `llm_proxy.db` | SQLite 数据库文件路径      |
| DBHost                 | `DB_HOST`                   | `localhost`    | MySQL 主机                 |
| DBPort                 | `DB_PORT`                   | `3306`         | MySQL 端口                 |
| DBUser                 | `DB_USER`                   | `root`         | MySQL 用户名               |
| DBPassword             | `DB_PASSWORD`               | （空）           | MySQL 密码                 |
| DBName                 | `DB_NAME`                   | `llm_proxy`    | 数据库名                   |
| ProxyPort              | `PROXY_PORT`                | `8888`         | 代理服务端口               |
| HTTPTimeout            | `HTTP_TIMEOUT`              | `300s`         | HTTP 请求超时              |
| StreamFirstByteTimeout | `STREAM_FIRST_BYTE_TIMEOUT` | `5s`           | 流式首次数据超时           |
| StreamMaxRetries       | `STREAM_MAX_RETRIES`        | `10`           | 流式最大重试次数           |
| RetryDelayBase         | `RETRY_DELAY_BASE`          | `500ms`        | 重试延迟基数               |
| ProviderCacheTTL       | `PROVIDER_CACHE_TTL`        | `30s`          | Provider 缓存过期时间      |
| LogCleanupDays         | `LOG_CLEANUP_DAYS`          | `14`           | 日志清理天数               |
| LogLevel               | `LOG_LEVEL`                 | `info`         | 日志级别（debug/info/warn/error） |
| AutoStartProxy         | `AUTO_START_PROXY`          | `true`         | 启动时是否自动启动代理服务 |

### 环境变量优先级

程序会优先读取环境变量，若未设置则使用默认值。`.env` 文件从 `%APPDATA%/llm-proxy/.env` 和当前工作目录加载，环境变量优先于 `.env` 文件。

---

## 使用指南

### 启动服务

启动桌面应用后，可在界面中点击「启动代理」按钮启动代理服务（默认自动启动）。

### 访问地址

| 服务               | 地址                                            |
| ------------------ | ----------------------------------------------- |
| OpenAI 代理        | `http://localhost:8888/v1/chat/completions`   |
| OpenAI 模型列表    | `http://localhost:8888/v1/models`             |
| Anthropic 代理     | `http://localhost:8888/anthropic/v1/messages` |
| Anthropic 模型列表 | `http://localhost:8888/anthropic/v1/models`   |
| Ollama 代理        | `http://localhost:8888/api/chat`              |
| Ollama 标签列表    | `http://localhost:8888/api/tags`              |
| 健康检查           | `http://localhost:8888/health`                |

### 添加 Provider

- 在「Provider 管理」页面点击「添加配置」按钮
- 填写配置信息：
  - **名称**：服务商标识名称（如 DeepSeek）
  - **Base URL**：API 端点地址（如 `https://api.deepseek.com/v1/chat/completions`）
  - **API Key**：服务商提供的 API 密钥
  - **模型**：模型名称（如 `deepseek-chat`）
  - **别名**：模型别名，多个用逗号分隔，用于请求时按模型名路由匹配
  - **自动后缀**：开启后可配置 URL 后缀，自动拼接到 Base URL
  - **额外参数**：自定义请求参数 JSON，会合并到请求体中
- 点击「保存」

### 调用代理接口

**OpenAI 兼容：**

```python
from openai import OpenAI

client = OpenAI(
    api_key="any-key",
    base_url="http://localhost:8888/v1"
)

response = client.chat.completions.create(
    model="deepseek-chat",
    messages=[{"role": "user", "content": "你好"}],
    stream=True
)
```

**Anthropic 兼容：**

```python
import anthropic

client = anthropic.Anthropic(
    api_key="any-key",
    base_url="http://localhost:8888/anthropic"
)

response = client.messages.create(
    model="claude-3-sonnet",
    max_tokens=1024,
    messages=[{"role": "user", "content": "你好"}]
)
```

**Ollama 兼容：**

```bash
curl http://localhost:8888/api/chat -d '{
  "model": "llama3",
  "messages": [{"role": "user", "content": "你好"}]
}'
```

---

## 项目结构

```
llm_statistic/
├── main.go                      # 程序入口（Wails 应用）
├── app.go                       # App 核心逻辑（Wails 绑定层）
├── app_error.go                 # 统一错误类型
├── vo.go                        # 视图对象（前端数据转换）
├── internal/
│   ├── config/
│   │   └── config.go            # 配置管理（.env 加载、校验、持久化）
│   ├── converter/
│   │   ├── anthropic_converter.go  # Anthropic <-> OpenAI 协议转换
│   │   ├── ollama_converter.go     # Ollama <-> OpenAI 协议转换
│   │   ├── openai_types.go         # OpenAI 类型定义
│   │   └── usage.go                # Token 用量提取
│   ├── handler/
│   │   ├── base_handler.go      # 公共基类 Handler（流式超时、重试、日志）
│   │   ├── proxy_handler.go     # OpenAI 兼容代理
│   │   ├── anthropic_handler.go # Anthropic 兼容代理
│   │   ├── anthropic_stream.go  # Anthropic 流式状态机
│   │   ├── ollama_handler.go    # Ollama 兼容代理
│   │   ├── active_tracker.go   # 活跃请求追踪器
│   │   ├── stream_tracker.go   # 流式内容追踪（工具调用等）
│   │   ├── web_handler.go      # Web 管理 API
│   │   └── response.go          # 统一响应格式
│   ├── logger/
│   │   ├── logger.go            # 日志初始化
│   │   ├── handler.go           # slog Handler（文件 + RingBuffer）
│   │   └── ringbuffer.go        # 环形缓冲区（实时日志推送）
│   ├── middleware/
│   │   └── cors.go              # CORS 中间件
│   ├── model/
│   │   ├── models.go            # 核心模型（ProviderConfig, RequestLog, HourlyStat）
│   │   ├── anthropic_models.go  # Anthropic 请求/响应模型
│   │   └── ollama_models.go     # Ollama 请求/响应模型
│   ├── repository/
│   │   ├── provider_repo.go     # Provider 数据访问
│   │   ├── request_log_repo.go  # 日志数据访问 + 统计查询
│   │   └── hourly_stat_repo.go  # 汇总表数据访问 + 汇总统计查询
│   ├── router/
│   │   └── router.go            # 代理服务路由定义
│   └── service/
│       ├── proxy_service.go     # 代理核心逻辑 + Provider 缓存
│       ├── provider_service.go  # Provider 管理
│       ├── stats_service.go     # 统计服务（汇总表+明细表混合查询）
│       └── cleanup_service.go   # 定时汇总和清理服务
├── frontend/
│   ├── src/
│   │   ├── main.js              # 前端入口
│   │   ├── common.js            # 公共工具函数
│   │   ├── style.css            # 全局样式
│   │   ├── pages/
│   │   │   ├── providers.html/js  # Provider 管理页面
│   │   │   ├── stats.html/js      # 统计报表页面
│   │   │   ├── logs.html/js       # 请求日志页面
│   │   │   ├── realtime.html/js   # 实时监控页面
│   │   │   └── settings.html/js   # 设置页面
│   │   ├── lib/                 # 第三方库
│   │   └── assets/              # 静态资源
│   ├── index.html               # Vite 入口
│   └── package.json
├── build.ps1                    # Windows 构建脚本
├── build.sh                     # Linux 构建脚本
├── wails.json                   # Wails 配置
├── .github/workflows/
│   └── release.yml              # GitHub Actions 自动发布
├── go.mod
├── go.sum
└── README.md
```

---

## 代理接口

| 方法 | 路径                   | 说明                    |
| ---- | ---------------------- | ----------------------- |
| POST | /v1/chat/completions   | OpenAI Chat Completions |
| GET  | /v1/models             | OpenAI 模型列表         |
| POST | /anthropic/v1/messages | Anthropic Messages      |
| GET  | /anthropic/v1/models   | Anthropic 模型列表      |
| POST | /api/chat              | Ollama Chat             |
| GET  | /api/tags              | Ollama 标签列表         |
| GET  | /health                | 健康检查                |

---

## 流式请求可靠性

代理服务对流式请求提供了多层保护机制：

| 机制                 | 说明                                                                 |
| -------------------- | -------------------------------------------------------------------- |
| 首字节超时检测       | 连接建立后若在 `STREAM_FIRST_BYTE_TIMEOUT` 内未收到首字节，自动重试 |
| 流中途卡住超时       | 流数据传输中若长时间无新数据（超过 `HTTP_TIMEOUT`），终止请求并报错  |
| 自动重试             | 首字节超时或请求失败时，按指数退避重试，最多 `STREAM_MAX_RETRIES` 次 |
| [DONE] 保证          | 流结束时确保向客户端发送 `data: [DONE]` 标记，防止客户端挂起等待     |
| goroutine 泄漏防护   | 超时时通过 context cancel 终止读取 goroutine，避免资源泄漏            |

---

## 日志文件

程序运行时在 `%APPDATA%/llm-proxy/` 目录下生成以下文件：

| 文件               | 说明         |
| ------------------ | ------------ |
| llm-proxy.log      | 服务运行日志 |
| llm_proxy.db       | SQLite 数据库（SQLite 模式） |
| .env               | 配置文件     |

日志文件会自动清理超过 `LOG_CLEANUP_DAYS` 天的记录。

---

## 常见问题

### Q: 数据库连接失败？

检查：

- MySQL 服务是否已启动
- 数据库 `llm_proxy` 是否已创建
- 用户名密码是否正确
- 防火墙是否允许 3306 端口
- 或切换为 SQLite 模式：设置 `DB_TYPE=sqlite`

### Q: 端口被占用？

修改环境变量 `PROXY_PORT` 为其他端口。

### Q: 如何查看详细错误？

- 应用内「设置」页面查看运行日志
- 或查看 `%APPDATA%/llm-proxy/llm-proxy.log` 日志文件

### Q: 如何使用 SQLite？

设置环境变量 `DB_TYPE=sqlite`，可选配置 `DB_PATH` 指定数据库文件路径，默认为 `llm_proxy.db`。

### Q: 流式请求卡住不返回？

- 检查 `STREAM_FIRST_BYTE_TIMEOUT` 是否过短（默认 5s）
- 检查 `HTTP_TIMEOUT` 是否足够（默认 300s，即 5 分钟）
- 查看日志中是否有 "stream首次数据等待超时" 或 "stream中途卡住超时" 的警告

---

## 性能优化

项目针对 SQLite 和 MySQL 双数据库模式进行了多层性能优化：

### 数据库连接优化

| 优化项       | SQLite           | MySQL                              |
| ------------ | ---------------- | ---------------------------------- |
| WAL 模式     | 启用（读写并发） | -                                  |
| busy_timeout | 5秒              | -                                  |
| synchronous  | NORMAL           | -                                  |
| cache_size   | 64MB             | -                                  |
| 连接池       | MaxOpenConns=1   | MaxOpenConns=25, MaxIdleConns=10   |
| 参数优化     | -                | interpolateParams=true, timeout=5s |

### 汇总表（hourly_stats）

核心优化：新增 `hourly_stats` 汇总表，按小时聚合 Token 统计数据。

- **统计查询**从汇总表读取（一天仅24条记录），不再扫描明细表
- **实时性保证**：统计查询 = 汇总表（历史已完成小时）+ 明细表（当前小时实时聚合）
- **明细表可安全删除**：旧记录删除后统计数据仍保留在汇总表中
- **启动时自动回填**：服务启动时检查并补全缺失的历史汇总数据

### 索引优化

`request_logs` 表添加以下索引：

| 索引名                             | 字段                 | 覆盖查询                |
| ---------------------------------- | -------------------- | ----------------------- |
| idx_request_logs_created_at_status | (created_at, status) | 统计查询、清理操作      |
| idx_request_logs_created_at        | (created_at)         | 排序、范围查询          |
| idx_request_logs_provider_id       | (provider_id)        | Provider 删除时关联更新 |

### 定时清理

- **每小时**：汇总上一小时的明细数据到 `hourly_stats`，汇总后标记明细记录 `aggregated=true`
- **每天**：删除已汇总且超过 `LOG_CLEANUP_DAYS` 天的明细记录
- 安全边界：只删除已确认汇总完成的记录
- 防重复计数：`aggregated` 行级标记确保同一条明细只被汇总一次

---

## 许可证

MIT License
