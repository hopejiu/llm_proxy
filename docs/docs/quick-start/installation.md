---
title: Installation
description: Get Wails installed and ready to build applications
---

# Installation

## Quick Install (5 Minutes)

TL;DR - Experienced Developers

```bash
# Install Go 1.25+, then:
go install github.com/wailsapp/wails/v3/cmd/wails3@latest
wails3 doctor  # Verify installation
```

If `wails3 doctor` passes, you're done. [Skip to First App →](https://v3.wails.io/quick-start/first-app)

## Step-by-Step Installation

### 1. Install Go (Required)

Wails requires Go 1.25 or later.

#### Windows

Download the Windows installer from [go.dev/dl](https://go.dev/dl/) and run it.

**Verify installation:**

```bash
go version  # Should show 1.25 or later
```

**Check PATH:**

```powershell
$env:PATH -split ';' | Where-Object { $_ -like '*\go\bin' }
```

If empty, add `C:\Users\YourName\go\bin` to your PATH.

#### macOS

**Option 1: Official Installer**

Download the macOS installer (.pkg file) from [go.dev/dl](https://go.dev/dl/) and run it.

**Option 2: Homebrew**

```bash
brew install go
```

**Verify installation:**

```bash
go version  # Should show 1.25 or later
echo $PATH | grep go/bin  # Should show ~/go/bin
```

If `~/go/bin` isn't in PATH, add to `~/.zshrc` or `~/.bash_profile`:

```bash
export PATH=$PATH:~/go/bin
```

#### Linux

**Option 1: Official Tarball**

Download the Linux tarball from [go.dev/dl](https://go.dev/dl/), then:

```bash
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.25.linux-amd64.tar.gz
```

**Option 2: Package Manager**

```bash
# Ubuntu/Debian
sudo apt install golang-go
# Fedora
sudo dnf install golang
# Arch
sudo pacman -S go
```

**Add to PATH** (add to `~/.bashrc` or `~/.zshrc`):

```bash
export PATH=$PATH:/usr/local/go/bin:~/go/bin
source ~/.bashrc  # Reload
```

**Verify:**

```bash
go version
echo $PATH | grep go/bin
```

### 2. Install Platform Dependencies

#### Windows

**WebView2 Runtime** (usually pre-installed)

Windows 10/11 includes WebView2 by default. If missing:
- Download from [Microsoft](https://developer.microsoft.com/microsoft-edge/webview2/)
- Or run `wails3 doctor` later—it will guide you

**That's it!** No other dependencies needed.

> **Performance Tip for Windows 11:** Consider using [Dev Drive](https://learn.microsoft.com/en-us/windows/dev-drive/) to store your projects. Dev Drives are optimized for developer workloads and can significantly improve build times and disk access speeds by up to 30%.

#### macOS

**Xcode Command Line Tools** (required)

```bash
xcode-select --install
```

Click "Install" in the dialog that appears.

**Verify:**

```bash
xcode-select -p  # Should show /Library/Developer/CommandLineTools
```

**That's it!** macOS includes WebKit by default.

#### Linux

**Build tools and WebKit**

> **Minimum distro versions:** Wails v3 requires **WebKitGTK 6.0** by default. Distributions that ship only WebKit2GTK 4.1 — Ubuntu 22.04 LTS, Debian 12, Fedora ≤ 39, RHEL 9.x — must build with the legacy `-tags gtk3` opt-in. Older releases that ship only WebKit2GTK 4.0 (Ubuntu 20.04, Debian 11, RHEL 8) are not supported.

**Ubuntu/Debian** (Requires Ubuntu 24.04+ or Debian 13+ for the default GTK4 stack):

```bash
sudo apt update
sudo apt install build-essential pkg-config libgtk-4-dev libwebkitgtk-6.0-dev
```

**Fedora:**

```bash
sudo dnf install gcc pkg-config gtk4-devel webkitgtk6.0-devel
```

**Arch:**

```bash
sudo pacman -S base-devel gtk4 webkitgtk-6.0
```

**openSUSE:**

```bash
sudo zypper install gcc pkg-config gtk4-devel webkitgtk-6_0-devel
```

**Gentoo:**

```bash
sudo emerge --ask net-libs/webkit-gtk:6
```

**NixOS:**

Add to your `shell.nix` or `devShell`:

```nix
buildInputs = with pkgs; [ webkitgtk_6_0 gtk4 pkg-config gcc ];
```

Run `wails3 doctor` after installing Wails—it will show the exact packages needed for your distribution.

**Legacy GTK3 stack:**

If your target distribution does not yet ship WebKitGTK 6.0 (e.g. Ubuntu 22.04 LTS, Debian 12), install GTK3 + WebKit2GTK 4.1 development libraries instead (`libgtk-3-dev libwebkit2gtk-4.1-dev` on Debian/Ubuntu; equivalents on other distros) and build with `wails3 build -tags gtk3`. The legacy path is supported through the v3.0.x line and will be removed in v3.1. See [Linux Packaging - Legacy GTK3 Support](https://v3.wails.io/guides/build/linux#legacy-gtk3-support) for details.

### 3. Install Wails CLI

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@latest
```

This installs the `wails3` command to `~/go/bin` (or `%USERPROFILE%\go\bin` on Windows).

### 4. Verify Installation

```bash
wails3 doctor
```

**Expected output (or similar):**

```
Wails (v3.0.0-dev)  Wails Doctor
# System
┌──────────────────────────────────────────────────┐
| Name          | MacOS                            |
| Version       | 26.0                             |
| ID            | 25A354                           |
| Branding      | MacOS 26.0                       |
| Platform      | darwin                           |
| Architecture  | arm64                            |
| Apple Silicon | true                             |
| CPU           | Apple M2 Pro                     |
| Memory        | 16 GB                            |
└──────────────────────────────────────────────────┘
# Build Environment
┌─────────────┬─────────────────┐
| Wails CLI   | v3.0.0-alpha.40 |
| Go Version  | go1.24.6        │
└─────────────┴─────────────────┘
# Dependencies
┌─────────────────┬─────────────────────────────────────────────────┐
| npm             │ 11.6.2                                          │
│ *NSIS           │ Not Installed. Install with `brew install...`.  │
│ Xcode cli tools │ 2412                                            │
└─────────────────┴─────────────────────────────────────────────────┘
# Checking for issues
SUCCESS No issues found
# Diagnosis
SUCCESS Your system is ready for Wails development!
```

> If `wails3` command not found: Your `~/go/bin` isn't in PATH. See step 1 above to fix this, then restart your terminal.

### 5. Install npm (Optional but Recommended)

Most Wails templates use npm for frontend tooling.

#### Windows

Download from [nodejs.org](https://nodejs.org/) and run the installer.

**Verify:**

```bash
npm --version
```

#### macOS

**Option 1: Official Installer**

Download from [nodejs.org](https://nodejs.org/)

**Option 2: Homebrew**

```bash
brew install node
```

**Verify:**

```bash
npm --version
```

#### Linux

**Option 1: NodeSource**

```bash
curl -fsSL https://deb.nodesource.com/setup_lts.x | sudo -E bash -
sudo apt-get install -y nodejs  # Ubuntu/Debian
```

**Option 2: Package Manager**

```bash
sudo dnf install nodejs  # Fedora
sudo pacman -S nodejs npm  # Arch
```

**Verify:**

```bash
npm --version
```

> **Alternative Package Managers:** Prefer `pnpm`, `yarn`, or `bun`? No problem! Just update the `Taskfile.yml` in your project to use your preferred tool.

## Troubleshooting

### `wails3` command not found

**Cause:** `~/go/bin` (or `%USERPROFILE%\go\bin`) isn't in your PATH.

**Solution:**

**Windows:**

1. Open "Environment Variables" (search in Start menu)
2. Under "User variables", find `Path`
3. Click "Edit" → "New"
4. Add: `C:\Users\YourName\go\bin` (replace `YourName`)
5. Click "OK" on all dialogs
6. **Restart your terminal**

**Verify:**

```powershell
$env:PATH -split ';' | Where-Object { $_ -like '*\go\bin' }
```

**macOS/Linux:**

Add to `~/.zshrc` (macOS) or `~/.bashrc` (Linux):

```bash
export PATH=$PATH:~/go/bin
```

Reload:

```bash
source ~/.zshrc  # or ~/.bashrc
```

**Verify:**

```bash
echo $PATH | grep go/bin
wails3 version
```

### `wails3 doctor` reports missing dependencies

**Linux:** The output tells you exactly which packages to install. Example:

```
❌ webkit2gtk not found
   Install with: sudo apt install libwebkit2gtk-4.1-dev
```

**Windows:** If WebView2 is missing:
- Download from [Microsoft](https://developer.microsoft.com/microsoft-edge/webview2/)
- Or it will be installed automatically when you run your first app

**macOS:** If Xcode tools are missing:

```bash
xcode-select --install
```

### Go version too old

Wails v3 requires Go 1.25+. If you have an older version:

**Windows/macOS:** Download the latest from [go.dev/dl](https://go.dev/dl/) and reinstall.

**Linux:** Download the latest tarball from [go.dev/dl](https://go.dev/dl/), then:

```bash
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.25.linux-amd64.tar.gz
```

## Development Version (Bleeding Edge)

Want to use the absolute latest code from the main development branch? This gives you access to new features and fixes before they're released, but comes with the risk of bugs and breaking changes. Only recommended for contributors or those who need to test upcoming features.

```bash
git clone https://github.com/wailsapp/wails.git
cd wails
git checkout v3
cd v3/cmd/wails3
go install
```

> **Development Version:**
> - May have bugs or breaking changes
> - Projects created will use `replace` directive to point to local Wails
> - Only recommended for contributors or testing new features

## Next Steps

**Installation Complete!** Your system is ready for Wails development.

**Build Your First App** - Create a working application in 10 minutes.

[First App Tutorial →](https://v3.wails.io/quick-start/first-app)

**Explore Templates** - See what's available out of the box.

```bash
wails3 init -l  # List templates
```

---

**Having issues?** Ask in [Discord](https://discord.gg/JDdSxwjhGf) or [open an issue](https://github.com/wailsapp/wails/issues).
