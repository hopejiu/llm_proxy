# 构建环境、CGO 依赖、CI/CD、开发模式

## CGO 依赖

### 问题

Wails v2 在 Windows 上使用 WebView2，其 Go 绑定（`github.com/AdrianSR/go-webview2` 或 Wails 内置的 webview 包装）**需要 CGO**。这与当前项目 `CGO_ENABLED=0` 的构建方式冲突。

### 解决方案

- Windows 构建环境需安装 **MinGW-w64**（推荐 `x86_64-8.1.0-posix-seh-rt_v6-rev0`）
- 构建命令不再使用 `CGO_ENABLED=0`
- `glebarez/sqlite` 是纯 Go 实现，不依赖 CGO，与 Wails 兼容
- Linux 构建需要 `libgtk-3-dev` 和 `libwebkit2gtk-4.0-dev`

### 构建环境要求

| 平台 | 依赖 |
|------|------|
| Windows | MinGW-w64, WebView2（Windows 10/11 自带） |
| Linux | gcc, libgtk-3-dev, libwebkit2gtk-4.0-dev |

## 构建命令变化

### 开发模式

```bash
# 安装 Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 开发模式（前端热更新 + Go 自动重编译）
wails dev

# 指定前端开发服务器端口（可选）
wails dev -devport 3000
```

`wails dev` 特性：
- Go 代码修改后自动重编译
- 前端文件修改后自动刷新 WebView
- 支持浏览器 DevTools 调试（右键 → 检查元素）

### 生产构建

```bash
# Windows 构建
wails build

# 带 UPX 压缩
wails build -upx

# 指定输出目录
wails build -o llm-proxy.exe

# 跨平台构建（需对应交叉编译环境）
GOOS=linux wails build -o llm-proxy-linux
```

### 构建脚本更新

```bash
# build.sh 改造
#!/bin/bash

# 安装 Wails CLI（如果未安装）
if ! command -v wails &> /dev/null; then
    go install github.com/wailsapp/wails/v2/cmd/wails@latest
fi

# 构建
wails build

echo "构建完成: build/bin/llm-proxy"
```

```powershell
# build.ps1 改造
# 检查 Wails CLI
if (-not (Get-Command wails -ErrorAction SilentlyContinue)) {
    go install github.com/wailsapp/wails/v2/cmd/wails@latest
}

# 构建
wails build

Write-Host "构建完成: build\bin\llm-proxy.exe"
```

## wails.json 配置

```json
{
  "$schema": "https://wails.io/schemas/config.v2.json",
  "name": "llm-proxy",
  "outputfilename": "llm-proxy",
  "frontend:install": "",
  "frontend:build": "",
  "frontend:dev:watcher": "",
  "frontend:dev:serverUrl": "auto",
  "author": {
    "name": "teamsun"
  }
}
```

注意：`frontend:install` 和 `frontend:build` 为空，因为使用 Vanilla JS 无需构建步骤。

## GitHub Actions 适配

### release.yml 改造

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    strategy:
      matrix:
        include:
          - os: windows-latest
            goos: windows
            ext: .exe
            artifact: llm-proxy-windows-amd64
          - os: ubuntu-latest
            goos: linux
            ext: ""
            artifact: llm-proxy-linux-amd64

    runs-on: ${{ matrix.os }}

    steps:
      - uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      # Linux 需要安装 GTK 和 WebKit 依赖
      - name: Install Linux dependencies
        if: matrix.goos == 'linux'
        run: |
          sudo apt-get update
          sudo apt-get install -y libgtk-3-dev libwebkit2gtk-4.0-dev

      - name: Install Wails CLI
        run: go install github.com/wailsapp/wails/v2/cmd/wails@latest

      - name: Build
        run: wails build

      - name: Package
        shell: bash
        run: |
          mkdir -p dist
          cp build/bin/llm-proxy${{ matrix.ext }} dist/
          cp -r frontend dist/frontend
          cp .env dist/.env 2>/dev/null || true

      - name: Upload artifact
        uses: actions/upload-artifact@v4
        with:
          name: ${{ matrix.artifact }}
          path: dist/
```

## 开发模式工作流

### 目录约定

Wails 要求前端文件在 `frontend/` 目录下，Go 入口在项目根目录。开发时：

```bash
# 启动开发模式
wails dev

# Wails 会自动：
# - 编译 Go 代码
# - 启动 WebView 窗口
# - 监听前端文件变更并热更新
# - 监听 Go 文件变更并自动重编译
```

### 调试

- WebView2 支持右键 → "检查" 打开 DevTools
- Go 端日志输出到终端（slog 同时写 stdout + 文件）
- 前端 `console.log` 在 DevTools Console 中可见

### 注意事项

- `wails dev` 启动后，前端文件通过 `wails://` 协议加载
- `fetch('pages/xxx.html')` 在 `wails://` 协议下可用（同源）
- Wails 自动生成的 `frontend/wailsjs/` 目录不应手动编辑，也不应提交到 Git（加入 .gitignore）
- 每次修改 App 绑定方法签名后，需重启 `wails dev` 以重新生成 JS 绑定

## .gitignore 更新

```
# Wails 自动生成
frontend/wailsjs/

# 构建输出
build/bin/

# 窗口状态
.window-state.json
```
