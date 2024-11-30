// Package main is the entry point for the Gargantua Sink SMTP server.
package main

import (
	"fmt"
	"os"

	"github.com/nathabonfim59/gargantua-sink/internal/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
