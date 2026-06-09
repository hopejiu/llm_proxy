---
title: "Getting Started"
description: "How to start contributing to Wails v3"
---

## Prerequisites

Before you begin, ensure you have:

* **Go 1.25+** installed
* **Node.js 20+** and **npm**
* **Git** configured with your GitHub account

### Platform-Specific Requirements

**macOS:** Xcode Command Line Tools: `xcode-select --install`

**Windows:** MSYS2 or similar Unix-like environment recommended

**Linux:** `sudo apt install build-essential pkg-config libgtk-4-dev libwebkitgtk-6.0-dev`

## Contribution Process Overview

1. **Fork & Clone** - Create your own copy of the Wails repository
2. **Setup** - Build the Wails CLI and verify your environment
3. **Branch** - Create a feature branch for your changes
4. **Develop** - Make your changes following our coding standards
5. **Test** - Run tests to ensure everything works
6. **Commit** - Commit with clear, conventional commit messages
7. **Submit** - Open a pull request for review
8. **Iterate** - Respond to feedback and make adjustments
9. **Merge** - Once approved, your changes become part of Wails!

## Bug Fix Workflow

1. Find or report the bug in GitHub Issues
2. Fork and clone the repository
3. Build and verify you can reproduce the bug
4. Create a branch: `git checkout -b fix/issue-123-window-crash`
5. Fix the bug and add tests
6. Run tests: `go test ./...`
7. Commit and submit a pull request

## Enhancement Workflow

1. Discuss the feature in GitHub Discussions
2. Wait for maintainer feedback
3. Fork, clone, and create a feature branch
4. Implement following coding standards
5. Add comprehensive tests
6. Commit with conventional commit messages
7. Submit a pull request

## Documentation Workflow

1. Identify documentation needs (typos, missing examples, etc.)
2. The docs are in `/docs/src/content/docs/` and built with Astro
3. Run `cd docs && npm install && npm run dev` to preview
4. Commit and submit a pull request

## Finding Issues to Work On

* Look for [good first issue](https://github.com/wailsapp/wails/labels/good%20first%20issue) labels
* Check [help wanted](https://github.com/wailsapp/wails/labels/help%20wanted) issues
* Browse [open issues](https://github.com/wailsapp/wails/issues)
