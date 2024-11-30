// Package smtp implements a simple SMTP server for capturing and storing emails.
package smtp

import (
	"crypto/tls"
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
	storage    *storage.EmailStorage
	from       string
	recipients []string
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

	// Extract domain and user from sender
	senderDomain, senderUser := parseEmailAddress(s.from)

	// Store email in sender's OUT directory
	subject := fmt.Sprintf("to-%s", s.recipients[0]) // Use first recipient for subject
	if err := s.storage.StoreEmail(storage.Outgoing, senderDomain, senderUser, subject, content); err != nil {
		log.Printf("Error storing outgoing email for sender %s: %v", s.from, err)
	}

	// Store email for each recipient in their IN directory
	for _, recipient := range s.recipients {
		domain, user := parseEmailAddress(recipient)
		subject := fmt.Sprintf("from-%s", s.from)

		if err := s.storage.StoreEmail(storage.Incoming, domain, user, subject, content); err != nil {
			log.Printf("Error storing email for recipient %s: %v", recipient, err)
		}
	}

	return nil
}

// Reset resets the session state as required by go-smtp.Session interface.
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
	port      int
	storage   *storage.EmailStorage
	server    *smtp.Server
	tlsConfig *tls.Config
}

// NewServer creates a new SMTP server instance.
func NewServer(port int, emailStorage *storage.EmailStorage) *Server {
	return &Server{
		port:    port,
		storage: emailStorage,
	}
}

// SetTLSConfig configures TLS for the server using certificate files
func (server *Server) SetTLSConfig(certFile, keyFile string) error {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("loading TLS certificates: %w", err)
	}

	server.tlsConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:  tls.VersionTLS12,
	}

	return nil
}

// Start initializes the SMTP server and begins listening for connections.
func (server *Server) Start() error {
	backend := &Backend{storage: server.storage}

	server.server = smtp.NewServer(backend)
	server.server.Addr = fmt.Sprintf(":%d", server.port)
	server.server.ReadTimeout = 10 * time.Second
	server.server.WriteTimeout = 10 * time.Second
	server.server.MaxMessageBytes = 1024 * 1024 // 1MB
	server.server.MaxRecipients = 50
	
	// Configure TLS if certificates are provided
	if server.tlsConfig != nil {
		server.server.TLSConfig = server.tlsConfig
		server.server.AllowInsecureAuth = false // Require secure authentication when TLS is enabled
		log.Printf("TLS enabled")
	} else {
		server.server.AllowInsecureAuth = true
		log.Printf("Warning: Running without TLS encryption")
	}

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
	for i := 0; i < len(email); i++ {
		if email[i] == '@' {
			return email[i+1:], email[:i]
		}
	}
	return "unknown", email
}
