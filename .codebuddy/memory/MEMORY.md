# LLM Proxy 项目记忆

## 项目概述
- **项目**: llm-proxy (Wails v3 桌面应用)
- **技术栈**: Go 1.25 + React 18 + TypeScript + Tailwind CSS + ECharts
- **构建**: wails3 build (Taskfile.yml 驱动)
- **主要功能**: LLM 代理、统计、日志、实时监控

## 构建体积分析 (2026-06-09)
- **二进制大小**: 30.15 MB (bin/llm-proxy.exe)
- **主要来源**: Go 运行时 + Wails 框架 + Gin + GORM + SQLite + 前端资源
- **优化建议**: UPX 压缩、按需引入 ECharts、检查 quic-go 依赖

## 架构决策
- **Wails v2 → v3 迁移**: internal/ 包零修改搬迁，前端从原生 JS 重写为 React
- **服务拆分**: 拆成多个 Wails 语义服务（非单一大 App）
- **日志页面**: 从请求日志改为运行日志查看器
- **统计页面**: 增强统计仪表盘，新增趋势指标、图表维度切换

## 前端组件系统 (2026-06-09)
- **Modal**: 通用弹窗组件 (components/Modal.tsx)
- **LoadingSpinner**: 通用加载图标 (components/LoadingSpinner.tsx)
- **ErrorBanner**: 统一错误提示 (components/ErrorBanner.tsx)
- **RecentRequestsTable**: 请求表格+日志详情弹窗
- **useLocalStorage**: localStorage 持久化 hook
- **useErrorHandler**: 统一错误处理 hook
- **Wails API 服务层**: services/index.ts

## 后端重构 (2026-06-09)
- **LogService**: 从 StatsService 提取独立日志服务
- **WebHandler**: 12 个端点待拆分（当前未在 v3 使用）
- **VO.go**: 按领域分组 Provider/Stats+Log/Common

## 用户偏好
- **语言**: 简体中文
- **构建工具**: uv 代替 python
- **设计风格**: 紫色+浅色商业化主题 (#7C3AED)