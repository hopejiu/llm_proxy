---
title: Self-Updating Wails App
description: Build a Wails v3 application that updates itself from GitHub Releases — from `wails3 init` through signed release verification and the helper-mode swap.
image: ../../assets/images/og-image.png
---

[Skip to content](#%5Ftop) 

# Self-Updating Wails App

In this tutorial you'll add an in-app updater to a fresh Wails v3 application. By the end, the app will:

* Check GitHub Releases on demand (and optionally on a timer).
* Download the right asset for the running OS + architecture.
* Verify a SHA-256 digest (and optionally an Ed25519 signature) against the bytes it downloaded.
* Show release notes in the framework's default update window.
* Swap the running binary and relaunch — all without shipping a separate helper executable.

We'll use **GitHub Releases** as the update source because it's free and requires no infrastructure. The same patterns work with [keygen.sh](https://v3.wails.io/guides/updater#keygensh--updaterproviderskeygen) and [Sparkle AppCast](https://v3.wails.io/guides/updater#sparkle-appcast--updaterprovidersappcast) — see the [Updater guide](https://v3.wails.io/guides/updater) once you finish.

Prerequisites

* Go 1.25 or newer
* `wails3` CLI installed (`go install github.com/wailsapp/wails/v3/cmd/wails3@latest`)
* A GitHub repository you can push releases to
* Familiarity with the [QR Code Service tutorial](https://v3.wails.io/tutorials/01-creating-a-service) is helpful but not required

1. ## Start with a fresh Wails app

Scaffold a new project with the vanilla template:

```
wails3 init --template vanilla --name updater-tutorial
cd updater-tutorial
```

You should now have a directory with `main.go`, `frontend/`, and a `Taskfile.yml`. Confirm it builds and launches:

```
wails3 task dev
```

A blank Wails window should open. Quit and continue.

2. ## Add the updater import

Open `main.go` and add the two updater packages to your imports:

```go
package main

import (
    _ "embed"
    "github.com/wailsapp/wails/v3/pkg/application"
    "github.com/wailsapp/wails/v3/pkg/updater"
    "github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)
```

These pull in the Updater itself and the GitHub Releases provider.

3. ## Configure the Updater

`app.Updater` is already wired into every `*application.App` — you just need to call `Init`:

```go
const currentVersion = "1.0.0"

gh, err := github.New(github.Config{
    Repository:    "yourorg/your-repo",
    ChecksumAsset: "SHA256SUMS",
})
if err != nil {
    log.Fatalf("github.New: %v", err)
}

if err := app.Updater.Init(updater.Config{
    CurrentVersion: currentVersion,
    Providers:      []updater.Provider{gh},
}); err != nil {
    log.Fatalf("Updater.Init: %v", err)
}
```

Place this after `application.New` and before `app.Run()`.

4. ## Add a menu item that triggers the update

```go
menu := app.Menu.New()
app.Menu.SetApplicationMenu(menu)
appMenu := menu.AddSubmenu("App")
appMenu.Add("Check for Updates...").OnClick(func(*application.Context) {
    go func() {
        if err := app.Updater.CheckAndInstall(context.Background()); err != nil {
            app.Logger.Error("update", "error", err)
        }
    }()
})
```

5. ## Run it once with no releases

```
wails3 task dev
```

Click **App → Check for Updates...**. You should see the update window open briefly, hit the GitHub API, find no releases newer than `1.0.0`, and settle on the **Up to Date** state with a green checkmark.

6. ## Publish a test release

Build for one platform to get a binary you can attach to a release:

```
wails3 task build:darwin
cd bin && zip -r updater-tutorial-darwin-arm64.zip updater-tutorial.app && cd ..
```

Generate a `SHA256SUMS` file:

```
cd bin
shasum -a 256 updater-tutorial-* > SHA256SUMS
```

Now publish this as **v2.0.0** on your GitHub repository:

```
gh release create v2.0.0 \
    --title "v2.0.0" \
    --notes "First update for the self-update tutorial." \
    bin/SHA256SUMS bin/updater-tutorial-*
```

7. ## Run the app and verify the update

With `currentVersion` still `1.0.0`, run the app again:

```
wails3 task dev
```

Click **App → Check for Updates...**. The updater stages the new binary in a temp directory. Click **Restart & Apply** to complete the update.

8. ## Wire currentVersion to the build

Replace the constant with a build-time variable:

```go
var (
    currentVersion = "dev" // overridden by -ldflags at release time
)
```

Then in your build command:

```
wails3 task build:darwin -- -ldflags "-X main.currentVersion=2.0.0"
```

9. ## Add cryptographic signing (recommended for production)

Generate the keypair:

```
ssh-keygen -t ed25519 -f updater-key -N "" -C "wails-updater"
```

Embed the public key in your app:

```go
//go:embed updater-key.pub
var updaterPublicKey []byte

app.Updater.Init(updater.Config{
    CurrentVersion: currentVersion,
    PublicKey:      updaterPublicKey,
    Providers:      []updater.Provider{gh},
})
```

10. ## Run automatic checks in the background

```go
app.Updater.Init(updater.Config{
    CurrentVersion: currentVersion,
    Providers:      []updater.Provider{gh},
    PublicKey:      updaterPublicKey,
    CheckInterval:  6 * time.Hour,
})
```

## You're done

You now have a Wails app that:

* Checks GitHub Releases for updates on demand and on a timer.
* Renders release notes as Markdown in a polished default window.
* Verifies downloads against a SHA-256 digest you publish.
* Optionally verifies an Ed25519 signature against a public key you embed at build time.
* Swaps the running binary in-place and relaunches automatically.

## Next steps

* The [Updater guide](https://v3.wails.io/guides/updater) has the full API reference, every event, every config option, and the helper-mode swap mechanics.
* Look at [v3/examples/updater](https://github.com/wailsapp/wails/tree/master/v3/examples/updater) for a complete working example you can clone.
* The test target repo [wailsapp/updater-demo](https://github.com/wailsapp/updater-demo) shows the recommended release-asset layout.
