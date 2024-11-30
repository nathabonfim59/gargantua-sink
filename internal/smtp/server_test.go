// Package smtp implements SMTP server tests
package smtp

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net"
	"net/textproto"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/emersion/go-smtp"
	"github.com/nathabonfim59/gargantua-sink/internal/storage"
)

func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func setupTestServer(t *testing.T) (*Server, *storage.EmailStorage, string, int, error) {
	port, err := getFreePort()
	if err != nil {
		return nil, nil, "", 0, fmt.Errorf("getting free port: %w", err)
	}

	tempDir := t.TempDir()
	emailStorage, err := storage.NewEmailStorage(tempDir)
	if err != nil {
		return nil, nil, "", 0, fmt.Errorf("creating email storage: %w", err)
	}

	server := NewServer(port, emailStorage)
	serverErrCh := make(chan error, 1)
	go func() {
		if err := server.Start(); err != nil {
			serverErrCh <- err
		}
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Check if server started successfully
	select {
	case err := <-serverErrCh:
		return nil, nil, "", 0, fmt.Errorf("server failed to start: %w", err)
	default:
		// Server started successfully
	}

	return server, emailStorage, tempDir, port, nil
}

func createTestEmail(from, to, subject, body string, attachments map[string][]byte) ([]byte, error) {
	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)

	header := make(textproto.MIMEHeader)
	header.Set("From", from)
	header.Set("To", to)
	header.Set("Subject", subject)
	header.Set("Content-Type", "multipart/mixed; boundary="+writer.Boundary())

	// Write body
	part, err := writer.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/plain"}})
	if err != nil {
		return nil, err
	}
	if _, err := part.Write([]byte(body)); err != nil {
		return nil, err
	}

	// Write attachments
	for filename, content := range attachments {
		part, err := writer.CreateFormFile("attachment", filename)
		if err != nil {
			return nil, err
		}
		if _, err := part.Write(content); err != nil {
			return nil, err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func TestReceivingEmailsFromDifferentDomains(t *testing.T) {
	server, _, tempDir, port, err := setupTestServer(t)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	defer server.Stop()

	domains := []string{"example.com", "test.org", "company.net"}
	for _, domain := range domains {
		t.Run(fmt.Sprintf("domain_%s", domain), func(t *testing.T) {
			from := fmt.Sprintf("sender@%s", domain)
			to := fmt.Sprintf("recipient@%s", domain)

			client, err := smtp.Dial(fmt.Sprintf("localhost:%d", port))
			if err != nil {
				t.Fatalf("dial failed: %v", err)
			}
			defer client.Close()

			if err := client.Mail(from, nil); err != nil {
				t.Fatalf("MAIL FROM failed: %v", err)
			}
			if err := client.Rcpt(to, nil); err != nil {
				t.Fatalf("RCPT TO failed: %v", err)
			}

			wc, err := client.Data()
			if err != nil {
				t.Fatalf("DATA failed: %v", err)
			}

			email, err := createTestEmail(from, to, "Test Subject", "Test Body", nil)
			if err != nil {
				t.Fatalf("creating email failed: %v", err)
			}

			if _, err = wc.Write(email); err != nil {
				t.Fatalf("write failed: %v", err)
			}
			if err := wc.Close(); err != nil {
				t.Fatalf("close failed: %v", err)
			}

			// Verify email was stored correctly
			domainDir := filepath.Join(tempDir, domain)
			files, err := os.ReadDir(domainDir)
			if err != nil {
				t.Fatalf("reading domain directory failed: %v", err)
			}
			if len(files) == 0 {
				t.Error("no emails stored for domain")
			}
		})
	}
}

func TestReceivingEmailsWithAttachments(t *testing.T) {
	server, _, tempDir, port, err := setupTestServer(t)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	defer server.Stop()

	attachments := map[string][]byte{
		"test.txt":  []byte("Hello, this is a test file!"),
		"image.jpg": bytes.Repeat([]byte{0xFF}, 1024), // Dummy image data
	}

	from := "sender@example.com"
	to := "recipient@example.com"

	client, err := smtp.Dial(fmt.Sprintf("localhost:%d", port))
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer client.Close()

	if err := client.Mail(from, nil); err != nil {
		t.Fatalf("MAIL FROM failed: %v", err)
	}
	if err := client.Rcpt(to, nil); err != nil {
		t.Fatalf("RCPT TO failed: %v", err)
	}

	wc, err := client.Data()
	if err != nil {
		t.Fatalf("DATA failed: %v", err)
	}

	email, err := createTestEmail(from, to, "Test with Attachments", "Email with attachments", attachments)
	if err != nil {
		t.Fatalf("creating email failed: %v", err)
	}

	if _, err = wc.Write(email); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if err := wc.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	// Give the server a moment to process and store the email
	time.Sleep(100 * time.Millisecond)

	// Verify email was stored with attachments
	storedDir := filepath.Join(tempDir, "example.com", "recipient", "IN")
	files, err := os.ReadDir(storedDir)
	if err != nil {
		t.Fatalf("reading stored directory failed: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no email file found")
	}

	// Read the first (and should be only) email file
	content, err := os.ReadFile(filepath.Join(storedDir, files[0].Name()))
	if err != nil {
		t.Fatalf("reading stored email failed: %v", err)
	}

	for filename := range attachments {
		if !bytes.Contains(content, []byte(filename)) {
			t.Errorf("attachment %s not found in stored email", filename)
		}
	}
}

func TestStressWithMultipleDomains(t *testing.T) {
	server, _, tempDir, port, err := setupTestServer(t)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	defer server.Stop()

	const (
		numDomains      = 100
		emailsPerDomain = 10
		concurrentSends = 20
	)

	start := time.Now()
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrentSends)

	for i := 0; i < numDomains; i++ {
		domain := fmt.Sprintf("domain%d.com", i)

		for j := 0; j < emailsPerDomain; j++ {
			wg.Add(1)
			semaphore <- struct{}{} // Acquire

			go func(d string, num int) {
				defer wg.Done()
				defer func() { <-semaphore }() // Release

				from := fmt.Sprintf("sender%d@%s", num, d)
				to := fmt.Sprintf("recipient%d@%s", num, d)

				client, err := smtp.Dial(fmt.Sprintf("localhost:%d", port))
				if err != nil {
					t.Errorf("dial failed for %s: %v", d, err)
					return
				}
				defer client.Close()

				if err := client.Mail(from, nil); err != nil {
					t.Errorf("MAIL FROM failed for %s: %v", d, err)
					return
				}
				if err := client.Rcpt(to, nil); err != nil {
					t.Errorf("RCPT TO failed for %s: %v", d, err)
					return
				}

				wc, err := client.Data()
				if err != nil {
					t.Errorf("DATA failed for %s: %v", d, err)
					return
				}

				email, err := createTestEmail(from, to, "Stress Test", "Test Body", nil)
				if err != nil {
					t.Errorf("creating email failed for %s: %v", d, err)
					return
				}

				if _, err = wc.Write(email); err != nil {
					t.Errorf("write failed for %s: %v", d, err)
					return
				}
				if err := wc.Close(); err != nil {
					t.Errorf("close failed for %s: %v", d, err)
					return
				}
			}(domain, j)
		}
	}

	wg.Wait()
	duration := time.Since(start)
	totalEmails := numDomains * emailsPerDomain
	emailsPerSecond := float64(totalEmails) / duration.Seconds()

	t.Logf("Processed %d emails in %v (%.2f emails/sec)", totalEmails, duration, emailsPerSecond)
	if emailsPerSecond < 100 {
		t.Errorf("Performance below target: %.2f emails/sec (target: 100+/sec)", emailsPerSecond)
	}

	// Verify storage
	domains, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("reading storage directory failed: %v", err)
	}

	if len(domains) != numDomains {
		t.Errorf("expected %d domains, got %d", numDomains, len(domains))
	}
}

func TestSimultaneousSMTPSessions(t *testing.T) {
	server, _, tempDir, port, err := setupTestServer(t)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	defer server.Stop()

	const (
		numSessions     = 10 // Number of simultaneous SMTP sessions
		emailsPerSession = 5 // Number of emails to send in each session
	)

	var wg sync.WaitGroup
	errCh := make(chan error, numSessions*emailsPerSession)

	// Start multiple sessions in parallel
	for session := 0; session < numSessions; session++ {
		wg.Add(1)
		go func(sessionID int) {
			defer wg.Done()

			// Establish SMTP connection
			client, err := smtp.Dial(fmt.Sprintf("localhost:%d", port))
			if err != nil {
				errCh <- fmt.Errorf("session %d dial failed: %w", sessionID, err)
				return
			}
			defer client.Close()

			// Send multiple emails within the same session
			for emailID := 0; emailID < emailsPerSession; emailID++ {
				from := fmt.Sprintf("sender%d@test.com", sessionID)
				to := fmt.Sprintf("recipient%d@test.com", sessionID)
				subject := fmt.Sprintf("Test Email %d from Session %d", emailID, sessionID)
				body := fmt.Sprintf("Email %d content from session %d", emailID, sessionID)

				email, err := createTestEmail(from, to, subject, body, nil)
				if err != nil {
					errCh <- fmt.Errorf("session %d email %d create failed: %w", sessionID, emailID, err)
					return
				}

				if err := client.Mail(from, nil); err != nil {
					errCh <- fmt.Errorf("session %d email %d MAIL FROM failed: %w", sessionID, emailID, err)
					return
				}

				if err := client.Rcpt(to, nil); err != nil {
					errCh <- fmt.Errorf("session %d email %d RCPT TO failed: %w", sessionID, emailID, err)
					return
				}

				wc, err := client.Data()
				if err != nil {
					errCh <- fmt.Errorf("session %d email %d DATA failed: %w", sessionID, emailID, err)
					return
				}

				if _, err = wc.Write(email); err != nil {
					errCh <- fmt.Errorf("session %d email %d write failed: %w", sessionID, emailID, err)
					return
				}

				if err := wc.Close(); err != nil {
					errCh <- fmt.Errorf("session %d email %d close failed: %w", sessionID, emailID, err)
					return
				}

				// Small delay to simulate real-world usage
				time.Sleep(10 * time.Millisecond)
			}

		}(session)
	}

	// Wait for all sessions to complete
	wg.Wait()
	close(errCh)

	// Check for any errors
	for err := range errCh {
		t.Error(err)
	}

	// Verify emails were stored correctly
	for session := 0; session < numSessions; session++ {
		domain := "test.com"
		user := fmt.Sprintf("recipient%d", session)
		userDir := filepath.Join(tempDir, domain, user, "IN")
		
		files, err := os.ReadDir(userDir)
		if err != nil {
			t.Errorf("reading directory for session %d failed: %v", session, err)
			continue
		}

		if len(files) != emailsPerSession {
			t.Errorf("session %d: expected %d emails, got %d", session, emailsPerSession, len(files))
			continue
		}

		// Verify each email's content
		for _, file := range files {
			content, err := os.ReadFile(filepath.Join(userDir, file.Name()))
			if err != nil {
				t.Errorf("reading email file %s failed: %v", file.Name(), err)
				continue
			}

			if !bytes.Contains(content, []byte(fmt.Sprintf("from session %d", session))) {
				t.Errorf("email %s does not contain expected session ID %d", file.Name(), session)
			}
		}
	}

	t.Logf("Successfully processed %d simultaneous sessions with %d emails each", numSessions, emailsPerSession)
}
