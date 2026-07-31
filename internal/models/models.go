package models

// FileInfo represents a single log file in a tracked folder.
type FileInfo struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// DropResult is the response sent to the frontend after a folder is dropped or polled.
type DropResult struct {
	Path  string     `json:"path"`
	Name  string     `json:"name"`
	Files []FileInfo `json:"files"`
	Error string     `json:"error"`
}

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
