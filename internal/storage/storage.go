// Package storage provides email storage functionality for the Gargantua Sink SMTP server.
package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// EmailStorage handles the persistence of email messages to the filesystem.
type EmailStorage struct {
	rootPath string
}

// NewEmailStorage creates a new storage instance with the specified root directory.
// It ensures the storage directory exists and is accessible.
func NewEmailStorage(rootPath string) (*EmailStorage, error) {
	if err := os.MkdirAll(rootPath, 0755); err != nil {
		return nil, fmt.Errorf("creating storage directory: %w", err)
	}

	return &EmailStorage{
		rootPath: rootPath,
	}, nil
}

// StoreEmail saves an email message to the filesystem using the specified metadata.
// The email is stored in the following structure:
// rootPath/domain/user/YYYYMMDDHHMMSS-subject.eml
func (storage *EmailStorage) StoreEmail(domain, user, subject string, content []byte) error {
	userPath := filepath.Join(storage.rootPath, domain, user)
	if err := os.MkdirAll(userPath, 0755); err != nil {
		return fmt.Errorf("creating user directory: %w", err)
	}

	timestamp := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("%s-%s.eml", timestamp, subject)
	emailPath := filepath.Join(userPath, filename)

	if err := os.WriteFile(emailPath, content, 0644); err != nil {
		return fmt.Errorf("writing email file: %w", err)
	}

	return nil
}
