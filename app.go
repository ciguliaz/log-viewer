package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/nxadm/tail"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type LogEntry struct {
	Id      string `json:"id"`
	Time    string `json:"time"`
	Level   string `json:"level"`
	Tag     string `json:"tag"`
	Message string `json:"message"`
	Raw     string `json:"raw"`
}

// App struct
type App struct {
	ctx           context.Context
	currentTail   *tail.Tail
	tailCancel    context.CancelFunc
	mu            sync.Mutex
	logEntries    []LogEntry
	isShadow      bool
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
}

// SelectFolder opens a dialog to select a directory
func (a *App) SelectFolder() string {
	opts := runtime.OpenDialogOptions{
		DefaultDirectory: `C:\Program Files (x86)\hatacone\logs`,
		Title:            "Select Log Folder",
	}
	dir, err := runtime.OpenDirectoryDialog(a.ctx, opts)
	if err != nil {
		fmt.Println("Error selecting directory:", err)
		return ""
	}
	return dir
}

type FileInfo struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// ListFiles lists .log files in a given directory
func (a *App) ListFiles(dirPath string) []FileInfo {
	var files []FileInfo
	if dirPath == "" {
		return files
	}
	
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		fmt.Println("Error reading directory:", err)
		return files
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".log" {
			files = append(files, FileInfo{
				Name: entry.Name(),
				Path: filepath.Join(dirPath, entry.Name()),
			})
		}
	}
	return files
}

// StartTailing stops any existing tail and starts tailing a new file
func (a *App) StartTailing(filePath string) {
	a.mu.Lock()
	if a.tailCancel != nil {
		a.tailCancel()
	}
	
	a.logEntries = make([]LogEntry, 0)
	a.isShadow = filepath.Base(filePath) == "shadow.log"
	
	ctx, cancel := context.WithCancel(context.Background())
	a.tailCancel = cancel
	a.mu.Unlock()

	go a.tailLogs(ctx, filePath)
}

var shadowRegex = regexp.MustCompile(`^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2})\s+\[(.+?)\]\s+(.*?)\s+hash=([a-zA-Z0-9]+)\s+:(\d+)\s+→\s+(.*?)\s+→\s+(.*)$`)
var bracketRegex = regexp.MustCompile(`^\[(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})\]\s+(.*)$`)

func (a *App) tailLogs(ctx context.Context, filePath string) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Println("Log file does not exist:", filePath)
		return
	}

	t, err := tail.TailFile(filePath, tail.Config{
		Follow:    true,
		ReOpen:    true,
		MustExist: false,
		Poll:      true, // Use polling on Windows to avoid CancelIo/CloseHandle fsnotify errors
		Location:  &tail.SeekInfo{Offset: 0, Whence: 2}, // Tail from end
		Logger:    tail.DiscardingLogger,
	})
	if err != nil {
		fmt.Println("Error tailing log:", err)
		return
	}

	a.mu.Lock()
	a.currentTail = t
	a.mu.Unlock()

	// Stop tailing if context is cancelled
	go func() {
		<-ctx.Done()
		t.Stop()
	}()

	for line := range t.Lines {
		a.parseLine(line.Text)
	}
}

func (a *App) parseLine(text string) {
	var entry LogEntry
	entry.Id = fmt.Sprintf("%d", time.Now().UnixNano())
	entry.Raw = text
	
	if a.isShadow {
		matches := shadowRegex.FindStringSubmatch(text)
		if len(matches) >= 8 {
			entry.Time = matches[1]
			entry.Tag = matches[2]
			entry.Message = fmt.Sprintf("%s hash=%s :%s → %s → %s", matches[3], matches[4], matches[5], matches[6], matches[7])
		} else {
			entry.Message = text
		}
	} else {
		// Generic parse `[YYYY-MM-DD HH:MM:SS] Message`
		matches := bracketRegex.FindStringSubmatch(text)
		if len(matches) >= 3 {
			entry.Time = matches[1]
			entry.Message = matches[2]
		} else {
			entry.Message = text
		}
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	a.logEntries = append(a.logEntries, entry)
	
	// Keep maximum 1000 entries in memory
	if len(a.logEntries) > 1000 {
		a.logEntries = a.logEntries[1:]
	}
}

func (a *App) broadcastLoop() {
	ticker := time.NewTicker(200 * time.Millisecond) // Update UI 5 times per second for smooth scrolling
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			a.mu.Lock()
			// Only broadcast if there are entries to avoid heavy JSON payload if nothing changed
			// For simplicity in a log viewer, we just send the whole list. In production, we'd send diffs.
			list := make([]LogEntry, len(a.logEntries))
			copy(list, a.logEntries)
			a.mu.Unlock()
			runtime.EventsEmit(a.ctx, "log_update", list)
		}
	}
}

// GetInitialLogs allows frontend to fetch immediately on load
func (a *App) GetInitialLogs() []LogEntry {
	a.mu.Lock()
	defer a.mu.Unlock()
	list := make([]LogEntry, len(a.logEntries))
	copy(list, a.logEntries)
	return list
}
