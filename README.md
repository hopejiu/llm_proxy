# LLM Proxy - 大语言模型中转服务

## 项目简介

LLM Proxy 是一个轻量级的大语言模型（LLM）API 中转代理服务，同时提供 **OpenAI**、**Anthropic**、**Ollama** 三种兼容接口，支持多服务商管理、模型别名路由、请求日志记录和 Token 使用统计。通过 Web 界面可视化管理 Provider 配置，实现一键切换不同的 LLM 服务商。

## 核心功能

- **多协议兼容代理**：同时提供 OpenAI、Anthropic、Ollama 三种 API 接口，无缝对接各类应用
- **多服务商管理**：支持配置多个 LLM 服务商（如 DeepSeek、OpenAI、Claude 等），一键切换激活
- **模型别名路由**：支持为 Provider 设置别名，请求时按模型名自动路由到对应 Provider
- **自动模型替换**：代理请求时自动将请求体中的 model 替换为当前激活服务商的模型
- **自定义请求参数**：支持通过 ExtraParams 注入额外请求参数（如 temperature、max_tokens 等）
- **流式/非流式支持**：完整支持 SSE 流式响应和非流式响应，流式自动注入 `stream_options` 确保 usage 返回
- **请求日志记录**：详细记录每次请求的输入输出 Token、缓存 Token、推理内容、响应内容、耗时等
- **统计报表**：提供 30 天 Token 使用趋势、今日分时统计、请求统计等可视化报表
- **统计体验优化**：骨架屏加载、错误 Toast 提示、环比趋势指标、图表维度切换、日志详情元信息、手动刷新等
- **Provider 导入导出**：支持 JSON 格式批量导入导出 Provider 配置
- **CodeBuddy 集成**：一键配置 CodeBuddy 的 models.json，快速接入代理服务
- **双数据库支持**：支持 MySQL 和 SQLite，SQLite 模式零依赖开箱即用
- **数据自动清理**：自动汇总和清理过期请求记录，汇总表永久保留统计数据
- **性能优化**：SQLite WAL 模式、数据库索引、汇总表加速统计查询

## 适用场景

- **开发测试**：在开发过程中快速切换不同的 LLM 服务商进行测试
- **成本优化**：根据不同场景选择性价比最优的模型
- **统一入口**：为多个应用提供统一的 LLM API 入口，便于管理和监控
- **Token 用量监控**：追踪和分析 API 调用成本
- **多协议适配**：同一服务同时支持 OpenAI、Anthropic、Ollama 客户端接入

---

## 运行环境

### 操作系统

- Windows 10/11
- macOS 10.15+
- Linux (Ubuntu 18.04+, CentOS 7+)

### 编程语言

- **Go**: 1.25+

### 数据库（二选一）

- **MySQL**: 5.7+ 或 8.0+
- **SQLite**: 无需额外安装（默认使用 MySQL，通过 `DB_TYPE=sqlite` 切换）

### 浏览器

- Chrome、Firefox、Edge 等现代浏览器（用于访问 Web 管理界面）

---

## 依赖库

### 核心依赖

| 依赖库                     | 版本    | 说明                          |
| -------------------------- | ------- | ----------------------------- |
| github.com/gin-gonic/gin   | v1.12.0 | Web 框架                      |
| gorm.io/gorm               | v1.31.1 | ORM 框架                      |
| gorm.io/driver/mysql       | v1.6.0  | MySQL 驱动                    |
| github.com/glebarez/sqlite | v1.11.0 | 纯 Go SQLite 驱动（无需 CGO） |
| github.com/joho/godotenv   | v1.5.1  | .env 文件加载                 |

---

## 快速开始

### 克隆项目

```bash
git clone <repository-url>
cd llm_statistic
```

bash### 安装依赖

```bash
go mod download
```

bash### 配置环境变量

创建 `.env` 文件或设置环境变量：

**MySQL 模式（默认）：**

```bash
DB_TYPE=mysql
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=your_password
DB_NAME=llm_proxy
```

bash**SQLite 模式（零依赖）：**

```bash
DB_TYPE=sqlite
DB_PATH=llm_proxy.db
```

bash### 编译运行

```bash
# 编译
go build -o server.exe ./cmd/server

# 运行
./server.exe
```

bash或直接运行：

```bash
go run ./cmd/server
```

bash---

## 自动构建

项目使用 GitHub Actions 实现自动编译和发布。当推送 `v*` 格式的 tag 时，会自动构建 Windows 和 Linux 二进制文件并发布到 GitHub Release。

### 触发方式

