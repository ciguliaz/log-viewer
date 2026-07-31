package tailer

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"log-viewer/internal/models"
	"log-viewer/internal/parser"
)

type Tailer struct {
	ctx                 context.Context
	tailCancel          context.CancelFunc
	mu                  sync.Mutex
	logEntries          []models.LogEntry
	isShadow            bool
	lastSentIdx         int
	sessionID           int64
	activeFilePath      string
	fileOffset          int64
	currentFirstLineNum int64
	currentLastLineNum  int64
}

func NewTailer(ctx context.Context) *Tailer {
	t := &Tailer{
		ctx:        ctx,
		logEntries: make([]models.LogEntry, 0),
	}
	go t.broadcastLoop()
	return t
}

func (t *Tailer) parseLine(text string, sessionID int64) {
	t.mu.Lock()
	if sessionID != t.sessionID {
		t.mu.Unlock()
		return
	}
	isShadow := t.isShadow
	t.mu.Unlock()

	entry := parser.ParseLine(text, isShadow)

	t.mu.Lock()
	t.currentLastLineNum++
	entry.LineNum = t.currentLastLineNum
	t.logEntries = append(t.logEntries, entry)

	// Keep maximum 50000 entries in memory for live tail
	if len(t.logEntries) > 50000 {
		t.logEntries = t.logEntries[1:]
		if t.lastSentIdx > 0 {
			t.lastSentIdx--
		}
	}
	t.mu.Unlock()
}

// StartTailing stops any existing tail and starts tailing a new file
func (t *Tailer) StartTailing(filePath string) {
	t.mu.Lock()
	if t.tailCancel != nil {
		t.tailCancel()
	}

	t.logEntries = make([]models.LogEntry, 0)
	t.lastSentIdx = 0
	t.isShadow = filepath.Base(filePath) == "shadow.log"
	t.activeFilePath = filePath
	t.sessionID = time.Now().UnixNano()

	fileInfo, err := os.Stat(filePath)
	if err == nil {
		t.fileOffset = fileInfo.Size()
	} else {
		t.fileOffset = 0
	}

	totalLines := countLinesFast(filePath)
	t.currentFirstLineNum = totalLines + 1
	t.currentLastLineNum = totalLines

	currentSession := t.sessionID

	ctx, cancel := context.WithCancel(context.Background())
	t.tailCancel = cancel
	t.mu.Unlock()

	go t.tailLogs(ctx, filePath, currentSession)
}

// StopTailing explicitly stops tailing the active file, releasing its lock
func (t *Tailer) StopTailing() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.tailCancel != nil {
		t.tailCancel()
		t.tailCancel = nil
	}
	t.activeFilePath = ""
	t.logEntries = make([]models.LogEntry, 0)
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

func (t *Tailer) tailLogs(ctx context.Context, filePath string, sessionID int64) {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.mu.Lock()
			if t.sessionID != sessionID {
				t.mu.Unlock()
				return
			}
			offset := t.fileOffset
			t.mu.Unlock()

			info, err := os.Stat(filePath)
			if err != nil {
				continue
			}

			if info.Size() < offset {
				offset = 0 // File truncated
				t.mu.Lock()
				t.logEntries = make([]models.LogEntry, 0)
				t.currentFirstLineNum = 1
				t.currentLastLineNum = 0
				t.mu.Unlock()
			} else if info.Size() == offset {
				continue // No new data
			}

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
				bytesToRead = 1024 * 1024 * 5
			}

			buf := make([]byte, bytesToRead)
			n, err := file.Read(buf)
			file.Close()

			if n > 0 {
				buf = buf[:n]
				lastNewline := bytes.LastIndexByte(buf, '\n')
				if lastNewline >= 0 {
					linesStr := string(buf[:lastNewline])
					lines := strings.Split(linesStr, "\n")
					for _, line := range lines {
						line = strings.TrimRight(line, "\r")
						if len(line) > 0 {
							t.parseLine(line, sessionID)
						}
					}

					t.mu.Lock()
					t.fileOffset = offset + int64(lastNewline) + 1
					t.mu.Unlock()
				}
			}
		}
	}
}

// LoadPreviousChunk reads a 1MB chunk of logs backwards from the current fileOffset
func (t *Tailer) LoadPreviousChunk() []models.LogEntry {
	t.mu.Lock()
	filePath := t.activeFilePath
	offset := t.fileOffset
	isShadow := t.isShadow
	t.mu.Unlock()

	if offset <= 0 || filePath == "" {
		return nil
	}

	readSize := int64(1024 * 1024)
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

	t.mu.Lock()
	t.fileOffset = newOffset
	t.mu.Unlock()

	text := string(buffer[startIdx:])
	lines := strings.Split(text, "\n")

	var validLines []string
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if line != "" {
			validLines = append(validLines, line)
		}
	}

	t.mu.Lock()
	t.currentFirstLineNum -= int64(len(validLines))
	startLine := t.currentFirstLineNum
	t.mu.Unlock()

	var prepended []models.LogEntry
	for i, line := range validLines {
		entry := parser.ParseLine(line, isShadow)
		entry.LineNum = startLine + int64(i)
		prepended = append(prepended, entry)
	}

	return prepended
}

func (t *Tailer) broadcastLoop() {
	ticker := time.NewTicker(50 * time.Millisecond)
	for {
		select {
		case <-t.ctx.Done():
			return
		case <-ticker.C:
			t.mu.Lock()

			var update models.LogUpdate
			if t.lastSentIdx > len(t.logEntries) {
				update.ClearLogs = true
				t.lastSentIdx = 0
			}

			if t.lastSentIdx > 0 && t.lastSentIdx <= len(t.logEntries) {
				last := t.logEntries[t.lastSentIdx-1]
				update.LastEntryUpdate = &last
			}

			if t.lastSentIdx < len(t.logEntries) {
				update.NewEntries = make([]models.LogEntry, len(t.logEntries)-t.lastSentIdx)
				copy(update.NewEntries, t.logEntries[t.lastSentIdx:])
				t.lastSentIdx = len(t.logEntries)
			}

			t.mu.Unlock()

			if update.ClearLogs || len(update.NewEntries) > 0 || update.LastEntryUpdate != nil {
				runtime.EventsEmit(t.ctx, "log_update", update)
			}
		}
	}
}

// GetInitialLogs allows frontend to fetch immediately on load
func (t *Tailer) GetInitialLogs() []models.LogEntry {
	t.mu.Lock()
	defer t.mu.Unlock()
	list := make([]models.LogEntry, len(t.logEntries))
	copy(list, t.logEntries)
	t.lastSentIdx = len(t.logEntries)
	return list
}
