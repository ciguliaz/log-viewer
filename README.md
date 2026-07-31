# Log Viewer

A high-performance, real-time log viewer built with Wails (Go) and React (TypeScript). Designed to handle massive log files seamlessly without freezing the UI or triggering Windows file locking issues.

## Features at a Glance
- **Real-Time Streaming:** Instant, real-time log tailing using a custom lock-free polling mechanism.
- **Massive Scale:** Virtualized rendering (React Virtuoso) supports viewing 500,000+ lines without UI lag.
- **Auto-Parsing:** Automatically extracts and organizes Timestamps, Log Levels (INFO/ERROR/WARN), and Tags into resizable, toggleable columns.
- **Smart Workspaces:** Persists your folders, detects file renames/deletions automatically, and natively handles drag-and-drop.
- **Safe on Windows:** Will never block you from renaming or deleting log folders in File Explorer while the app is running.

## Live Development

To run in live development mode, ensure you have [Wails](https://wails.io/) installed, then run:

```bash
wails dev
```

This will boot a Vite development server that provides extremely fast hot-reloading of frontend changes. It also starts a local dev server at `http://localhost:34115` if you prefer testing the UI in your web browser.

## Building for Production

To build a standalone, redistributable `.exe` for Windows, use:

```bash
wails build
```

The compiled binary will be placed in the `build/bin/` directory.

---
*For in-depth technical details on how the lock-free polling architecture works and a list of known edge cases, please see [ARCHITECTURE.md](ARCHITECTURE.md).*
