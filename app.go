package main

import (
	"bytes"
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

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

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

type LogUpdate struct {
	NewEntries      []LogEntry `json:"newEntries"`
	LastEntryUpdate *LogEntry  `json:"lastEntryUpdate"`
	ClearLogs       bool       `json:"clearLogs"`
}

// App struct
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
	
	// Reset CWD to prevent Windows from locking the selected directory
	os.Chdir(`C:\`)
	
	return dir
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

type DropResult struct {
	Path  string     `json:"path"`
	Name  string     `json:"name"`
	Files []FileInfo `json:"files"`
	Error string     `json:"error"`
}

func (a *App) ProcessDrop(path string) *DropResult {
	info, err := os.Stat(path)
	if err != nil {
		name := filepath.Base(path)
		if name == "" || name == "." {
			name = path
		}
		return &DropResult{
			Path:  path,
			Name:  name,
			Files: []FileInfo{},
			Error: "Folder inaccessible or deleted",
		}
	}

	dir := path
	if !info.IsDir() {
		dir = filepath.Dir(path)
	}

	files := a.ListFiles(dir)
	name := filepath.Base(dir)
	if name == "" || name == "." {
		name = dir
	}

	return &DropResult{
		Path:  dir,
		Name:  name,
		Files: files,
		Error: "",
	}
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
	
	totalLines := countLinesFast(filePath)
	a.currentFirstLineNum = totalLines + 1
	a.currentLastLineNum = totalLines
	
	currentSession := a.sessionID
	
	ctx, cancel := context.WithCancel(context.Background())
	a.tailCancel = cancel
	a.mu.Unlock()

	go a.tailLogs(ctx, filePath, currentSession)
}

var (
	shadowRegex  = regexp.MustCompile(`^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2})\s+\[(.+?)\]\s+(.*?)\s+hash=([a-zA-Z0-9]+)\s+:(\d+)\s+→\s+(.*?)\s+→\s+(.*)$`)
	bracketRegex = regexp.MustCompile(`^\[(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})\]\s+(.*)$`)
	pythonRegex  = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2},\d{3})\s+-\s+(.*)$`)
	kvTimeRegex  = regexp.MustCompile(`time="?([^"\s]+)"?`)
	kvTagRegex   = regexp.MustCompile(`tag="?([^"\s]+)"?`)
	kvLevelRegex = regexp.MustCompile(`level="?([^"\s]+)"?`)
	timeSplitRegex = regexp.MustCompile(`^(\d{4}[-/]\d{2}[-/]\d{2})\s+(\d{2}:\d{2}:\d{2})(?:[,.](\d+))?\s*(Z|[+-]\d{2}:?\d{2})?`)
)

func parseTimeComponents(rawTime string) (date, timePart, ms, tz string) {
	// Normalize T delimiter from ISO format to space for regex matching
	t := strings.Replace(rawTime, "T", " ", 1)
	
	matches := timeSplitRegex.FindStringSubmatch(t)
	if len(matches) > 0 {
		date = matches[1]
		timePart = matches[2]
		ms = matches[3]
		tz = matches[4]
	} else {
		// Fallback if it doesn't strictly match the expected YYYY-MM-DD HH:MM:SS format
		timePart = rawTime
	}
	return
}

func countLinesFast(filePath string) int64 {
	file, err := os.Open(filePath)
	if err != nil {
		return 0
	}
	defer file.Close()

	buf := make([]byte, 64*1024)
	var count int64
	for {
		c, err := file.Read(buf)
		count += int64(bytes.Count(buf[:c], []byte{'\n'}))
		if err != nil {
			break
		}
	}
	return count
}

