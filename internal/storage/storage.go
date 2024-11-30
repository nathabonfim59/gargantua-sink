// Package storage provides email storage functionality for the Gargantua Sink SMTP server.
package storage

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

// Direction represents the flow of an email (incoming or outgoing)
type Direction int

const (
	// Incoming represents emails received by the server
	Incoming Direction = iota
	// Outgoing represents emails sent through the server
	Outgoing
)

func (d Direction) String() string {
	switch d {
	case Incoming:
		return "IN"
	case Outgoing:
		return "OUT"
	default:
		return "UNKNOWN"
	}
}

// EmailStorage handles the persistence of email messages to the filesystem.
type EmailStorage struct {
	rootPath string
	mu       sync.Mutex
}

var (
	// safeFilename replaces unsafe characters with underscores
	safeFilename = regexp.MustCompile(`[^a-zA-Z0-9-.]`)
)

// generateUniqueID generates a random 8-character hex string
func generateUniqueID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
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
// rootPath/domain/user/YYYYMMDDHHMMSS-[unique-id]-[IN|OUT]-subject.eml
func (storage *EmailStorage) StoreEmail(direction Direction, domain, user, subject string, content []byte) error {
	storage.mu.Lock()
	defer storage.mu.Unlock()

	// Create safe filename from subject
	safeSubject := safeFilename.ReplaceAllString(subject, "_")
	timestamp := time.Now().Format("20060102150405")
	uniqueID := generateUniqueID()
	filename := fmt.Sprintf("%s-%s-%s-%s.eml", timestamp, uniqueID, direction, safeSubject)

	// Create user directory
	userPath := filepath.Join(storage.rootPath, domain, user)
	if err := os.MkdirAll(userPath, 0755); err != nil {
		return fmt.Errorf("creating user directory: %w", err)
	}

	// Write email file
	emailPath := filepath.Join(userPath, filename)
	if err := os.WriteFile(emailPath, content, 0644); err != nil {
		return fmt.Errorf("writing email file: %w", err)
	}

	return nil
}
