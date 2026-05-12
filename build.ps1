# LLM Proxy Build Script
# Build with Wails, copy output to dist

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $ScriptDir

$DistPath = Join-Path $ScriptDir "dist"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  LLM Proxy Build Script" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Clean output directory
if (Test-Path $DistPath) {
    # Try to stop running instance first
    $exePath = Join-Path $DistPath "llm-proxy.exe"
    if (Test-Path $exePath) {
        $proc = Get-Process -Name "llm-proxy" -ErrorAction SilentlyContinue
        if ($proc) {
            Write-Host "[STOP] Stopping running llm-proxy..." -ForegroundColor Yellow
            Stop-Process -Name "llm-proxy" -Force -ErrorAction SilentlyContinue
            Start-Sleep -Milliseconds 500
        }
    }
    Write-Host "[CLEAN] Removing old dist..." -ForegroundColor Yellow
    Remove-Item -Path $DistPath -Recurse -Force -ErrorAction SilentlyContinue
    if (Test-Path $DistPath) {
        Write-Host "[WARN] Could not fully clean dist (files may be locked)" -ForegroundColor Yellow
    }
}

# Clear WebView2 cache to ensure new frontend assets are loaded
$WebViewDataDir = Join-Path $env:APPDATA "llm-proxy\EBWebView\Default"
if (Test-Path $WebViewDataDir) {
    $cachePaths = @("Cache", "Code Cache")
    foreach ($cp in $cachePaths) {
        $fullPath = Join-Path $WebViewDataDir $cp
        if (Test-Path $fullPath) {
            Remove-Item -Path $fullPath -Recurse -Force -ErrorAction SilentlyContinue
            Write-Host "[CLEAN] WebView2 $cp cleared" -ForegroundColor Green
        }
    }
}
New-Item -ItemType Directory -Path $DistPath -Force | Out-Null
Write-Host "[DONE] Output directory: $DistPath" -ForegroundColor Green

# Clean frontend dist and Vite cache to ensure fresh build
$FrontendDist = Join-Path $ScriptDir "frontend\dist"
$ViteCache = Join-Path $ScriptDir "frontend\node_modules\.vite"
if (Test-Path $FrontendDist) {
    Write-Host "[CLEAN] Removing old frontend/dist..." -ForegroundColor Yellow
    Remove-Item -Path $FrontendDist -Recurse -Force -ErrorAction SilentlyContinue
}
if (Test-Path $ViteCache) {
    Write-Host "[CLEAN] Removing Vite cache..." -ForegroundColor Yellow
    Remove-Item -Path $ViteCache -Recurse -Force -ErrorAction SilentlyContinue
}

# Sync page HTML fragments from src/pages/ to public/pages/
# Vite copies public/ to dist/ during build, so this ensures go:embed picks up the latest HTML.
$PagesSrc = Join-Path $ScriptDir "frontend\src\pages"
$PagesPublic = Join-Path $ScriptDir "frontend\public\pages"
if (Test-Path $PagesSrc) {
    if (-not (Test-Path $PagesPublic)) { New-Item -ItemType Directory -Path $PagesPublic -Force | Out-Null }
    Copy-Item -Path "$PagesSrc\*.html" -Destination $PagesPublic -Force
    Write-Host "[SYNC] Page HTML fragments synced to frontend/public/pages/" -ForegroundColor Green
}

# Build with Wails
Write-Host ""
Write-Host "[BUILD] Compiling with Wails..." -ForegroundColor Yellow

# Generate version info
$BuildTime = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
$GitHash = "unknown"
try {
    $GitHash = (git rev-parse --short HEAD 2>$null).Trim()
} catch {}

$VersionStr = "1.0.0"
# BuildTime 用 ISO 格式避免空格问题
$BuildTimeCompact = Get-Date -Format "yyyy-MM-dd_HH:mm:ss"
$Ldflags = "-s -w -X main.Version=$VersionStr -X main.BuildTime=$BuildTimeCompact"
Write-Host "[INFO] Version: $VersionStr, Build: $BuildTimeCompact, Git: $GitHash" -ForegroundColor Cyan

wails build -ldflags $Ldflags

if ($LASTEXITCODE -ne 0) {
    Write-Host "[ERROR] Build failed!" -ForegroundColor Red
    exit 1
}
Write-Host "[DONE] Build successful" -ForegroundColor Green

# Copy .env if exists
if (Test-Path ".env") {
    Copy-Item ".env" $DistPath -Force
    Write-Host "[COPY] .env copied" -ForegroundColor Green
}

# Copy build output to dist
$BuiltExe = Join-Path $ScriptDir "build\bin\llm-proxy.exe"
if (Test-Path $BuiltExe) {
    Copy-Item $BuiltExe $DistPath -Force
    Write-Host "[COPY] llm-proxy.exe copied to dist" -ForegroundColor Green
} else {
    Write-Host "[WARN] build/bin/llm-proxy.exe not found" -ForegroundColor Red
}

# Show output
Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Build Complete!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Output directory: $DistPath" -ForegroundColor White
Write-Host ""
Get-ChildItem -Path $DistPath -Recurse | ForEach-Object {
    $RelativePath = $_.FullName.Substring($DistPath.Length + 1)
    if ($_.PSIsContainer) {
        Write-Host "  [DIR]  $RelativePath" -ForegroundColor DarkGray
    } else {
        $SizeKB = [math]::Round($_.Length / 1KB, 2)
        Write-Host "  [FILE] $RelativePath ($SizeKB KB)" -ForegroundColor Gray
    }
}
