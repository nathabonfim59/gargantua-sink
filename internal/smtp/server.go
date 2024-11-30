// Package smtp implements a simple SMTP server for capturing and storing emails.
package smtp

import (
	"fmt"
	"io"
	"log"
	"time"

	"github.com/emersion/go-smtp"
	"github.com/nathabonfim59/gargantua-sink/internal/storage"
)

// Backend implements SMTP server handler.
type Backend struct {
	storage *storage.EmailStorage
}

// NewSession creates a new SMTP session.
func (bkd *Backend) NewSession(_ *smtp.Conn) (smtp.Session, error) {
	return &Session{
		storage: bkd.storage,
	}, nil
}

// Session represents an SMTP session.
type Session struct {
	storage     *storage.EmailStorage
	from        string
	recipients  []string
}

// AuthPlain implements authentication - always returns nil as we accept all auth.
func (s *Session) AuthPlain(username, password string) error {
	return nil
}

// Mail sets the sender address.
func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	s.from = from
	return nil
}

// Rcpt adds a recipient address.
func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	s.recipients = append(s.recipients, to)
	return nil
}

// Data handles the email content.
func (s *Session) Data(r io.Reader) error {
	content, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("reading email content: %w", err)
	}

	// Store email for each recipient
	for _, recipient := range s.recipients {
		domain, user := parseEmailAddress(recipient)
		subject := fmt.Sprintf("from-%s", s.from) // Simple subject based on sender

		if err := s.storage.StoreEmail(domain, user, subject, content); err != nil {
			log.Printf("Error storing email for recipient %s: %v", recipient, err)
		}
	}

	return nil
}

// Reset resets the session state.
func (s *Session) Reset() {
	s.from = ""
	s.recipients = nil
}

// Logout closes the session.
func (s *Session) Logout() error {
	return nil
}

// Server represents an SMTP server instance.
type Server struct {
	port    int
	storage *storage.EmailStorage
	server  *smtp.Server
}

// NewServer creates a new SMTP server instance.
func NewServer(port int, emailStorage *storage.EmailStorage) *Server {
	return &Server{
		port:    port,
		storage: emailStorage,
	}
}

// Start initializes the SMTP server and begins listening for connections.
func (server *Server) Start() error {
	backend := &Backend{storage: server.storage}
	
	server.server = smtp.NewServer(backend)
	server.server.Addr = fmt.Sprintf(":%d", server.port)
	server.server.Domain = "localhost"
	server.server.ReadTimeout = 30 * time.Second
	server.server.WriteTimeout = 30 * time.Second
	server.server.MaxMessageBytes = 1024 * 1024 // 1MB
	server.server.MaxRecipients = 50
	server.server.AllowInsecureAuth = true

	log.Printf("Starting SMTP server on :%d", server.port)
	return server.server.ListenAndServe()
}

// Stop gracefully shuts down the SMTP server.
func (server *Server) Stop() error {
	if server.server != nil {
		return server.server.Close()
	}
	return nil
}

// parseEmailAddress extracts domain and user from email address.
func parseEmailAddress(email string) (domain, user string) {
	// Simple parsing, can be enhanced for edge cases
	for i := 0; i < len(email); i++ {
		if email[i] == '@' {
			return email[i+1:], email[:i]
		}
	}
	return "unknown", email
}
