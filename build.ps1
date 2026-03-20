# LLM Proxy Build Script
# Clean dist, compile main.exe, copy web and .env

param(
    [string]$OutputDir = "dist"
)

$ErrorActionPreference = "Stop"

# Get script directory
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $ScriptDir

# Output directory
$DistPath = Join-Path $ScriptDir $OutputDir

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  LLM Proxy Build Script" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Create output directory
if (-not (Test-Path $DistPath)) {
    New-Item -ItemType Directory -Path $DistPath -Force | Out-Null
    Write-Host "[CREATE] Output directory: $DistPath" -ForegroundColor Green
} else {
    Write-Host "[EXISTS] Output directory: $DistPath" -ForegroundColor Yellow
}

# Clean all files in output directory
Write-Host ""
Write-Host "[CLEAN] Removing all files in output directory..." -ForegroundColor Yellow
Get-ChildItem -Path $DistPath -Force | Remove-Item -Recurse -Force -ErrorAction SilentlyContinue
Write-Host "[DONE] Output directory cleaned" -ForegroundColor Green

# Compile Go program
Write-Host ""
Write-Host "[BUILD] Compiling main.exe..." -ForegroundColor Yellow
$Env:CGO_ENABLED = 0
$Env:GOOS = "windows"
$Env:GOARCH = "amd64"

$MainExe = Join-Path $DistPath "main.exe"
go build -ldflags="-s -w" -o $MainExe ./cmd/server/main.go

if ($LASTEXITCODE -ne 0) {
    Write-Host "[ERROR] Build failed!" -ForegroundColor Red
    exit 1
}
Write-Host "[DONE] main.exe compiled successfully" -ForegroundColor Green

# Copy web directory
Write-Host ""
Write-Host "[COPY] Copying web directory..." -ForegroundColor Yellow
$WebSource = Join-Path $ScriptDir "web"
$WebDest = Join-Path $DistPath "web"
if (Test-Path $WebSource) {
    Copy-Item -Path $WebSource -Destination $WebDest -Recurse -Force
    Write-Host "[DONE] web directory copied" -ForegroundColor Green
} else {
    Write-Host "[WARN] web directory not found" -ForegroundColor Red
}

# Copy .env config file
Write-Host ""
Write-Host "[COPY] Copying .env config file..." -ForegroundColor Yellow
$EnvSource = Join-Path $ScriptDir ".env"
$EnvDest = Join-Path $DistPath ".env"
if (Test-Path $EnvSource) {
    Copy-Item -Path $EnvSource -Destination $EnvDest -Force
    Write-Host "[DONE] .env config file copied" -ForegroundColor Green
} else {
    Write-Host "[WARN] .env file not found, skipped" -ForegroundColor Yellow
}

# Show output directory contents
Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Build Complete!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Output directory: $DistPath" -ForegroundColor White
Write-Host ""
Write-Host "Contents:" -ForegroundColor White
Get-ChildItem -Path $DistPath -Recurse | ForEach-Object {
    $RelativePath = $_.FullName.Substring($DistPath.Length + 1)
    if ($_.PSIsContainer) {
        Write-Host "  [DIR]  $RelativePath" -ForegroundColor DarkGray
    } else {
        $SizeKB = [math]::Round($_.Length / 1KB, 2)
        Write-Host "  [FILE] $RelativePath ($SizeKB KB)" -ForegroundColor Gray
    }
}

Write-Host ""
Write-Host "Run: cd $DistPath; .\main.exe" -ForegroundColor Cyan
