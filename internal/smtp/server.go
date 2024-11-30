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
	domains map[string]DomainConfig
}

// NewSession creates a new SMTP session.
func (bkd *Backend) NewSession(_ *smtp.Conn) (smtp.Session, error) {
	return &Session{
		storage: bkd.storage,
		domains: bkd.domains,
	}, nil
}

// Session represents an SMTP session.
type Session struct {
	storage    *storage.EmailStorage
	domains    map[string]DomainConfig
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

// DomainConfig represents the configuration for a specific domain
type DomainConfig struct {
	Domain     string
	TLSConfig  *tls.Config
	Storage    *storage.EmailStorage
	StorageDir string
}

// Server represents an SMTP server instance.
type Server struct {
	port    int
	domains map[string]DomainConfig
	storage *storage.EmailStorage
	server  *smtp.Server
}

// NewServer creates a new SMTP server instance.
func NewServer(port int, defaultStorage *storage.EmailStorage) *Server {
	return &Server{
		port:    port,
		storage: defaultStorage,
		domains: make(map[string]DomainConfig),
	}
}

// AddDomain adds a new domain configuration to the server
func (s *Server) AddDomain(domain, certFile, keyFile, storageDir string) error {
	if domain == "" {
		return fmt.Errorf("domain name cannot be empty")
	}

	// Validate TLS configuration
	if (certFile != "" && keyFile == "") || (certFile == "" && keyFile != "") {
		return fmt.Errorf("both certificate and key files must be provided for TLS")
	}

	// Create storage for the domain
	storage, err := storage.NewEmailStorage(storageDir)
	if err != nil {
		return fmt.Errorf("creating storage for domain %s: %w", domain, err)
	}

	// Create domain configuration
	config := &DomainConfig{
		Domain:  domain,
		Storage: storage,
	}

	// Configure TLS if certificate files are provided
	if certFile != "" && keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return fmt.Errorf("loading TLS certificate for domain %s: %w", domain, err)
		}
		config.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			ServerName:   domain,
		}
	}

	s.domains[domain] = *config
	return nil
}

// Start initializes the SMTP server and begins listening for connections.
func (server *Server) Start() error {
	backend := &Backend{
		storage: server.storage,
		domains: server.domains,
	}

	server.server = smtp.NewServer(backend)
	server.server.Addr = fmt.Sprintf(":%d", server.port)
	server.server.ReadTimeout = 10 * time.Second
	server.server.WriteTimeout = 10 * time.Second
	server.server.MaxMessageBytes = 1024 * 1024 // 1MB
	server.server.MaxRecipients = 50

	// Configure TLS if any domains are configured
	if len(server.domains) > 0 {
		// Create a TLS config that selects the appropriate certificate based on domain
		tlsConfig := &tls.Config{
			GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
				if config, ok := server.domains[hello.ServerName]; ok {
					return &config.TLSConfig.Certificates[0], nil
				}
				// Return the first certificate as default if domain not found
				for _, config := range server.domains {
					return &config.TLSConfig.Certificates[0], nil
				}
				return nil, fmt.Errorf("no certificate found for domain: %s", hello.ServerName)
			},
			MinVersion: tls.VersionTLS12,
		}
		server.server.TLSConfig = tlsConfig
		server.server.AllowInsecureAuth = false
		log.Printf("TLS enabled for %d domains", len(server.domains))
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
