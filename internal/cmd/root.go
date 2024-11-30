// Package cmd implements command-line interface for the Gargantua Sink SMTP server.
package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/nathabonfim59/gargantua-sink/internal/smtp"
	"github.com/nathabonfim59/gargantua-sink/internal/storage"
)

// DomainConfig represents the configuration for a domain
type DomainConfig struct {
	Domain     string `json:"domain"`
	CertFile   string `json:"cert_file"`
	KeyFile    string `json:"key_file"`
	StorageDir string `json:"storage_dir"`
}

var (
	// Configuration flags
	serverPort    int
	defaultStorage string
	configFile    string
	domains       []DomainConfig

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
	rootCmd.PersistentFlags().StringVarP(&defaultStorage, "storage", "s", "", "Default storage directory for emails")
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Path to domain configuration JSON file")
	rootCmd.MarkPersistentFlagRequired("storage")
}

// loadDomainConfig loads domain configurations from a JSON file
func loadDomainConfig(configPath string) ([]DomainConfig, error) {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var config struct {
		Domains []DomainConfig `json:"domains"`
	}
	if err := json.Unmarshal(content, &config); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	return config.Domains, nil
}

// Execute starts the root command.
func Execute() error {
	return rootCmd.Execute()
}

// runServer initializes and starts the SMTP server.
func runServer(cmd *cobra.Command, args []string) error {
	// Initialize default storage
	defaultEmailStorage, err := storage.NewEmailStorage(defaultStorage)
	if err != nil {
		return fmt.Errorf("initializing default storage: %w", err)
	}

	// Create server instance
	server := smtp.NewServer(serverPort, defaultEmailStorage)

	// Load domain configurations if provided
	if configFile != "" {
		domains, err := loadDomainConfig(configFile)
		if err != nil {
			return fmt.Errorf("loading domain config: %w", err)
		}

		// Configure each domain
		for _, domain := range domains {
			if err := server.AddDomain(domain.Domain, domain.CertFile, domain.KeyFile, domain.StorageDir); err != nil {
				return fmt.Errorf("configuring domain %s: %w", domain.Domain, err)
			}
			log.Printf("Configured domain: %s (storage: %s)", domain.Domain, domain.StorageDir)
		}
	}

	log.Printf("Starting Gargantua Sink SMTP server on port %d", serverPort)
	log.Printf("Default storage directory: %s", defaultStorage)
	
	return server.Start()
}
