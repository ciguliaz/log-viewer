package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
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
	EndTime string `json:"endTime"`
	Count   int    `json:"count"`
}

type LogUpdate struct {
	NewEntries      []LogEntry `json:"newEntries"`
	LastEntryUpdate *LogEntry  `json:"lastEntryUpdate"`
}

// App struct
type App struct {
	ctx           context.Context
	currentTail   *tail.Tail
	tailCancel    context.CancelFunc
	mu             sync.Mutex
	logEntries     []LogEntry
	isShadow       bool
	lastSentIdx    int
	sessionID      int64
	activeFilePath string
	fileOffset     int64
}

var idCounter int64

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

func getDefaultDir() string {
	dir := `C:\Program Files (x86)\hatacone\logs`
	if _, err := os.Stat(dir); err == nil {
		return dir
	}
	
	if _, err := os.Stat(`C:\`); err == nil {
		return `C:\`
	}
	
	for _, drive := range "DEFGHIJKLMNOPQRSTUVWXYZ" {
		path := string(drive) + `:\`
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	
	return ""
}

// SelectFolder opens a dialog to select a directory
func (a *App) SelectFolder() string {
	opts := runtime.OpenDialogOptions{
		DefaultDirectory: getDefaultDir(),
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
	a.lastSentIdx = 0
	a.isShadow = filepath.Base(filePath) == "shadow.log"
	a.activeFilePath = filePath
	a.sessionID = time.Now().UnixNano()
	
	fileInfo, err := os.Stat(filePath)
	if err == nil {
		a.fileOffset = fileInfo.Size()
	} else {
		a.fileOffset = 0
	}
	
	currentSession := a.sessionID
	
	ctx, cancel := context.WithCancel(context.Background())
	a.tailCancel = cancel
	a.mu.Unlock()

	go a.tailLogs(ctx, filePath, currentSession)
}

var (
	shadowRegex  = regexp.MustCompile(`^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2})\s+\[(.+?)\]\s+(.*?)\s+hash=([a-zA-Z0-9]+)\s+:(\d+)\s+→\s+(.*?)\s+→\s+(.*)$`)
	bracketRegex = regexp.MustCompile(`^\[(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})\]\s+(.*)$`)
	kvTimeRegex  = regexp.MustCompile(`time="?([^"\s]+)"?`)
	kvTagRegex   = regexp.MustCompile(`tag="?([^"\s]+)"?`)
)

func (a *App) tailLogs(ctx context.Context, filePath string, sessionID int64) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Println("Log file does not exist:", filePath)
		return
	}

	t, err := tail.TailFile(filePath, tail.Config{
		Follow:    true,
		ReOpen:    true,
		MustExist: false,
		Poll:      true, // Use polling on Windows to avoid CancelIo/CloseHandle fsnotify errors
		Location:  &tail.SeekInfo{Offset: 0, Whence: 2}, // Tail from EOF since we lazy load history
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
		a.parseLine(line.Text, sessionID)
	}
}

func (a *App) parseLine(text string, sessionID int64) {
	a.mu.Lock()
	if sessionID != a.sessionID {
		a.mu.Unlock()
		return
	}
	isShadow := a.isShadow
	a.mu.Unlock()
	
	entry := a.parseSingleLine(text, isShadow)

	a.mu.Lock()
	defer a.mu.Unlock()
	
	a.logEntries = append(a.logEntries, entry)
	
	// Keep maximum 50000 entries in memory for live tail
	if len(a.logEntries) > 50000 {
		a.logEntries = a.logEntries[1:]
		if a.lastSentIdx > 0 {
			a.lastSentIdx--
		}
	}
}

