package storage

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestNewEmailStorage(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid_path",
			path:    tempDir,
			wantErr: false,
		},
		{
			name:    "nested_path",
			path:    filepath.Join(tempDir, "nested", "storage"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, err := NewEmailStorage(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEmailStorage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if storage == nil {
				t.Error("NewEmailStorage() returned nil storage")
			}
		})
	}
}

func TestStoreEmail(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewEmailStorage(tempDir)
	if err != nil {
		t.Fatalf("NewEmailStorage() error = %v", err)
	}

	tests := []struct {
		name     string
		domain   string
		user     string
		subject  string
		content  []byte
		wantPath string
		wantErr  bool
	}{
		{
			name:     "simple_email",
			domain:   "example.com",
			user:     "john",
			subject:  "test-subject",
			content:  []byte("Hello, World!"),
			wantPath: filepath.Join("example.com", "john"),
			wantErr:  false,
		},
		{
			name:     "special_chars_in_subject",
			domain:   "test.org",
			user:     "jane",
			subject:  "test/subject:with*special?chars",
			content:  []byte("Special characters test"),
			wantPath: filepath.Join("test.org", "jane"),
			wantErr:  false,
		},
		{
			name:     "large_email",
			domain:   "large.com",
			user:     "user",
			subject:  "large-email",
			content:  bytes.Repeat([]byte("Large content "), 1000),
			wantPath: filepath.Join("large.com", "user"),
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := storage.StoreEmail(tt.domain, tt.user, tt.subject, tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("StoreEmail() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check if directory was created
			fullPath := filepath.Join(tempDir, tt.wantPath)
			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				t.Errorf("Expected directory %s was not created", fullPath)
			}

			// Check if email file exists and contains correct content
			files, err := os.ReadDir(fullPath)
			if err != nil {
				t.Fatalf("Reading directory failed: %v", err)
			}
			if len(files) != 1 {
				t.Errorf("Expected 1 file, got %d", len(files))
				return
			}

			content, err := os.ReadFile(filepath.Join(fullPath, files[0].Name()))
			if err != nil {
				t.Fatalf("Reading email file failed: %v", err)
			}
			if !bytes.Contains(content, tt.content) {
				t.Error("Stored email does not contain expected content")
			}
		})
	}
}

func TestConcurrentStorage(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewEmailStorage(tempDir)
	if err != nil {
		t.Fatalf("NewEmailStorage() error = %v", err)
	}

	const numConcurrent = 100
	var wg sync.WaitGroup
	errCh := make(chan error, numConcurrent)

	// Start concurrent writes
	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(num int) {
			defer wg.Done()
			
			domain := "concurrent.com"
			user := "user"
			subject := fmt.Sprintf("concurrent-test-%d", num)
			content := []byte(fmt.Sprintf("Content %d: %s", num, time.Now()))

			if err := storage.StoreEmail(domain, user, subject, content); err != nil {
				errCh <- err
			}
		}(i)
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(errCh)

	// Check for any errors
	for err := range errCh {
		if err != nil {
			t.Errorf("Concurrent storage error: %v", err)
		}
	}

	// Verify files
	files, err := os.ReadDir(filepath.Join(tempDir, "concurrent.com", "user"))
	if err != nil {
		t.Fatalf("Reading directory failed: %v", err)
	}

	if len(files) != numConcurrent {
		t.Errorf("Expected %d files, got %d", numConcurrent, len(files))
	}

	// Verify each file has unique content
	seen := make(map[string]bool)
	for _, file := range files {
		content, err := os.ReadFile(filepath.Join(tempDir, "concurrent.com", "user", file.Name()))
		if err != nil {
			t.Errorf("Reading file %s failed: %v", file.Name(), err)
			continue
		}
		
		if seen[string(content)] {
			t.Errorf("Duplicate content found in file %s", file.Name())
		}
		seen[string(content)] = true
	}
}
