package flux

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// Config interface to avoid import cycle
type Config interface {
	GetAPIEndpoint() string
	GetDefaultNumOutputs() int
	GetDefaultAspectRatio() string
	GetDefaultFormat() string
	GetDefaultQuality() int
	GetDisableSafetyCheck() bool
}

// Client manages API communication with the Flux service
type Client struct {
	apiURL     string
	httpClient *http.Client
	config     Config
}

// NewClient creates a new Flux API client
func NewClient(config Config) *Client {
	return &Client{
		apiURL: config.GetAPIEndpoint(),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		config: config,
	}
}

// GenerateOptions represents options for image generation
type GenerateOptions struct {
	NumOutputs   int
	AspectRatio  string
	OutputFormat string
	Quality      int
	Seed         *int
}

// GenerateImages creates images based on the provided prompt
func (c *Client) GenerateImages(prompt string) ([]string, error) {
	return c.GenerateImagesWithOptions(prompt, GenerateOptions{
		NumOutputs:   c.config.GetDefaultNumOutputs(),
		AspectRatio:  c.config.GetDefaultAspectRatio(),
		OutputFormat: c.config.GetDefaultFormat(),
		Quality:      c.config.GetDefaultQuality(),
	})
}

// GenerateImagesWithOptions creates images with custom options
func (c *Client) GenerateImagesWithOptions(prompt string, opts GenerateOptions) ([]string, error) {
	if prompt == "" {
		return nil, errors.New("prompt cannot be empty")
	}

	if c.apiURL == "" {
		return nil, errors.New("API URL not configured")
	}

	input := Input{
		Prompt:             prompt,
		NumOutputs:         opts.NumOutputs,
		AspectRatio:        opts.AspectRatio,
		OutputFormat:       opts.OutputFormat,
		OutputQuality:      opts.Quality,
		DisableSafetyCheck: c.config.GetDisableSafetyCheck(),
		Seed:               opts.Seed,
	}

	payload := map[string]interface{}{"input": input}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned non-200 status code: %d", resp.StatusCode)
	}

	var urls []string
	if err := json.NewDecoder(resp.Body).Decode(&urls); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return urls, nil
}
