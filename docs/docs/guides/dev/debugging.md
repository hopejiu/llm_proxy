---
title: Debugging
description: Using runtime/trace for performance graphs and debugging in Wails applications.
---

# Debugging

## Performance Tracing with `runtime/trace`

Go's `runtime/trace` package generates execution trace files that visualize your
application's performance. These traces show goroutine scheduling, syscall
activity, garbage collection, and more.

### Step 1: Prepare Your Application for Tracing

Import the `runtime/trace` package and set up tracing early in your application:

```go
package main

import (
	"log"
	"os"
	"runtime/trace"
)

func main() {
	// Create the trace output file
	f, err := os.Create("trace.out")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// Start tracing
	trace.Start(f)
	defer trace.Stop()

	// ... your application code here
}
```

### Step 2: Run Your Application

Run the application normally. Interact with the UI to exercise the code paths
you want to profile:

```bash
wails3 dev
```

Once you are done, shut down the application cleanly. The `trace.out` file will
be written to the project root.

### Step 3: Install Graphviz

The trace visualization tool requires Graphviz to render the output graphs:

```bash
# macOS
brew install graphviz

# Ubuntu/Debian
sudo apt-get install graphviz

# Windows (using winget)
winget install graphviz
```

### Step 4: Visualize the Trace

Open the trace file with the Go trace tool:

```bash
go tool trace trace.out
```

This starts a local web server and opens a browser with interactive
visualizations of your application's execution.

## Reading the Trace

The trace viewer provides several views:

- **Goroutines** — Shows goroutine creation, blocking, and scheduling
- **Network** — Displays network I/O activity
- **Syscalls** — Details system call frequency and duration
- **GC** — Garbage collection pauses and heap changes

The **syscall profile** screen is the most useful for first-time users. It
aggregates syscall activity and helps you quickly identify performance
bottlenecks without needing to interpret raw goroutine timelines.
