# LLM Proxy - 大语言模型中转服务

## 项目简介

LLM Proxy 是一个轻量级的大语言模型（LLM）API 中转代理服务，提供统一的 OpenAI 兼容接口，支持多服务商切换、请求日志记录和 Token 使用统计。通过 Web 界面可视化管理 Provider 配置，实现一键切换不同的 LLM 服务商。

## 核心功能

- **OpenAI 兼容代理**：提供标准的 `/v1/chat/completions` 和 `/v1/models` 接口，无缝对接现有应用
- **多服务商管理**：支持配置多个 LLM 服务商（如 DeepSeek、OpenAI、Claude 等），一键切换激活
- **自动模型替换**：代理请求时自动将请求体中的 model 替换为当前激活服务商的模型
- **流式/非流式支持**：完整支持 SSE 流式响应和非流式响应
- **请求日志记录**：详细记录每次请求的输入输出 Token、响应内容、耗时等信息
- **统计报表**：提供 30 天 Token 使用趋势、请求统计等可视化报表
- **数据自动清理**：自动清理 14 天前的请求/响应体数据，节省存储空间

## 适用场景

- **开发测试**：在开发过程中快速切换不同的 LLM 服务商进行测试
- **成本优化**：根据不同场景选择性价比最优的模型
- **统一入口**：为多个应用提供统一的 LLM API 入口，便于管理和监控
- **Token 用量监控**：追踪和分析 API 调用成本

---

## 运行环境

### 操作系统
- Windows 10/11
- macOS 10.15+
- Linux (Ubuntu 18.04+, CentOS 7+)

### 编程语言
- **Go**: 1.21+ (推荐 1.25)

### 数据库
- **MySQL**: 5.7+ 或 8.0+

### 浏览器
- Chrome、Firefox、Edge 等现代浏览器（用于访问 Web 管理界面）

---

## 依赖库

### 核心依赖

| 依赖库 | 版本 | 说明 |
|--------|------|------|
| github.com/gin-gonic/gin | v1.12.0 | Web 框架 |
| gorm.io/gorm | v1.31.1 | ORM 框架 |
| gorm.io/driver/mysql | v1.6.0 | MySQL 驱动 |

### 间接依赖

| 依赖库 | 版本 |
|--------|------|
| github.com/go-sql-driver/mysql | v1.8.1 |
| github.com/go-playground/validator/v10 | v10.30.1 |
| github.com/bytedance/sonic | v1.15.0 |
| golang.org/x/crypto | v0.48.0 |

---

## 安装步骤

### 1. 克隆项目

```bash
git clone <repository-url>
cd llm_statistic
```

### 2. 安装依赖

```bash
go mod download
```

### 3. 创建数据库

```sql
CREATE DATABASE llm_proxy CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

### 4. 配置环境变量（可选）

创建 `.env` 文件或设置环境变量：

```bash
# 数据库配置
export DB_HOST=localhost
export DB_PORT=3306
export DB_USER=root
export DB_PASSWORD=your_password
export DB_NAME=llm_proxy

# 服务端口
export WEB_PORT=80
export PROXY_PORT=8888
```

### 5. 编译运行

```bash
# 编译
go build -o server.exe ./cmd/server

# 运行
./server.exe
```

或直接运行：

```bash
go run ./cmd/server
```

---

## 配置说明

### 默认配置

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| DB_HOST | localhost | 数据库主机 |
| DB_PORT | 3306 | 数据库端口 |
| DB_USER | root | 数据库用户名 |
| DB_PASSWORD | wang | 数据库密码 |
| DB_NAME | llm_proxy | 数据库名称 |
| WEB_PORT | 80 | Web 管理界面端口 |
| PROXY_PORT | 8888 | 代理服务端口 |

### 环境变量优先级

程序会优先读取环境变量，若未设置则使用默认值。

---

## 使用指南

### 启动服务

启动后会自动打开浏览器访问管理界面：

```
http://localhost
```

### 访问地址

- **管理界面**: http://localhost
- **统计报表**: http://localhost/stats
- **代理接口**: http://localhost:8888/v1/chat/completions
- **模型列表**: http://localhost:8888/v1/models
- **健康检查**: http://localhost:8888/health

### 添加 Provider

1. 点击右上角「添加配置」按钮
2. 填写配置信息：
   - **名称**: 服务商标识名称（如 DeepSeek）
   - **Base URL**: API 端点地址（如 `https://api.deepseek.com/v1/chat/completions`）
   - **API Key**: 服务商提供的 API 密钥
   - **模型**: 模型名称（如 `deepseek-chat`）