func (a *App) parseSingleLine(text string, isShadow bool) LogEntry {
	text = strings.TrimRight(text, "\r\n")
	var entry LogEntry
	id := atomic.AddInt64(&idCounter, 1)
	entry.Id = fmt.Sprintf("%d-%d", time.Now().UnixNano(), id)
	entry.Raw = text
	entry.Count = 1
	
	if isShadow {
		matches := shadowRegex.FindStringSubmatch(text)
		if len(matches) >= 8 {
			entry.Time = matches[1]
			entry.Tag = matches[2]
			entry.Message = fmt.Sprintf("%s hash=%s :%s → %s → %s", matches[3], matches[4], matches[5], matches[6], matches[7])
		} else {
			entry.Message = text
		}
	} else {
		// Try bracket parse `[YYYY-MM-DD HH:MM:SS] Message`
		if matches := bracketRegex.FindStringSubmatch(text); len(matches) >= 3 {
			entry.Time = matches[1]
			entry.Message = matches[2]
		} else if matches := kvTimeRegex.FindStringSubmatch(text); len(matches) > 1 {
			// Try key-value parse like time="2026-07-08T16:40:38+07:00"
			entry.Time = matches[1]
			
			if m := kvTagRegex.FindStringSubmatch(text); len(m) > 1 {
				entry.Tag = m[1]
			}
			
			// Remove the time and tag segments from the message since they are displayed in other columns
			cleanMsg := kvTimeRegex.ReplaceAllString(text, "")
			cleanMsg = kvTagRegex.ReplaceAllString(cleanMsg, "")
			entry.Message = strings.TrimSpace(cleanMsg)
		} else {
			entry.Message = text
		}
	}
	
	entry.EndTime = entry.Time
	return entry
}

// LoadPreviousChunk reads a 64KB chunk of logs backwards from the current fileOffset
func (a *App) LoadPreviousChunk() []LogEntry {
	a.mu.Lock()
	filePath := a.activeFilePath
	offset := a.fileOffset
	isShadow := a.isShadow
	a.mu.Unlock()

	if offset <= 0 || filePath == "" {
		return nil
	}

	readSize := int64(64 * 1024) // 64KB chunk
	if offset < readSize {
		readSize = offset
	}

	newOffset := offset - readSize
	file, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer file.Close()

	buffer := make([]byte, readSize)
	_, err = file.ReadAt(buffer, newOffset)
	if err != nil && err != io.EOF {
		return nil
	}

	// Find the first newline to avoid parsing partial lines
	startIdx := 0
	if newOffset > 0 {
		for i := 0; i < len(buffer); i++ {
			if buffer[i] == '\n' {
				startIdx = i + 1
				break
			}
		}
		newOffset += int64(startIdx)
	}

	a.mu.Lock()
	a.fileOffset = newOffset
	a.mu.Unlock()

	text := string(buffer[startIdx:])
	lines := strings.Split(text, "\n")
	
	var prepended []LogEntry
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		entry := a.parseSingleLine(line, isShadow)
		prepended = append(prepended, entry)
	}

	return prepended
}

func (a *App) broadcastLoop() {
	ticker := time.NewTicker(200 * time.Millisecond) // Update UI 5 times per second for smooth scrolling
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			a.mu.Lock()
			
			var update LogUpdate
			if a.lastSentIdx > 0 && a.lastSentIdx <= len(a.logEntries) {
				last := a.logEntries[a.lastSentIdx-1]
				update.LastEntryUpdate = &last
			}
			
			if a.lastSentIdx < len(a.logEntries) {
				update.NewEntries = make([]LogEntry, len(a.logEntries)-a.lastSentIdx)
				copy(update.NewEntries, a.logEntries[a.lastSentIdx:])
				a.lastSentIdx = len(a.logEntries)
			}
			
			a.mu.Unlock()
			
			if len(update.NewEntries) > 0 || update.LastEntryUpdate != nil {
				runtime.EventsEmit(a.ctx, "log_update", update)
			}
		}
	}
}

// GetInitialLogs allows frontend to fetch immediately on load
func (a *App) GetInitialLogs() []LogEntry {
	a.mu.Lock()
	defer a.mu.Unlock()
	list := make([]LogEntry, len(a.logEntries))
	copy(list, a.logEntries)
	a.lastSentIdx = len(a.logEntries)
	return list
}