func (a *App) tailLogs(ctx context.Context, filePath string, sessionID int64) {
	ticker := time.NewTicker(50 * time.Millisecond) // 50ms polling feels real-time (20 FPS)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.mu.Lock()
			if a.sessionID != sessionID {
				a.mu.Unlock()
				return
			}
			offset := a.fileOffset
			a.mu.Unlock()

			info, err := os.Stat(filePath)
			if err != nil {
				continue // File might be temporarily unavailable
			}

			// Check if file was truncated (e.g. wiped or rotated).
			// Cases handled perfectly:
			// 1. File wiped to 0 bytes: info.Size() < offset -> triggers reset.
			// 2. File wiped and new logs written in same 50ms tick: if new size < old offset, triggers reset and reads new logs immediately.
			// 3. File manually edited to remove last line: triggers reset and re-reads entire file.
			// THEORETICAL BLIND SPOT: If the file is wiped AND more bytes are written than the old offset
			// within a single 50ms window (e.g. writing 100MB of logs in <50ms), this size check won't detect the wipe.
			// If bugs occur where a rotated file isn't cleared in the UI, consider checking file ModTime/CreationTime or inode.
			if info.Size() < offset {
				offset = 0 // File truncated
				a.mu.Lock()
				a.logEntries = make([]LogEntry, 0)
				a.currentFirstLineNum = 1
				a.currentLastLineNum = 0
				a.mu.Unlock()
			} else if info.Size() == offset {
				continue // No new data
			}

			// Open file, read, and close IMMEDIATELY to prevent Windows file locks
			file, err := os.Open(filePath)
			if err != nil {
				continue
			}

			_, err = file.Seek(offset, 0)
			if err != nil {
				file.Close()
				continue
			}

			bytesToRead := info.Size() - offset
			if bytesToRead > 1024*1024*5 {
				bytesToRead = 1024 * 1024 * 5 // Max 5MB per poll to keep it responsive
			}

			buf := make([]byte, bytesToRead)
			n, err := file.Read(buf)
			file.Close() // ALWAYS close immediately

			if n > 0 {
				buf = buf[:n]
				lastNewline := bytes.LastIndexByte(buf, '\n')
				if lastNewline >= 0 {
					linesStr := string(buf[:lastNewline])
					lines := strings.Split(linesStr, "\n")
					for _, line := range lines {
						line = strings.TrimRight(line, "\r")
						if len(line) > 0 {
							a.parseLine(line, sessionID)
						}
					}
					
					a.mu.Lock()
					a.fileOffset = offset + int64(lastNewline) + 1
					a.mu.Unlock()
				}
			}
		}
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
	a.currentLastLineNum++
	entry.LineNum = a.currentLastLineNum
	a.logEntries = append(a.logEntries, entry)
	
	// Keep maximum 50000 entries in memory for live tail
	if len(a.logEntries) > 50000 {
		a.logEntries = a.logEntries[1:]
		if a.lastSentIdx > 0 {
			a.lastSentIdx--
		}
	}
	a.mu.Unlock()
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
			d, t, ms, tz := parseTimeComponents(matches[1])
			entry.Date = d
			entry.Time = t
			entry.Ms = ms
			entry.Tz = tz
			entry.Tag = matches[2]
			entry.Message = fmt.Sprintf("%s hash=%s :%s → %s → %s", matches[3], matches[4], matches[5], matches[6], matches[7])
		} else {
			entry.Message = text
		}
	} else {
		msg := text

		// Step 1: Extract Time
		rawTime := ""
		if matches := bracketRegex.FindStringSubmatch(msg); len(matches) >= 3 {
			rawTime = matches[1]
			msg = matches[2]
		} else if matches := pythonRegex.FindStringSubmatch(msg); len(matches) >= 3 {
			rawTime = matches[1]
			msg = matches[2]
		} else if matches := kvTimeRegex.FindStringSubmatch(msg); len(matches) > 1 {
			rawTime = matches[1]
			msg = kvTimeRegex.ReplaceAllString(msg, "")
		} else {
			if loc := timeSplitRegex.FindStringIndex(msg); loc != nil {
				rawTime = strings.TrimSpace(msg[loc[0]:loc[1]])
				msg = strings.TrimSpace(msg[loc[1]:])
			}
		}
		
		if rawTime != "" {
			d, t, ms, tz := parseTimeComponents(rawTime)
			entry.Date = d
			entry.Time = t
			entry.Ms = ms
			entry.Tz = tz
		}

		// Step 2: Extract explicit Key-Value metadata
		if m := kvLevelRegex.FindStringSubmatch(msg); len(m) > 1 {
			entry.Level = m[1]
			msg = kvLevelRegex.ReplaceAllString(msg, "")
		}
		if m := kvTagRegex.FindStringSubmatch(msg); len(m) > 1 {
			entry.Tag = m[1]
			msg = kvTagRegex.ReplaceAllString(msg, "")
		}

		msg = strings.TrimSpace(msg)

		// Step 3 & 4: Extract preset levels and remaining leading brackets
		for i := 0; i < 2; i++ {
			if strings.HasPrefix(msg, "[") {
				endIdx := strings.Index(msg, "]")
				if endIdx > 0 {
					content := msg[1:endIdx]
					
					isLevel := false
					lowerContent := strings.ToLower(content)
					if lowerContent == "info" || lowerContent == "error" || lowerContent == "warn" || lowerContent == "warning" || lowerContent == "debug" || lowerContent == "fatal" || lowerContent == "trace" {
						isLevel = true
					}
					
					if isLevel && entry.Level == "" {
						entry.Level = content
					} else if entry.Tag == "" {
						entry.Tag = content
					} else {
						// Both Level and Tag are filled, stop stripping brackets
						break
					}
					
					// Consume the bracket from the message
					msg = strings.TrimSpace(msg[endIdx+1:])
				} else {
					break // No matching end bracket found
				}
			} else {
				break // Does not start with a bracket
			}
		}

		// Step 4.5: Check for unbracketed level prefixes like "Warning:" or "ERROR:"
		if entry.Level == "" {
			spaceIdx := strings.Index(msg, " ")
			if spaceIdx > 0 {
				firstWord := msg[:spaceIdx]
				cleanWord := strings.TrimRight(firstWord, ":-")
				lowerWord := strings.ToLower(cleanWord)
				if lowerWord == "info" || lowerWord == "error" || lowerWord == "warn" || lowerWord == "warning" || lowerWord == "debug" || lowerWord == "fatal" || lowerWord == "trace" {
					entry.Level = cleanWord
					msg = strings.TrimSpace(msg[spaceIdx:])
				}
			}
		}

		// Step 5: Finalize Message
		entry.Message = strings.TrimSpace(msg)
	}
	
	entry.EndDate = entry.Date
	entry.EndTime = entry.Time
	entry.EndMs = entry.Ms
	entry.EndTz = entry.Tz
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

	readSize := int64(1024 * 1024) // 1MB chunk
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
	
	var validLines []string
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if line != "" {
			validLines = append(validLines, line)
		}
	}

	a.mu.Lock()
	a.currentFirstLineNum -= int64(len(validLines))
	startLine := a.currentFirstLineNum
	a.mu.Unlock()

	var prepended []LogEntry
	for i, line := range validLines {
		entry := a.parseSingleLine(line, isShadow)
		entry.LineNum = startLine + int64(i)
		prepended = append(prepended, entry)
	}

	return prepended
}

func (a *App) broadcastLoop() {
	ticker := time.NewTicker(50 * time.Millisecond) // 50ms UI update for near real-time rendering
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			a.mu.Lock()
			
			var update LogUpdate
			if a.lastSentIdx > len(a.logEntries) {
				update.ClearLogs = true
				a.lastSentIdx = 0
			}
			
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
			
			if update.ClearLogs || len(update.NewEntries) > 0 || update.LastEntryUpdate != nil {
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