3. 点击「保存」

### 切换激活模型

- 在 Provider 列表中，点击目标配置的开关即可激活
- 系统保证同一时间只有一个 Provider 处于激活状态
- 激活新的 Provider 会自动禁用其他所有 Provider

### 调用代理接口

使用 OpenAI SDK 或任何兼容 OpenAI API 的客户端：

```python
from openai import OpenAI

client = OpenAI(
    api_key="any-key",  # 代理服务不验证 API Key
    base_url="http://localhost:8888/v1"
)

response = client.chat.completions.create(
    model="any-model",  # 会自动替换为当前激活的模型
    messages=[
        {"role": "user", "content": "你好"}
    ],
    stream=True  # 支持流式响应
)

for chunk in response:
    print(chunk.choices[0].delta.content, end="")
```

### 查看统计

访问 http://localhost/stats 查看：
- 总请求数、成功率
- Token 使用总量（输入/输出/缓存）
- 30 天使用趋势图
- 最近请求日志

---

## 项目结构

```
llm_statistic/
├── cmd/
│   └── server/
│       └── main.go          # 程序入口
├── internal/
│   ├── config/
│   │   └── config.go        # 配置管理
│   ├── handler/
│   │   ├── proxy_handler.go # 代理请求处理
│   │   └── web_handler.go   # Web API 处理
│   ├── middleware/
│   │   └── cors.go          # CORS 中间件
│   ├── model/
│   │   └── models.go        # 数据模型
│   ├── repository/
│   │   ├── provider_repo.go # Provider 数据访问
│   │   └── request_log_repo.go # 日志数据访问
│   └── service/
│       ├── provider_service.go  # Provider 业务逻辑
│       ├── proxy_service.go     # 代理业务逻辑
│       └── stats_service.go     # 统计业务逻辑
├── web/
│   ├── static/
│   │   ├── css/             # 样式文件
│   │   └── js/              # JavaScript 文件
│   └── templates/
│       ├── index.html       # 配置管理页面
│       └── stats.html       # 统计报表页面
├── go.mod
├── go.sum
└── README.md
```

---

## API 接口

### Provider 管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/providers | 获取所有 Provider |
| GET | /api/providers/:id | 获取单个 Provider |
| POST | /api/providers | 创建 Provider |
| PUT | /api/providers/:id | 更新 Provider |
| DELETE | /api/providers/:id | 删除 Provider |
| POST | /api/providers/:id/toggle | 切换激活状态 |

### 统计接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/stats | 获取仪表盘统计 |
| GET | /api/stats/daily | 获取 30 天统计 |
| GET | /api/logs/recent | 获取最近日志 |

### 代理接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /v1/chat/completions | Chat Completions API |
| GET | /v1/models | 获取可用模型列表 |
| GET | /health | 健康检查 |

---

## 日志文件

程序运行时会生成以下日志文件：

| 文件 | 说明 |
|------|------|
| llm-proxy.log | 服务运行日志 |
| proxy-requests.log | 代理请求日志 |
| proxy-reqbody.log | 请求体日志 |

日志文件会自动清理 14 天前的记录。

---

## 常见问题

### Q: 数据库连接失败？

检查：
1. MySQL 服务是否已启动
2. 数据库 `llm_proxy` 是否已创建
3. 用户名密码是否正确
4. 防火墙是否允许 3306 端口

### Q: 端口被占用？

修改环境变量 `WEB_PORT` 和 `PROXY_PORT` 为其他端口。

### Q: 如何查看详细错误？

查看 `llm-proxy.log` 日志文件获取详细错误信息。

---

## 许可证

MIT License
