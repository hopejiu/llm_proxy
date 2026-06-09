---
title: "Updater"
description: "In-app self-updates for Wails v3"
---

The updater ships in-app software updates without forcing you to build your own download / verify / swap pipeline. It sits on `app.Updater`, accepts one or more pluggable `Provider`s (GitHub Releases, keygen.sh, Sparkle AppCast, or your own), authenticates downloads against a configured public key, swaps the running binary safely, and surfaces every transition through the standard Wails event bus.

## Quick start

```go
package main

import (
    "context"
    "log"

    "github.com/wailsapp/wails/v3/pkg/application"
    "github.com/wailsapp/wails/v3/pkg/updater"
    "github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

func main() {
    app := application.New(application.Options{Name: "Demo"})

    gh, _ := github.New(github.Config{Repository: "myorg/myapp"})
    if err := app.Updater.Init(updater.Config{
        CurrentVersion: "1.0.0",
        Providers:      []updater.Provider{gh},
    }); err != nil {
        log.Fatal(err)
    }

    if err := app.Updater.CheckAndInstall(context.Background()); err != nil {
        log.Printf("update: %v", err)
    }

    _ = app.Run()
}
```

That opens the framework's update window, checks GitHub, downloads the platform asset, verifies it, swaps the binary, and waits for the user to restart.

## The lifecycle

`app.Updater` is a state machine with these states (`updater.State`):

| State        | When                                                           |
| ------------ | -------------------------------------------------------------- |
| unconfigured | Before Init is called                                          |
| idle         | After Init, before any check                                   |
| checking     | A Check is in flight                                           |
| up-to-date   | The latest provider response said the caller is current        |
| available    | A new release was found; not yet downloading                   |
| downloading  | Bytes are streaming from the provider                          |
| verifying    | Download complete, signature/digest being checked              |
| installing   | Verified bytes being unpacked and renamed into the staging dir |
| ready        | Update staged; call Restart to apply                           |
| error        | Any prior step failed                                          |

You can read the current state with `app.Updater.State()` at any time. Every transition also emits a Wails event.

## Providers

A `Provider` is anything that satisfies this interface:

```go
type Provider interface {
    Name() string
    Check(ctx context.Context, req CheckRequest) (*Release, error)
    Download(ctx context.Context, r *Release, dst io.Writer, onProgress func(written, total int64)) error
}
```

Three implementations ship in-tree.

### GitHub Releases

```go
gh, err := github.New(github.Config{
    Repository:    "myorg/myapp",
    Token:         "ghp_...",
    Prerelease:    false,
    ChecksumAsset: "SHA256SUMS",
    BaseURL:       "",
    AssetMatcher:  nil,
    HTTPClient:    nil,
})
```

### keygen.sh

```go
kg, err := keygen.New(keygen.Config{
    Account:    "your-account-slug",
    Product:    "product-uuid",
    Package:    "",
    Channel:    "stable",
    Filetype:   "",
    Token:      "prod-...",
    LicenseKey: "",
    BaseURL:    "",
    HTTPClient: nil,
})
```

### Sparkle AppCast

```go
ac, err := appcast.New(appcast.Config{
    URL:        "https://your.app/appcast.xml",
    Channel:    "stable",
    HTTPClient: nil,
})
```

### Fallback chain

`Config.Providers` is ordered. The updater walks it sequentially.

## Cryptographic verification

Releases are authenticated by the framework's verifier using `Config.PublicKey` as the trust root:

```go
//go:embed publickey.pem
var publicKey []byte

app.Updater.Init(updater.Config{
    CurrentVersion: "1.0.0",
    Providers:      []updater.Provider{...},
    PublicKey:      publicKey,
})
```

## How the swap works

`Restart` re-execs the current binary with sentinel environment variables set. `application.New` detects them at startup and diverts to helper-mode.

## Distribution checklist

Before you publish a release that the updater will install:

1. **Pick the right archive format.** macOS: `.zip` of the `.app` bundle. Linux: single binary or `.tar.gz`. Windows: single `.exe` or `.zip`. `.dmg` / `.msi` / `.pkg` are not supported.
2. **Sign the artifact** with the private key matching `Config.PublicKey`.
3. **Match the version string.** `Config.CurrentVersion` and the release's version tag must match exactly.
4. **Test the swap on the target platform** at least once before shipping.
