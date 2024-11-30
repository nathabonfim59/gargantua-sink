package storage

import (
	"bytes"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestNewEmailStorage(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid_path",
			path:    t.TempDir(),
			wantErr: false,
		},
		{
			name:    "nested_path",
			path:    filepath.Join(t.TempDir(), "nested", "path"),
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
	tests := []struct {
		name      string
		domain    string
		user      string
		subject   string
		content   []byte
		direction Direction
		wantErr   bool
	}{
		{
			name:      "simple_email",
			domain:    "example.com",
			user:      "john",
			subject:   "test-subject",
			content:   []byte("test content"),
			direction: Incoming,
			wantErr:   false,
		},
		{
			name:      "outgoing_email",
			domain:    "example.com",
			user:      "john",
			subject:   "test-subject",
			content:   []byte("test content"),
			direction: Outgoing,
			wantErr:   false,
		},
		{
			name:      "special_chars_in_subject",
			domain:    "example.com",
			user:      "john",
			subject:   "test/subject*with?special:chars",
			content:   []byte("test content"),
			direction: Incoming,
			wantErr:   false,
		},
		{
			name:      "large_email",
			domain:    "example.com",
			user:      "john",
			subject:   "large-email",
			content:   bytes.Repeat([]byte("a"), 1024*1024), // 1MB
			direction: Incoming,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new storage instance for each test case
			tempDir := t.TempDir()
			storage, err := NewEmailStorage(tempDir)
			if err != nil {
				t.Fatalf("Failed to create storage: %v", err)
			}

			err = storage.StoreEmail(tt.direction, tt.domain, tt.user, tt.subject, tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("StoreEmail() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Verify directory structure and file
			dirPath := filepath.Join(tempDir, tt.domain, tt.user, tt.direction.String())
			files, err := os.ReadDir(dirPath)
			if err != nil {
				t.Fatalf("Failed to read directory: %v", err)
			}

			if len(files) != 1 {
				t.Errorf("Expected 1 file in %s, got %d", dirPath, len(files))
				return
			}

			// Verify content
			content, err := os.ReadFile(filepath.Join(dirPath, files[0].Name()))
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}

			if !bytes.Equal(content, tt.content) {
				t.Error("Stored content does not match input")
			}
		})
	}
}

func TestConcurrentStorage(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewEmailStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	const (
		numGoroutines    = 10
		emailsPerRoutine = 100
	)

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < emailsPerRoutine; j++ {
				direction := Incoming
				if j%2 == 0 {
					direction = Outgoing
				}
				err := storage.StoreEmail(
					direction,
					"example.com",
					"user",
					"test-subject",
					[]byte("test content"),
				)
				if err != nil {
					t.Errorf("Failed to store email: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify total number of files in each direction
	expectedPerDirection := numGoroutines * emailsPerRoutine / 2

	// Check incoming directory
	inFiles, err := os.ReadDir(filepath.Join(tempDir, "example.com", "user", "IN"))
	if err != nil {
		t.Fatalf("Failed to read IN directory: %v", err)
	}
	if len(inFiles) != expectedPerDirection {
		t.Errorf("Expected %d files in IN directory, got %d", expectedPerDirection, len(inFiles))
	}

	// Check outgoing directory
	outFiles, err := os.ReadDir(filepath.Join(tempDir, "example.com", "user", "OUT"))
	if err != nil {
		t.Fatalf("Failed to read OUT directory: %v", err)
	}
	if len(outFiles) != expectedPerDirection {
		t.Errorf("Expected %d files in OUT directory, got %d", expectedPerDirection, len(outFiles))
	}
}
