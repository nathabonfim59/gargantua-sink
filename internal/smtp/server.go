// Package smtp implements a simple SMTP server for capturing and storing emails.
package smtp

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
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

// Start initializes and starts the SMTP server
func (server *Server) Start() error {
	// If port is 0, find a random available port
	if server.port == 0 {
		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			return fmt.Errorf("finding available port: %w", err)
		}
		server.port = listener.Addr().(*net.TCPAddr).Port
		listener.Close()
	}

	backend := &Backend{
		storage: server.storage,
		domains: server.domains,
	}

	server.server = smtp.NewServer(backend)
	server.server.Addr = fmt.Sprintf(":%d", server.port)
	server.server.Domain = "localhost"
	server.server.ReadTimeout = 10 * time.Second
	server.server.WriteTimeout = 10 * time.Second
	server.server.MaxMessageBytes = 1024 * 1024 // 1MB
	server.server.MaxRecipients = 50
	server.server.AllowInsecureAuth = true

	// Configure TLS if any domains are configured with certificates
	tlsConfigs := make(map[string]*tls.Config)
	for domain, config := range server.domains {
		if config.TLSConfig != nil {
			tlsConfigs[domain] = config.TLSConfig
		}
	}

	if len(tlsConfigs) > 0 {
		server.server.TLSConfig = &tls.Config{
			GetConfigForClient: func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
				if config, ok := tlsConfigs[hello.ServerName]; ok {
					return config, nil
				}
				// Return default config if no matching domain found
				return &tls.Config{
					Certificates: []tls.Certificate{},
				}, nil
			},
		}
	}

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
