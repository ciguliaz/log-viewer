package main

import (
	"context"
	"os"
	"sync"
)

// LogEntry represents a single parsed log line displayed in the frontend.
type LogEntry struct {
	Id      string `json:"id"`
	Date    string `json:"date"`
	Time    string `json:"time"`
	Ms      string `json:"ms"`
	Tz      string `json:"tz"`
	Level   string `json:"level"`
	Tag     string `json:"tag"`
	Message string `json:"message"`
	Raw     string `json:"raw"`
	EndDate string `json:"endDate"`
	EndTime string `json:"endTime"`
	EndMs   string `json:"endMs"`
	EndTz   string `json:"endTz"`
	Count   int    `json:"count"`
	LineNum int64  `json:"lineNum"`
}

// LogUpdate is the delta payload sent to the frontend via EventsEmit every 50ms.
type LogUpdate struct {
	NewEntries      []LogEntry `json:"newEntries"`
	LastEntryUpdate *LogEntry  `json:"lastEntryUpdate"`
	ClearLogs       bool       `json:"clearLogs"`
}

// App struct holds all application state.
// Methods on App are split across multiple files:
//   - workspace.go: SelectFolder, ListFiles, ProcessDrop
//   - tailer.go:    StartTailing, tailLogs, broadcastLoop, LoadPreviousChunk, GetInitialLogs
//   - parser.go:    parseLine, parseSingleLine
type App struct {
	ctx           context.Context
	tailCancel    context.CancelFunc
	mu             sync.Mutex
	logEntries     []LogEntry
	isShadow       bool
	lastSentIdx         int
	sessionID           int64
	activeFilePath      string
	fileOffset          int64
	currentFirstLineNum int64
	currentLastLineNum  int64
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		logEntries: make([]LogEntry, 0),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	go a.broadcastLoop()

	// Clean up leftover .old file from a previous self-update
	go func() {
		exe, err := os.Executable()
		if err == nil {
			oldExe := exe + ".old"
			if _, err := os.Stat(oldExe); err == nil {
				os.Remove(oldExe)
			}
		}
	}()
}

// StopTailing explicitly stops tailing the active file, releasing its lock
func (a *App) StopTailing() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.tailCancel != nil {
		a.tailCancel()
		a.tailCancel = nil
	}
	a.activeFilePath = ""
	a.logEntries = make([]LogEntry, 0)
}
