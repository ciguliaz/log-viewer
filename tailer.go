package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

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

// LoadPreviousChunk reads a 1MB chunk of logs backwards from the current fileOffset
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
