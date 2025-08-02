package config

import (
	"fmt"
	"log"
)

// Example demonstrates how to use the config package
func ExampleUsage() {
	// Load configuration
	cfg, err := Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Use configuration
	fmt.Printf("Starting server on %s\n", cfg.Address())
	fmt.Printf("Log level: %s\n", cfg.LogLevel)
	fmt.Printf("Persistence: %s\n", cfg.PersistenceType)

	if cfg.IsDebugEnabled() {
		fmt.Println("Debug mode enabled")
	}

	// Example of different persistence configurations
	switch cfg.PersistenceType {
	case PersistenceMemory:
		fmt.Println("Using in-memory storage")
	case PersistenceFile:
		fmt.Printf("Using file storage: %s\n", cfg.PersistencePath)
	case PersistenceDB:
		fmt.Printf("Using database: %s\n", cfg.DatabaseURL)
	}
}
