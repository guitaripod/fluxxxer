package main

import (
	"fmt"
	"os"

	"fluxxxer/internal/app"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Error loading .env file: %v\n", err)
	}

	if os.Getenv("FLUX_API_URL") == "" {
		fmt.Fprintln(os.Stderr, "Error: FLUX_API_URL environment variable is not set")
		os.Exit(1)
	}

	application := app.New()
	if code := application.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}
