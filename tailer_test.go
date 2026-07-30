package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	app := NewApp()
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "tail.log")
	
	// Start with empty file
	os.WriteFile(logPath, []byte(""), 0644)
	
	// We need to bypass StartTailing because it creates a background context we can't easily wait on.
	// We will manually setup the state and call tailLogs with a cancelable context.
	app.logEntries = make([]LogEntry, 0)
	app.activeFilePath = logPath
	app.sessionID = 1
	app.fileOffset = 0
	app.currentFirstLineNum = 1
	app.currentLastLineNum = 0
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	go app.tailLogs(ctx, logPath, 1)
	
	// A1. Normal Append
	os.WriteFile(logPath, []byte("2023-10-25 10:00:00 INFO Line 1\n"), 0644)
	time.Sleep(300 * time.Millisecond) // Wait for 50ms tick
	
	app.mu.Lock()
	count := len(app.logEntries)
	msg := ""
	if count > 0 {
		msg = app.logEntries[0].Message
	}
	app.mu.Unlock()
	
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
	
	app.mu.Lock()
	count = len(app.logEntries)
	app.mu.Unlock()
	
	if count != 3 {
		t.Errorf("A2. Expected 3 log entries total, got %d", count)
	}
	
	// C1/C2. File Wiped/Truncated
	// Simulate log rotation (wiped to 0 bytes, then new logs written)
	os.WriteFile(logPath, []byte("2023-10-25 10:00:03 INFO New Line 1 after rotate\n"), 0644)
	time.Sleep(300 * time.Millisecond)
	
	app.mu.Lock()
	count = len(app.logEntries)
	offset := app.fileOffset
	if count > 0 {
		msg = app.logEntries[0].Message
	}
	app.mu.Unlock()
	
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
	app := NewApp()
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "history.log")
	
	// Write a 3 line file
	os.WriteFile(logPath, []byte("Line 1\nLine 2\nLine 3\n"), 0644)
	
	app.activeFilePath = logPath
	app.fileOffset = 21 // Size of "Line 1\nLine 2\nLine 3\n"
	app.currentFirstLineNum = 4
	app.currentLastLineNum = 3
	
	// Load history
	chunk := app.LoadPreviousChunk()
	
	if len(chunk) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(chunk))
	}
	if len(chunk) > 0 && chunk[0].Message != "Line 1" {
		t.Errorf("Expected first line to be 'Line 1', got '%s'", chunk[0].Message)
	}
	
	app.mu.Lock()
	offset := app.fileOffset
	firstLine := app.currentFirstLineNum
	app.mu.Unlock()
	
	if offset != 0 {
		t.Errorf("Expected offset to reach 0, got %d", offset)
	}
	if firstLine != 1 {
		t.Errorf("Expected firstLine to reach 1, got %d", firstLine)
	}
	
	// Test loading when offset is 0
	chunk2 := app.LoadPreviousChunk()
	if chunk2 != nil {
		t.Errorf("Expected nil when offset is 0, got %v", chunk2)
	}
	
	// Test file missing
	app.activeFilePath = "does_not_exist.log"
	app.fileOffset = 100
	chunk3 := app.LoadPreviousChunk()
	if chunk3 != nil {
		t.Errorf("Expected nil when file missing, got %v", chunk3)
	}
}

func TestGetInitialLogs(t *testing.T) {
	app := NewApp()
	app.logEntries = []LogEntry{ {Message: "A"}, {Message: "B"} }
	app.lastSentIdx = 0
	
	logs := app.GetInitialLogs()
	if len(logs) != 2 {
		t.Errorf("Expected 2 logs, got %d", len(logs))
	}
	
	if app.lastSentIdx != 2 {
		t.Errorf("Expected lastSentIdx to be updated to 2, got %d", app.lastSentIdx)
	}
}

func TestStopTailing(t *testing.T) {
	app := NewApp()
	
	ctx, cancel := context.WithCancel(context.Background())
	app.tailCancel = cancel
	app.activeFilePath = "test.log"
	app.logEntries = []LogEntry{ {Message: "A"} }
	
	app.StopTailing()
	
	if app.tailCancel != nil {
		t.Error("tailCancel should be nil")
	}
	if app.activeFilePath != "" {
		t.Error("activeFilePath should be empty")
	}
	if len(app.logEntries) != 0 {
		t.Error("logEntries should be cleared")
	}
	
	// Test context cancellation
	err := ctx.Err()
	if err == nil {
		t.Error("Context should be canceled")
	}
}

func TestStartTailing(t *testing.T) {
	app := NewApp()
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "start.log")
	os.WriteFile(logPath, []byte("Hello\nWorld\n"), 0644)
	
	// Need a dummy context for the tailLogs to not hang forever
	ctx, cancel := context.WithCancel(context.Background())
	app.ctx = ctx
	defer cancel()
	
	app.StartTailing(logPath)
	
	app.mu.Lock()
	cancelTail := app.tailCancel
	offset := app.fileOffset
	totalLines := app.currentFirstLineNum
	app.mu.Unlock()
	
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
	app.StartTailing(filepath.Join(tmpDir, "missing.log"))
	
	// Wait for goroutine to hit error path
	time.Sleep(20 * time.Millisecond)
	
	// Test tailing a directory (triggers file read error/EOF loops but covers branch)
	app.StartTailing(tmpDir)
	time.Sleep(20 * time.Millisecond)
	
	app.mu.Lock()
	offset2 := app.fileOffset
	app.mu.Unlock()
	
	if offset2 != 0 {
		t.Errorf("Expected offset 0 for missing file, got %d", offset2)
	}
}
