// Package cmd implements command-line interface for the Gargantua Sink SMTP server.
package cmd

import (
	"github.com/spf13/cobra"
)

var (
	// Configuration flags
	serverPort  int
	storagePath string

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
	rootCmd.MarkPersistentFlagRequired("storage-path")
}

// Execute starts the root command.
func Execute() error {
	return rootCmd.Execute()
}

// runServer initializes and starts the SMTP server.
func runServer(cmd *cobra.Command, args []string) error {
	// TODO: Initialize and start SMTP server
	return nil
}