```bash
git tag v1.0.0
git push origin v1.0.0
```

bash### 构建产物

| 平台    | 架构  | 文件名                           |
| ------- | ----- | -------------------------------- |
| Windows | amd64 | `llm-proxy-windows-amd64.zip`  |
| Linux   | amd64 | `llm-proxy-linux-amd64.tar.gz` |

压缩包内包含二进制文件和 `web` 目录，解压后即可运行。

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
| WebPort                | `WEB_PORT`                  | `80`           | Web 管理界面端口           |
| ProxyPort              | `PROXY_PORT`                | `8888`         | 代理服务端口               |
| HTTPTimeout            | `HTTP_TIMEOUT`              | `300s`         | HTTP 请求超时              |
| StreamFirstByteTimeout | `STREAM_FIRST_BYTE_TIMEOUT` | `5s`           | 流式首次数据超时           |
| StreamMaxRetries       | `STREAM_MAX_RETRIES`        | `10`           | 流式最大重试次数           |
| RetryDelayBase         | `RETRY_DELAY_BASE`          | `500ms`        | 重试延迟基数               |
| ProviderCacheTTL       | `PROVIDER_CACHE_TTL`        | `30s`          | Provider 缓存过期时间      |
| LogCleanupDays         | `LOG_CLEANUP_DAYS`          | `14`           | 日志清理天数               |

### 环境变量优先级

程序会优先读取环境变量，若未设置则使用默认值。支持 `.env` 文件。

---

## 使用指南

### 启动服务

启动后会自动打开浏览器访问管理界面：

```
http://localhost
```

language### 访问地址

| 服务               | 地址                                            |
| ------------------ | ----------------------------------------------- |
| 管理界面           | `http://localhost`                            |
| 统计报表           | `http://localhost/stats`                      |
| OpenAI 代理        | `http://localhost:8888/v1/chat/completions`   |
| OpenAI 模型列表    | `http://localhost:8888/v1/models`             |
| Anthropic 代理     | `http://localhost:8888/anthropic/v1/messages` |
| Anthropic 模型列表 | `http://localhost:8888/anthropic/v1/models`   |
| Ollama 代理        | `http://localhost:8888/api/chat`              |
| Ollama 标签列表    | `http://localhost:8888/api/tags`              |
| 健康检查           | `http://localhost:8888/health`                |

### 添加 Provider

- 点击右上角「添加配置」按钮
- 填写配置信息：
  - **名称**：服务商标识名称（如 DeepSeek）
  - **Base URL**：API 端点地址（如 `https://api.deepseek.com/v1/chat/completions`）
  - **API Key**：服务商提供的 API 密钥
  - **模型**：模型名称（如 `deepseek-chat`）
  - **别名**：模型别名，多个用逗号分隔，用于请求时按模型名路由匹配
  - **自动后缀**：开启后可配置 URL 后缀，自动拼接到 Base URL
  - **额外参数**：自定义请求参数 JSON，会合并到请求体中
- 点击「保存」

### 切换激活模型

- 在 Provider 列表中，点击目标配置的开关即可激活
- 系统保证同一时间只有一个 Provider 处于激活状态
- 激活新的 Provider 会自动禁用其他所有 Provider

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

python**Anthropic 兼容：**

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

python**Ollama 兼容：**

```bash
curl http://localhost:8888/api/chat -d '{
  "model": "llama3",
  "messages": [{"role": "user", "content": "你好"}]
}'
```

bash### 查看统计

访问 `http://localhost/stats` 查看：

- 概览卡片：今日/本周/总计 Token 用量，含环比趋势指标和 Tokens 单位标注
- 30 天趋势图：支持 Token 用量/请求数维度切换
- 今日分时统计：仅显示到当前小时，避免未来空白噪音
- 最近请求日志：点击可查看完整详情（含时间、模型、Provider、耗时、Token/s 等元信息）
- 手动刷新按钮：一键刷新所有数据，带旋转动画反馈

---

## 项目结构

