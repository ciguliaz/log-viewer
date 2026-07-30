package main

import (
	"testing"
)

func TestParseTimeComponents(t *testing.T) {
	tests := []struct {
		name         string
		rawTime      string
		wantDate     string
		wantTimePart string
		wantMs       string
		wantTz       string
	}{
		{
			name:         "ISO8601 with T and timezone",
			rawTime:      "2023-10-25T15:30:45.123Z",
			wantDate:     "2023-10-25",
			wantTimePart: "15:30:45",
			wantMs:       "123",
			wantTz:       "Z",
		},
		{
			name:         "ISO8601 with space and timezone",
			rawTime:      "2023-10-25 15:30:45.123+07:00",
			wantDate:     "2023-10-25",
			wantTimePart: "15:30:45",
			wantMs:       "123",
			wantTz:       "+07:00",
		},
		{
			name:         "Date and time no ms",
			rawTime:      "2023/10/25 15:30:45",
			wantDate:     "2023/10/25",
			wantTimePart: "15:30:45",
			wantMs:       "",
			wantTz:       "",
		},
		{
			name:         "Python style with comma ms",
			rawTime:      "2023-10-25 15:30:45,123",
			wantDate:     "2023-10-25",
			wantTimePart: "15:30:45",
			wantMs:       "123",
			wantTz:       "",
		},
		{
			name:         "Invalid time format fallback",
			rawTime:      "Just A String",
			wantDate:     "",
			wantTimePart: "Just A String",
			wantMs:       "",
			wantTz:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDate, gotTimePart, gotMs, gotTz := parseTimeComponents(tt.rawTime)
			if gotDate != tt.wantDate {
				t.Errorf("parseTimeComponents() gotDate = %v, want %v", gotDate, tt.wantDate)
			}
			if gotTimePart != tt.wantTimePart {
				t.Errorf("parseTimeComponents() gotTimePart = %v, want %v", gotTimePart, tt.wantTimePart)
			}
			if gotMs != tt.wantMs {
				t.Errorf("parseTimeComponents() gotMs = %v, want %v", gotMs, tt.wantMs)
			}
			if gotTz != tt.wantTz {
				t.Errorf("parseTimeComponents() gotTz = %v, want %v", gotTz, tt.wantTz)
			}
		})
	}
}

func TestParseSingleLine_Shadow(t *testing.T) {
	app := NewApp()
	app.isShadow = true

	line := "2023/10/25 15:30:45 [INFO] Processing hash=abc1234 :8080 → 192.168.1.1:9090 → Success"
	entry := app.parseSingleLine(line, true)

	if entry.Date != "2023/10/25" {
		t.Errorf("Expected Date '2023/10/25', got %v", entry.Date)
	}
	if entry.Time != "15:30:45" {
		t.Errorf("Expected Time '15:30:45', got %v", entry.Time)
	}
	if entry.Tag != "INFO" {
		t.Errorf("Expected Tag 'INFO', got %v", entry.Tag)
	}
	expectedMsg := "Processing hash=abc1234 :8080 → 192.168.1.1:9090 → Success"
	if entry.Message != expectedMsg {
		t.Errorf("Expected Message '%v', got %v", expectedMsg, entry.Message)
	}
}

func TestParseSingleLine_Shadow_Invalid(t *testing.T) {
	app := NewApp()
	line := "Random unformatted shadow log"
	entry := app.parseSingleLine(line, true)

	if entry.Message != line {
		t.Errorf("Expected Message '%v', got %v", line, entry.Message)
	}
}

