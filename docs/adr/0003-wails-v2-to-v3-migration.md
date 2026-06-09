# ADR-0003: Wails v2 → v3 平台迁移方案

Wails v3 发布了 alpha 版本，引入了服务架构、新的事件系统、Taskfile 构建系统和 `@wailsio/runtime`。当前项目构建在 Wails v2 上，需要迁移到 v3 以获得长期支持和更好的开发体验。

## 迁移策略

采用 **分层移植 (Layered Porting)** 策略：`internal/` 目录下的 31 个 Go 业务文件（model/repository/service/handler/converter/config/logger）零修改搬迁到 v3 项目结构。只有 Wails 胶水层（`main.go`、`app.go`、`vo.go`、`app_error.go`）需要重写。

## Go 后端

- **服务拆分**: 将一个单一大 `App` 结构体拆分为 3 个 Wails 绑定服务 —— `ProviderService`、`StatsService`、`AppService`
- **Wails 绑定服务 → 业务服务**: 绑定服务作为薄适配层，调用 `internal/service/` 的业务逻辑，执行 VO 转换和错误包装
- **依赖注入**: 在 `main.go` 中手动构造所有依赖并显式注入到服务构造函数（与 v2 现有 DI 风格一致）
- **生命周期**: 代理启动失败时降级不阻塞 UI（log + 状态标记，不返回 error）；CleanupService 自管理 goroutine（通过 `ServiceStartup`/`ServiceShutdown` 接口）

## 前端

- **技术栈**: React 18 + TypeScript + `@wailsio/runtime`
- **路由**: `react-router-dom` (`/providers`, `/stats`, `/logs`, `/realtime`, `/settings`)
- **UI**: Tailwind CSS + shadcn/ui（基于 Radix，copy-paste 模式）
- **数据获取**: 自定义 hooks 封装（`useProviders`、`useStats`、`useProxyStatus` 等）
- **全局状态**: 单 React Context（代理状态 + 回退消息 + Provider 列表刷新信号）

## 被否决的方案

- **单一大 App 服务**: 不符合 v3 服务化理念，绑定方法组织混乱（否决理由：架构清晰度）
- **React Query**: 对桌面应用过重，简单的 hooks 模式就够了（否决理由：复杂度）
- **antd/shadcn 组件库**: antd 包体大且样式固化（否决理由：包体/灵活性平衡后选 shadcn/ui 的 Tailwind-first 模式）
- **增量重写**: 前端从零用 React + TS 重写，不存在渐进迁移（否决理由：v2 前端是原生 JS，无法增量混用）

## 后果

- Go 后端 `internal/` 包无需修改，降低移植风险
- 前端需要完全重写，工作量较大但可复用 v2 的业务逻辑理解
- Wails 绑定服务的 TS 类型定义自动生成，减少前端类型错误
- `main.go` 仍保持显式 DI，未来可逐步接入依赖注入容器
