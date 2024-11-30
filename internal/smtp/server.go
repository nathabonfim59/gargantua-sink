// Package smtp implements a simple SMTP server for capturing and storing emails.
package smtp

import (
	"fmt"
	"net"

	"github.com/nathabonfim59/gargantua-sink/internal/storage"
)

// Server represents an SMTP server instance.
type Server struct {
	port     int
	storage  *storage.EmailStorage
	listener net.Listener
}

// NewServer creates a new SMTP server instance with the specified configuration.
func NewServer(port int, emailStorage *storage.EmailStorage) *Server {
	return &Server{
		port:    port,
		storage: emailStorage,
	}
}

// Start initializes the SMTP server and begins listening for connections.
func (server *Server) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", server.port))
	if err != nil {
		return fmt.Errorf("starting SMTP server: %w", err)
	}
	server.listener = listener

	// TODO: Implement SMTP protocol handling
	return nil
}

// Stop gracefully shuts down the SMTP server.
func (server *Server) Stop() error {
	if server.listener != nil {
		return server.listener.Close()
	}
	return nil
}
