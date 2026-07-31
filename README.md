# Log Viewer

A fast, real-time log file viewer for Windows. Built with [Wails](https://wails.io/) (Go) and React (TypeScript).

Handles massive log files without freezing the UI or locking files in Windows Explorer.

## Download

Go to the [Releases](https://github.com/ciguliaz/log-viewer/releases/latest) page, download the `.zip` file, extract `log-viewer.exe`, and run it. No installation required.

The app checks for updates automatically — when a new version is available, a notification appears in the title bar.

## Features

- **Real-time tailing** — Streams log updates live as they are written to disk
- **Large file support** — Virtualized rendering handles 500,000+ lines without lag
- **Auto-parsing** — Extracts timestamps, log levels (INFO/ERROR/WARN), and tags into columns
- **Compact mode** — Groups repeated log entries with occurrence counts and time ranges
- **Drag & drop** — Drop folders directly into the app to start viewing
- **Workspace persistence** — Remembers your folders between sessions
- **File-safe** — Uses a lock-free polling mechanism that never blocks file operations in Explorer
- **Auto-update** — Built-in update checker with one-click update from within the app

## Development

### Prerequisites

- [Go](https://go.dev/dl/) 1.23+
- [Node.js](https://nodejs.org/) 20+
- [Wails CLI](https://wails.io/docs/gettingstarted/installation)

### Run in dev mode

```bash
wails dev
```

### Build for production

```bash
wails build
```

The compiled binary will be in `build/bin/`.

## Architecture

For technical details on the lock-free polling mechanism and parsing pipeline, see [ARCHITECTURE.md](ARCHITECTURE.md).

## License

MIT
