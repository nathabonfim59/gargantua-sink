// Package cmd implements command-line interface for the Gargantua Sink SMTP server.
package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/nathabonfim59/gargantua-sink/internal/smtp"
	"github.com/nathabonfim59/gargantua-sink/internal/storage"
)

var (
	// Configuration flags
	serverPort  int
	storagePath string
	certFile    string
	keyFile     string

	rootCmd = &cobra.Command{
		Use:   "gargantua-sink",
		Short: "A robust SMTP server for capturing and storing emails",
		Long: `Gargantua Sink is an SMTP server designed to capture and store emails
for development and testing purposes. It provides a reliable way to intercept
and inspect emails during application development.`,
		RunE: runServer,
	}
)

func init() {
	rootCmd.PersistentFlags().IntVarP(&serverPort, "port", "p", 2525, "SMTP server listening port")
	rootCmd.PersistentFlags().StringVarP(&storagePath, "storage-path", "s", "", "Directory path for email storage")
	rootCmd.PersistentFlags().StringVar(&certFile, "tls-cert", "", "Path to TLS certificate file (optional)")
	rootCmd.PersistentFlags().StringVar(&keyFile, "tls-key", "", "Path to TLS private key file (optional)")
	rootCmd.MarkPersistentFlagRequired("storage-path")
}

// Execute starts the root command.
func Execute() error {
	return rootCmd.Execute()
}

// runServer initializes and starts the SMTP server.
func runServer(cmd *cobra.Command, args []string) error {
	emailStorage, err := storage.NewEmailStorage(storagePath)
	if err != nil {
		return err
	}

	server := smtp.NewServer(serverPort, emailStorage)

	// Configure TLS if certificate files are provided
	if certFile != "" && keyFile != "" {
		if err := server.SetTLSConfig(certFile, keyFile); err != nil {
			return fmt.Errorf("configuring TLS: %w", err)
		}
		log.Printf("TLS enabled with certificate: %s", certFile)
	}

	log.Printf("Starting Gargantua Sink SMTP server on port %d", serverPort)
	log.Printf("Emails will be stored in: %s", storagePath)
	
	return server.Start()
}
