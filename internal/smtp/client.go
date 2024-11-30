// Package smtp implements SMTP client and server functionality.
package smtp

import (
	"bytes"
	"fmt"
	"net/smtp"

	"github.com/nathabonfim59/gargantua-sink/internal/storage"
)

// Client represents an SMTP client that can send emails.
type Client struct {
	storage    *storage.EmailStorage
	forwardTo  string // Optional SMTP server to forward emails to
	forwardAuth smtp.Auth
}

// ClientConfig holds configuration for the SMTP client.
type ClientConfig struct {
	ForwardTo     string // SMTP server to forward emails to (optional)
	ForwardUser   string // Username for forwarding server (optional)
	ForwardPass   string // Password for forwarding server (optional)
	ForwardHost   string // Hostname for forwarding server (optional)
}

// NewClient creates a new SMTP client instance.
func NewClient(storage *storage.EmailStorage, config *ClientConfig) *Client {
	client := &Client{
		storage: storage,
	}

	if config != nil && config.ForwardTo != "" {
		client.forwardTo = config.ForwardTo
		if config.ForwardUser != "" && config.ForwardPass != "" {
			client.forwardAuth = smtp.PlainAuth("", config.ForwardUser, config.ForwardPass, config.ForwardHost)
		}
	}

	return client
}

// SendMail sends an email through the client.
// If forwarding is configured, it will attempt to send through the forwarding server.
// In all cases, it stores the email as an outgoing message.
func (c *Client) SendMail(from string, to []string, data []byte) error {
	// Store as outgoing email for each recipient
	for _, recipient := range to {
		domain, user := parseEmailAddress(recipient)
		if err := c.storage.StoreEmail(storage.Outgoing, domain, user, fmt.Sprintf("from-%s", from), data); err != nil {
			return fmt.Errorf("storing outgoing email: %w", err)
		}
	}

	// If forwarding is configured, attempt to send through the forwarding server
	if c.forwardTo != "" {
		if err := smtp.SendMail(c.forwardTo, c.forwardAuth, from, to, data); err != nil {
			return fmt.Errorf("forwarding email: %w", err)
		}
	}

	return nil
}

// SendMailWithAttachments sends an email with attachments.
func (c *Client) SendMailWithAttachments(from string, to []string, subject, body string, attachments map[string][]byte) error {
	// Create email content with attachments
	email, err := createTestEmail(from, to[0], subject, body, attachments)
	if err != nil {
		return fmt.Errorf("creating email with attachments: %w", err)
	}

	return c.SendMail(from, to, email)
}

// createEmail creates a properly formatted email message.
func createEmail(from string, to []string, subject, body string) *bytes.Buffer {
	buf := bytes.NewBuffer(nil)
	buf.WriteString(fmt.Sprintf("From: %s\r\n", from))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", to[0])) // Using first recipient for header
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	buf.WriteString("\r\n")
	buf.WriteString(body)
	return buf
}
