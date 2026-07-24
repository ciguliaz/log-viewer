# Architecture & Technical Details

This document outlines the internal architecture of the Hatacone Log Viewer, detailing what the application natively handles, the edge cases it solves, and the known gaps for future development.

## 1. Core Architecture & File Handling

Because log files can grow to gigabytes in size, the viewer abandons standard file monitoring (like `fsnotify` or standard `tail`) in favor of a **Stateless Polling** architecture.

### Lock-Free Reading
Instead of maintaining a continuous open file handle, the Go backend polls the file size every 50ms using `os.Stat`. 
- **If the size increases**, it briefly opens the file, reads the new bytes (capped at 5MB per tick to prevent RAM/CPU spikes), and immediately closes the file.
- **Why?** This ensures the file is held open for less than 1 millisecond at a time. It prevents Windows from throwing "Access Denied" errors, allowing the user to rename, delete, or move log folders in File Explorer at any time, even while actively viewing them.

### Log Rotation & Truncation
If a log rotation tool (like `lumberjack`) renames the file and starts a new one, or if a user manually wipes the file contents, the file size drops. 
When the backend detects `info.Size() < currentOffset`, it assumes a wipe occurred. It instantly:
1. Resets its reading cursor to `0`.
2. Sends a `ClearLogs` signal to the React frontend to garbage-collect the old array.
3. Begins streaming the new logs seamlessly from the top.

### Incomplete Lines
If an external program flushes a log string before it has finished writing the newline character `\n`, the viewer detects the line is incomplete. It ignores the incomplete chunk and waits for the rest of the line to be written in the next 50ms tick.

### Mid-file Editing or Deletion
Log files are treated as strictly "append-only". Standard log viewers only monitor the end of the file. If you open a 500MB log file in Notepad and change a typo in the middle of the file, the viewer will not detect it. Detecting retroactive edits would require hashing or diffing the entire file every 50ms, which is computationally impossible for large files.

## 2. Frontend & Data Pipeline

### Parsing 
- **Time/Level/Tags:** The backend parses standard levels (`[INFO]`, `Warning:`), extracts timestamps (ISO8601, Python, KV formats), and isolates tags.
- **Shadow Logs:** Includes a specialized parser for `shadow.log` format matching network routing hashes.

### UI Virtualization & Throttling
- **Virtuoso:** Uses `react-virtuoso` to render only the visible DOM rows. This allows smooth 60fps scrolling even with half a million lines loaded in the JavaScript array.
- **50ms Batching:** The backend batches new log lines and pushes them to the React state every 50ms (20fps). This acts as a throttle to prevent the UI from completely freezing during massive burst logging.

### Smart Workspaces
Workspaces are persisted to `localStorage`. The frontend runs a background worker every 3 seconds to poll all tracked folders:
- If a folder is deleted, it shows a ⚠️ indicator.
- By using deep string comparisons of the file names (`name1|name2`), it detects if files inside a folder are renamed, added, or deleted, and updates the sidebar in real-time.

---

## 3. Known Gaps & Future Work

While the core streaming architecture is highly optimized, the following gaps have been identified for future improvement:

### Missing Core Features
1. **Log Coloring & Highlighting:** Add CSS classes to colorize rows based on the extracted `Level` (e.g., Red for `ERROR`, Yellow for `WARN`, Gray for `DEBUG`).
2. **Direct "Close File" Button:** Currently, to stop tailing a file and free memory, the user must select a different file or remove the parent folder. Add an `✕` button on the active file tab.

### Performance & Edge Cases
3. **Search Performance Lag:** The search bar currently filters the entire `logs` array on every keystroke. With 500,000+ lines in memory, typing causes noticeable UI lag. *Solution: Implement debouncing (wait 300ms) or move search to a Web Worker.*
4. **Gigantic Single Lines:** The backend reads in 5MB chunks. If a *single* log line (like a massive JSON payload) is larger than 5MB and has no newlines, the backend will stall. *Solution: Add a fallback to forcefully split strings if no `\n` is found after 5MB.*
5. **Memory Bloat on Extreme History Loading:** Continuously scrolling up triggers backwards history fetching. While the DOM stays clean, the JS array will eventually crash the browser tab if it exceeds ~2GB of RAM. *Solution: Implement a rolling sliding-window to garbage-collect the bottom lines when loading massive amounts of top lines.*
6. **Non UTF-8 Encodings:** The parser currently assumes all text is standard UTF-8/ASCII. UTF-16LE logs will render poorly.

### Advanced Capabilities
7. **Regex Search:** Search is currently limited to basic case-insensitive substring matching. Add a toggle for Regular Expression searching.
