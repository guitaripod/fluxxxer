package main

import (
	"fmt"
	"os"
	"path/filepath"

	"fluxxxer/internal/app"
	"github.com/joho/godotenv"
)

// Version information (can be set at build time)
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func main() {
	// Try to load environment from different possible locations
	loadEnvironment()

	// Validate required environment variables
	if os.Getenv("FLUX_API_URL") == "" {
		fmt.Fprintln(os.Stderr, "Error: FLUX_API_URL environment variable is not set")
		fmt.Fprintln(os.Stderr, "Please set it in your .env file or environment")
		os.Exit(1)
	}

	// Create and run the application
	application := app.New()
	if code := application.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}

// loadEnvironment tries to load environment variables from multiple locations
func loadEnvironment() {
	// Try current directory first
	if err := godotenv.Load(); err == nil {
		return
	}

	// Try user's home directory
	home, err := os.UserHomeDir()
	if err == nil {
		homePath := filepath.Join(home, ".fluxxxer", ".env")
		if err := godotenv.Load(homePath); err == nil {
			return
		}
	}

	// Try XDG config directory
	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig == "" && home != "" {
		xdgConfig = filepath.Join(home, ".config")
	}
	
	if xdgConfig != "" {
		xdgPath := filepath.Join(xdgConfig, "fluxxxer", ".env")
		if err := godotenv.Load(xdgPath); err == nil {
			return
		}
	}

	// Try executable directory
	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		execEnvPath := filepath.Join(execDir, ".env")
		if err := godotenv.Load(execEnvPath); err == nil {
			return
		}
	}

	// Log that no .env file was found but continue anyway
	fmt.Fprintf(os.Stderr, "Warning: No .env file found. Using environment variables.\n")
}
