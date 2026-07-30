package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

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

func getFirstValidPath(paths []string) string {
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func getDefaultDir() string {
	paths := []string{
		`C:\Program Files (x86)\hatacone\logs`,
		`C:\`,
	}
	
	for _, drive := range "DEFGHIJKLMNOPQRSTUVWXYZ" {
		paths = append(paths, string(drive)+`:\`)
	}
	
	return getFirstValidPath(paths)
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

// ProcessDrop validates a dropped/polled folder path and returns its contents.
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
