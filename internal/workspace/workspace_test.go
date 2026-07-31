package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetDefaultDir(t *testing.T) {
	dir := getDefaultDir()
	if dir == "" {
		t.Error("getDefaultDir() should return a valid fallback path (like C:\\)")
	}
}

func TestListFiles(t *testing.T) {
	// Create a temp directory
	tmpDir := t.TempDir()
	
	// Create some files
	os.WriteFile(filepath.Join(tmpDir, "test1.log"), []byte("log"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "test2.log"), []byte("log"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "ignore.txt"), []byte("txt"), 0644)
	
	// Create a subdirectory with a .log file (should not be listed by ListFiles which is non-recursive)
	subDir := filepath.Join(tmpDir, "subdir")
	os.Mkdir(subDir, 0755)
	os.WriteFile(filepath.Join(subDir, "test3.log"), []byte("log"), 0644)
	
	files := ListFiles(tmpDir)
	
	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}
	
	// Test listing non-existent directory
	files = ListFiles("C:\\does_not_exist_12345")
	if len(files) != 0 {
		t.Error("Expected empty slice for non-existent directory")
	}
}


func TestListFilesNames(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test1.log"), []byte("log"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "test2.log"), []byte("log"), 0644)
	
	files := ListFiles(tmpDir)
	
	// Verify names
	names := make(map[string]bool)
	for _, f := range files {
		names[f.Name] = true
	}
	
	if !names["test1.log"] || !names["test2.log"] {
		t.Errorf("Expected test1.log and test2.log, got %v", files)
	}
	
	// Test empty path
	if len(ListFiles("")) != 0 {
		t.Error("ListFiles with empty path should return empty slice")
	}
}

func TestProcessDrop(t *testing.T) {
	tmpDir := t.TempDir()
	
	logPath := filepath.Join(tmpDir, "app.log")
	os.WriteFile(logPath, []byte("hello"), 0644)
	
	// Case 1: Drop a directory
	res := ProcessDrop(tmpDir)
	if res.Error != "" {
		t.Errorf("ProcessDrop(dir) returned error: %s", res.Error)
	}
	if res.Path != tmpDir {
		t.Errorf("Expected path %s, got %s", tmpDir, res.Path)
	}
	if len(res.Files) != 1 || res.Files[0].Name != "app.log" {
		t.Errorf("Expected 1 file (app.log), got %v", res.Files)
	}
	
	// Case 2: Drop a specific file (should resolve to its parent directory)
	res2 := ProcessDrop(logPath)
	if res2.Error != "" {
		t.Errorf("ProcessDrop(file) returned error: %s", res2.Error)
	}
	if res2.Path != tmpDir {
		t.Errorf("Expected resolved path to be %s, got %s", tmpDir, res2.Path)
	}
	
	// Case 3: Drop invalid path
	res3 := ProcessDrop(filepath.Join(tmpDir, "does-not-exist"))
	if res3.Error == "" {
		t.Error("Expected error for non-existent path")
	}
	if res3.Path != filepath.Join(tmpDir, "does-not-exist") {
		t.Errorf("Expected path %s, got %s", filepath.Join(tmpDir, "does-not-exist"), res3.Path)
	}
}



func TestGetFirstValidPath(t *testing.T) {
	tmpDir := t.TempDir()
	paths := []string{"does_not_exist", tmpDir, "does_not_exist_2"}
	res := getFirstValidPath(paths)
	if res != tmpDir {
		t.Errorf("Expected %s, got %s", tmpDir, res)
	}

	res = getFirstValidPath([]string{})
	if res != "" {
		t.Errorf("Expected empty string, got %s", res)
	}
}
