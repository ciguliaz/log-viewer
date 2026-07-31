package workspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"log-viewer/internal/models"
)

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
		`C:\Program Files (x86)\logs`,
		`C:\`,
	}
	
	for _, drive := range "DEFGHIJKLMNOPQRSTUVWXYZ" {
		paths = append(paths, string(drive)+`:\`)
	}
	
	return getFirstValidPath(paths)
}

// SelectFolder opens a dialog to select a directory
func SelectFolder(ctx context.Context) string {
	opts := runtime.OpenDialogOptions{
		DefaultDirectory: getDefaultDir(),
		Title:            "Select Log Folder",
	}
	dir, err := runtime.OpenDirectoryDialog(ctx, opts)
	if err != nil {
		fmt.Println("Error selecting directory:", err)
		return ""
	}
	
	// Reset CWD to prevent Windows from locking the selected directory
	os.Chdir(`C:\`)
	
	return dir
}

// ListFiles lists .log files in a given directory
func ListFiles(dirPath string) []models.FileInfo {
	var files []models.FileInfo
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
			files = append(files, models.FileInfo{
				Name: entry.Name(),
				Path: filepath.Join(dirPath, entry.Name()),
			})
		}
	}
	return files
}

// ProcessDrop validates a dropped/polled folder path and returns its contents.
func ProcessDrop(path string) *models.DropResult {
	info, err := os.Stat(path)
	if err != nil {
		name := filepath.Base(path)
		if name == "" || name == "." {
			name = path
		}
		return &models.DropResult{
			Path:  path,
			Name:  name,
			Files: []models.FileInfo{},
			Error: "Folder inaccessible or deleted",
		}
	}

	dir := path
	if !info.IsDir() {
		dir = filepath.Dir(path)
	}

	files := ListFiles(dir)
	name := filepath.Base(dir)
	if name == "" || name == "." {
		name = dir
	}

	return &models.DropResult{
		Path:  dir,
		Name:  name,
		Files: files,
		Error: "",
	}
}
