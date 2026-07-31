package main

import (
	"context"
	"os"

	"log-viewer/internal/models"
	"log-viewer/internal/tailer"
	"log-viewer/internal/workspace"
)

// App struct holds all application state and serves as the Wails API Controller.
type App struct {
	ctx    context.Context
	tailer *tailer.Tailer
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.tailer = tailer.NewTailer(ctx)

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

// ---------------------------------------------------------
// Wails Frontend API Endpoints
// ---------------------------------------------------------

// SelectFolder opens a dialog to select a directory
func (a *App) SelectFolder() string {
	return workspace.SelectFolder(a.ctx)
}

// ListFiles lists .log files in a given directory
func (a *App) ListFiles(dirPath string) []models.FileInfo {
	return workspace.ListFiles(dirPath)
}

// ProcessDrop validates a dropped/polled folder path and returns its contents.
func (a *App) ProcessDrop(path string) *models.DropResult {
	return workspace.ProcessDrop(path)
}

// StartTailing stops any existing tail and starts tailing a new file
func (a *App) StartTailing(filePath string) {
	if a.tailer != nil {
		a.tailer.StartTailing(filePath)
	}
}

// StopTailing explicitly stops tailing the active file, releasing its lock
func (a *App) StopTailing() {
	if a.tailer != nil {
		a.tailer.StopTailing()
	}
}

// LoadPreviousChunk reads a 1MB chunk of logs backwards from the current fileOffset
func (a *App) LoadPreviousChunk() []models.LogEntry {
	if a.tailer != nil {
		return a.tailer.LoadPreviousChunk()
	}
	return nil
}

// GetInitialLogs allows frontend to fetch immediately on load
func (a *App) GetInitialLogs() []models.LogEntry {
	if a.tailer != nil {
		return a.tailer.GetInitialLogs()
	}
	return []models.LogEntry{}
}