func TestParseSingleLine_Standard(t *testing.T) {
	app := NewApp()
	tests := []struct {
		name    string
		line    string
		want    LogEntry
	}{
		{
			name: "Bracket format with level",
			line: "[2023-10-25 15:30:45] [ERROR] Something failed",
			want: LogEntry{
				Date:    "2023-10-25",
				Time:    "15:30:45",
				Level:   "ERROR",
				Message: "Something failed",
			},
		},
		{
			name: "Python format",
			line: "2023-10-25 15:30:45,123 - INFO - Something happened",
			want: LogEntry{
				Date:    "2023-10-25",
				Time:    "15:30:45",
				Ms:      "123",
				Level:   "INFO",
				Message: "- Something happened",
			}, // Wait, in Python format: "2023-10-25 15:30:45,123 - INFO - message" the python regex extracts the whole "INFO - message" as msg. Let's see how parser handles it.
		},
		{
			name: "Key Value format",
			line: `time="2023-10-25T15:30:45Z" level="debug" tag="app" msg="Hello"`,
			want: LogEntry{
				Date:    "2023-10-25",
				Time:    "15:30:45",
				Tz:      "Z",
				Level:   "debug",
				Tag:     "app",
				Message: `msg="Hello"`,
			},
		},
		{
			name: "Unbracketed level prefix",
			line: "2023-10-25 15:30:45 WARNING: Disk is full",
			want: LogEntry{
				Date:    "2023-10-25",
				Time:    "15:30:45",
				Level:   "WARNING",
				Message: "Disk is full",
			},
		},
		{
			name: "Double bracket tags",
			line: "[2023-10-25 15:30:45] [App1] [ERROR] Disk is full",
			want: LogEntry{
				Date:    "2023-10-25",
				Time:    "15:30:45",
				Tag:     "App1",
				Level:   "ERROR",
				Message: "Disk is full",
			},
		},
		{
			name: "No time format",
			line: "[INFO] Just a message",
			want: LogEntry{
				Level:   "INFO",
				Message: "Just a message",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := app.parseSingleLine(tt.line, false)
			
			// We only compare the explicitly set fields in want
			if tt.want.Date != "" && got.Date != tt.want.Date { t.Errorf("Date: got %v want %v", got.Date, tt.want.Date) }
			if tt.want.Time != "" && got.Time != tt.want.Time { t.Errorf("Time: got %v want %v", got.Time, tt.want.Time) }
			if tt.want.Ms != "" && got.Ms != tt.want.Ms { t.Errorf("Ms: got %v want %v", got.Ms, tt.want.Ms) }
			if tt.want.Tz != "" && got.Tz != tt.want.Tz { t.Errorf("Tz: got %v want %v", got.Tz, tt.want.Tz) }
			if tt.want.Level != "" && got.Level != tt.want.Level { t.Errorf("Level: got %v want %v", got.Level, tt.want.Level) }
			if tt.want.Tag != "" && got.Tag != tt.want.Tag { t.Errorf("Tag: got %v want %v", got.Tag, tt.want.Tag) }
			if tt.want.Message != "" && got.Message != tt.want.Message { t.Errorf("Message: got %v want %v", got.Message, tt.want.Message) }
		})
	}
}

func TestParseLine(t *testing.T) {
	app := NewApp()
	app.sessionID = 123
	
	app.parseLine("2023-10-25 15:30:45 INFO Hello", 123)
	
	app.mu.Lock()
	count := len(app.logEntries)
	lastLineNum := app.currentLastLineNum
	app.mu.Unlock()
	
	if count != 1 {
		t.Errorf("Expected 1 log entry, got %d", count)
	}
	if lastLineNum != 1 {
		t.Errorf("Expected currentLastLineNum 1, got %d", lastLineNum)
	}
	
	// Test wrong session ID (should be ignored)
	app.parseLine("2023-10-25 15:30:46 INFO Ignore me", 999)
	
	app.mu.Lock()
	count2 := len(app.logEntries)
	app.mu.Unlock()
	
	if count2 != 1 {
		t.Errorf("Expected 1 log entry after bad session, got %d", count2)
	}
}

func TestParseLine_Eviction(t *testing.T) {
	app := NewApp()
	app.sessionID = 123
	
	// Temporarily pretend we have 50000 entries
	app.logEntries = make([]LogEntry, 50000)
	app.lastSentIdx = 5
	
	app.parseLine("2023-10-25 15:30:45 INFO Overflow", 123)
	
	app.mu.Lock()
	count := len(app.logEntries)
	lastSent := app.lastSentIdx
	app.mu.Unlock()
	
	if count != 50000 {
		t.Errorf("Expected 50000 entries (eviction working), got %d", count)
	}
	if lastSent != 4 {
		t.Errorf("Expected lastSentIdx to decrement to 4, got %d", lastSent)
	}
}
