// Package smtp implements SMTP client and server functionality.
package smtp

import (
	"bytes"
	"fmt"
	"net/smtp"
	"strings"

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
func (c *Client) SendMail(from string, to []string, subject string, body []byte) error {
	// Parse sender's email address
	fromDomain, fromUser := parseEmailAddress(from)

	// Store outgoing email
	err := c.storage.StoreEmail(
		storage.Outgoing,
		fromDomain,
		fromUser,
		subject,
		body,
	)
	if err != nil {
		return fmt.Errorf("failed to store outgoing email: %w", err)
	}

	// If forwarding is enabled, send the email
	if c.forwardTo != "" {
		err = smtp.SendMail(
			c.forwardTo,
			c.forwardAuth,
			from,
			to,
			body,
		)
		if err != nil {
			return fmt.Errorf("failed to forward email: %w", err)
		}
	}

	return nil
}

// SendMailWithAttachments sends an email with attachments.
func (c *Client) SendMailWithAttachments(from string, to []string, subject, body string, attachments map[string][]byte) error {
	// Create email content with attachments
	email := createEmail(from, to, subject, body)
	for filename, attachment := range attachments {
		email.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n\r\n", filename))
		email.Write(attachment)
		email.WriteString("\r\n")
	}

	return c.SendMail(from, to, subject, email.Bytes())
}

// createEmail creates a properly formatted email message.
func createEmail(from string, to []string, subject, body string) *bytes.Buffer {
	email := bytes.NewBuffer(nil)

	// Add headers
	email.WriteString(fmt.Sprintf("From: %s\r\n", from))
	email.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(to, ", ")))
	email.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	email.WriteString("MIME-Version: 1.0\r\n")
	email.WriteString("Content-Type: multipart/mixed; boundary=boundary123\r\n")
	email.WriteString("\r\n")

	// Add body
	email.WriteString("--boundary123\r\n")
	email.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	email.WriteString("\r\n")
	email.WriteString(body)
	email.WriteString("\r\n")

	return email
}