```
llm_statistic/
├── cmd/
│   └── server/
│       └── main.go              # 程序入口
├── internal/
│   ├── config/
│   │   └── config.go            # 配置管理
│   ├── converter/
│   │   ├── anthropic_converter.go  # Anthropic <-> OpenAI 协议转换
│   │   ├── ollama_converter.go     # Ollama <-> OpenAI 协议转换
│   │   ├── openai_types.go         # OpenAI 类型定义
│   │   └── usage.go                # Token 用量提取
│   ├── handler/
│   │   ├── base_handler.go      # 公共基类 Handler
│   │   ├── proxy_handler.go     # OpenAI 兼容代理
│   │   ├── anthropic_handler.go # Anthropic 兼容代理
│   │   ├── ollama_handler.go    # Ollama 兼容代理
│   │   ├── web_handler.go       # Web 管理 API
│   │   └── response.go          # 统一响应格式
│   ├── logger/
│   │   └── logger.go            # 日志初始化
│   ├── middleware/
│   │   └── cors.go              # CORS 中间件
│   ├── model/
│   │   ├── models.go            # 核心模型（ProviderConfig, RequestLog, TokenStats, HourlyStat）
│   │   ├── anthropic_models.go  # Anthropic 请求/响应模型
│   │   └── ollama_models.go     # Ollama 请求/响应模型
│   ├── repository/
│   │   ├── provider_repo.go     # Provider 数据访问
│   │   ├── request_log_repo.go  # 日志数据访问 + 统计查询
│   │   └── hourly_stat_repo.go  # 汇总表数据访问 + 汇总统计查询
│   ├── router/
│   │   └── router.go            # 路由定义
│   └── service/
│       ├── proxy_service.go     # 代理核心逻辑 + Provider 缓存
│       ├── provider_service.go  # Provider 管理
│       ├── stats_service.go     # 统计服务（汇总表+明细表混合查询）
│       └── cleanup_service.go   # 定时汇总和清理服务
├── web/
│   ├── static/
│   │   ├── css/
│   │   │   └── style.css          # 全局样式（含骨架屏、趋势指标、维度切换等）
│   │   └── js/
│   │       ├── common.js          # 公共工具函数
│   │       ├── stats.js           # 统计页面逻辑
│   │       └── echarts.min.js     # ECharts 图表库
│   └── templates/
│       ├── index.html             # 配置管理页面
│       └── stats.html             # 统计报表页面
├── .github/workflows/
│   └── release.yml              # GitHub Actions 自动发布
├── build.ps1                    # Windows 构建脚本
├── build.sh                     # Linux 构建脚本
├── go.mod
├── go.sum
└── README.md
```

language---

## API 接口

### Provider 管理

| 方法   | 路径                  | 说明                   |
| ------ | --------------------- | ---------------------- |
| GET    | /api/providers        | 获取所有 Provider      |
| GET    | /api/providers/:id    | 获取单个 Provider      |
| POST   | /api/providers        | 创建 Provider          |
| PUT    | /api/providers/:id    | 更新 Provider          |
| DELETE | /api/providers/:id    | 删除 Provider          |
| GET    | /api/providers/export | 导出所有 Provider 配置 |
| POST   | /api/providers/import | 导入 Provider 配置     |
| POST   | /api/codebuddy/setup  | 一键配置 CodeBuddy     |

### 统计接口

| 方法 | 路径              | 说明             |
| ---- | ----------------- | ---------------- |
| GET  | /api/stats        | 获取仪表盘统计   |
| GET  | /api/stats/daily  | 获取 30 天统计   |
| GET  | /api/stats/hourly | 获取今日分时统计 |
| GET  | /api/logs/recent  | 获取最近日志     |
| GET  | /api/logs/:id     | 获取日志详情     |

### 代理接口

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

## 日志文件

程序运行时会生成以下日志文件：

| 文件               | 说明         |
| ------------------ | ------------ |
| llm-proxy.log      | 服务运行日志 |
| proxy-requests.log | 代理请求日志 |
| proxy-reqbody.log  | 请求体日志   |

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

修改环境变量 `WEB_PORT` 和 `PROXY_PORT` 为其他端口。

### Q: 如何查看详细错误？

查看 `llm-proxy.log` 日志文件获取详细错误信息。

### Q: 如何使用 SQLite？

设置环境变量 `DB_TYPE=sqlite`，可选配置 `DB_PATH` 指定数据库文件路径，默认为 `llm_proxy.db`。

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

### 查询优化

- `GetRecent` 排除 longtext 大字段，只查询摘要信息
- `GetDashboardStats` 从汇总表一次查询获取今日/本周/总计
- 日志详情通过 `GetByID` 单独获取完整数据

### 定时清理

- **每小时**：汇总上一小时的明细数据到 `hourly_stats`，汇总后标记明细记录 `aggregated=true`
- **每天**：删除已汇总且超过 `LOG_CLEANUP_DAYS` 天的明细记录
- 安全边界：只删除已确认汇总完成的记录
- 防重复计数：`aggregated` 行级标记确保同一条明细只被汇总一次

---

## 许可证

MIT License
