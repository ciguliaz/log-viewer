package tailer

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"log-viewer/internal/models"
)

func TestCountLinesFast(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")
	
	// Write 5 lines (4 newlines)
	os.WriteFile(logPath, []byte("Line 1\nLine 2\nLine 3\nLine 4\nLine 5"), 0644)
	
	count := countLinesFast(logPath)
	if count != 4 {
		t.Errorf("Expected 4 newlines, got %d", count)
	}
	
	// Test empty file
	os.WriteFile(logPath, []byte(""), 0644)
	count = countLinesFast(logPath)
	if count != 0 {
		t.Errorf("Expected 0 newlines, got %d", count)
	}
	
	// Test missing file
	count = countLinesFast(filepath.Join(tmpDir, "missing.log"))
	if count != 0 {
		t.Errorf("Expected 0 newlines for missing file, got %d", count)
	}
}

func TestTailLogs_Integration(t *testing.T) {
	tailer := NewTailer(context.Background())
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "tail.log")
	
	// Start with empty file
	os.WriteFile(logPath, []byte(""), 0644)
	
	// We need to bypass StartTailing because it creates a background context we can't easily wait on.
	// We will manually setup the state and call tailLogs with a cancelable context.
	tailer.logEntries = make([]models.LogEntry, 0)
	tailer.activeFilePath = logPath
	tailer.sessionID = 1
	tailer.fileOffset = 0
	tailer.currentFirstLineNum = 1
	tailer.currentLastLineNum = 0
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	go tailer.tailLogs(ctx, logPath, 1)
	
	// A1. Normal Append
	os.WriteFile(logPath, []byte("2023-10-25 10:00:00 INFO Line 1\n"), 0644)
	time.Sleep(300 * time.Millisecond) // Wait for 50ms tick
	
	tailer.mu.Lock()
	count := len(tailer.logEntries)
	msg := ""
	if count > 0 {
		msg = tailer.logEntries[0].Message
	}
	tailer.mu.Unlock()
	
	if count != 1 {
		t.Errorf("A1. Expected 1 log entry, got %d", count)
	}
	if msg != "Line 1" {
		t.Errorf("A1. Expected msg 'Line 1', got '%s'", msg)
	}
	
	// A2. Fast Append (write multiple lines at once)
	f, _ := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0644)
	f.WriteString("2023-10-25 10:00:01 INFO Line 2\n2023-10-25 10:00:02 INFO Line 3\n")
	f.Close()
	time.Sleep(300 * time.Millisecond)
	
	tailer.mu.Lock()
	count = len(tailer.logEntries)
	tailer.mu.Unlock()
	
	if count != 3 {
		t.Errorf("A2. Expected 3 log entries total, got %d", count)
	}
	
	// C1/C2. File Wiped/Truncated
	// Simulate log rotation (wiped to 0 bytes, then new logs written)
	os.WriteFile(logPath, []byte("2023-10-25 10:00:03 INFO New Line 1 after rotate\n"), 0644)
	time.Sleep(300 * time.Millisecond)
	
	tailer.mu.Lock()
	count = len(tailer.logEntries)
	offset := tailer.fileOffset
	if count > 0 {
		msg = tailer.logEntries[0].Message
	}
	tailer.mu.Unlock()
	
	if count != 1 {
		t.Errorf("C1. Expected logEntries to reset to 1, got %d", count)
	}
	if !strings.Contains(msg, "rotate") {
		t.Errorf("C1. Expected new message, got '%s'", msg)
	}
	if offset <= 0 {
		t.Errorf("C1. Offset should be > 0, got %d", offset)
	}
	
	cancel() // Stop the goroutine
}

func TestLoadPreviousChunk(t *testing.T) {
	tailer := NewTailer(context.Background())
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "history.log")
	
	// Write a 3 line file
	os.WriteFile(logPath, []byte("Line 1\nLine 2\nLine 3\n"), 0644)
	
	tailer.activeFilePath = logPath
	tailer.fileOffset = 21 // Size of "Line 1\nLine 2\nLine 3\n"
	tailer.currentFirstLineNum = 4
	tailer.currentLastLineNum = 3
	
	// Load history
	chunk := tailer.LoadPreviousChunk()
	
	if len(chunk) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(chunk))
	}
	if len(chunk) > 0 && chunk[0].Message != "Line 1" {
		t.Errorf("Expected first line to be 'Line 1', got '%s'", chunk[0].Message)
	}
	
	tailer.mu.Lock()
	offset := tailer.fileOffset
	firstLine := tailer.currentFirstLineNum
	tailer.mu.Unlock()
	
	if offset != 0 {
		t.Errorf("Expected offset to reach 0, got %d", offset)
	}
	if firstLine != 1 {
		t.Errorf("Expected firstLine to reach 1, got %d", firstLine)
	}
	
	// Test loading when offset is 0
	chunk2 := tailer.LoadPreviousChunk()
	if chunk2 != nil {
		t.Errorf("Expected nil when offset is 0, got %v", chunk2)
	}
	
	// Test file missing
	tailer.activeFilePath = "does_not_exist.log"
	tailer.fileOffset = 100
	chunk3 := tailer.LoadPreviousChunk()
	if chunk3 != nil {
		t.Errorf("Expected nil when file missing, got %v", chunk3)
	}
}

func TestGetInitialLogs(t *testing.T) {
	tailer := NewTailer(context.Background())
	tailer.logEntries = []models.LogEntry{ {Message: "A"}, {Message: "B"} }
	tailer.lastSentIdx = 0
	
	logs := tailer.GetInitialLogs()
	if len(logs) != 2 {
		t.Errorf("Expected 2 logs, got %d", len(logs))
	}
	
	if tailer.lastSentIdx != 2 {
		t.Errorf("Expected lastSentIdx to be updated to 2, got %d", tailer.lastSentIdx)
	}
}

func TestStopTailing(t *testing.T) {
	tailer := NewTailer(context.Background())
	
	ctx, cancel := context.WithCancel(context.Background())
	tailer.tailCancel = cancel
	tailer.activeFilePath = "test.log"
	tailer.logEntries = []models.LogEntry{ {Message: "A"} }
	
	tailer.StopTailing()
	
	if tailer.tailCancel != nil {
		t.Error("tailCancel should be nil")
	}
	if tailer.activeFilePath != "" {
		t.Error("activeFilePath should be empty")
	}
	if len(tailer.logEntries) != 0 {
		t.Error("logEntries should be cleared")
	}
	
	// Test context cancellation
	err := ctx.Err()
	if err == nil {
		t.Error("Context should be canceled")
	}
}

func TestStartTailing(t *testing.T) {
	tailer := NewTailer(context.Background())
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "start.log")
	os.WriteFile(logPath, []byte("Hello\nWorld\n"), 0644)
	
	// Need a dummy context for the tailLogs to not hang forever
	ctx, cancel := context.WithCancel(context.Background())
	tailer.ctx = ctx
	defer cancel()
	
	tailer.StartTailing(logPath)
	
	tailer.mu.Lock()
	cancelTail := tailer.tailCancel
	offset := tailer.fileOffset
	totalLines := tailer.currentFirstLineNum
	tailer.mu.Unlock()
	
	if cancelTail == nil {
		t.Error("tailCancel should be set")
	}
	if offset == 0 {
		t.Error("fileOffset should be > 0 for existing file")
	}
	if totalLines != 3 { // countLinesFast returns 2, currentFirstLineNum = 2 + 1 = 3
		t.Errorf("Expected currentFirstLineNum 3, got %d", totalLines)
	}
	
	// Test stopping via StartTailing (stops previous)
	tailer.StartTailing(filepath.Join(tmpDir, "missing.log"))
	
	// Wait for goroutine to hit error path
	time.Sleep(20 * time.Millisecond)
	
	// Test tailing a directory (triggers file read error/EOF loops but covers branch)
	tailer.StartTailing(tmpDir)
	time.Sleep(20 * time.Millisecond)
	
	tailer.mu.Lock()
	offset2 := tailer.fileOffset
	tailer.mu.Unlock()
	
	if offset2 != 0 {
		t.Errorf("Expected offset 0 for missing file, got %d", offset2)
	}
}
