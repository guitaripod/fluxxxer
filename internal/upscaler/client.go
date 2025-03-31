package upscaler

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Client handles communication with the Stability AI upscaling service
type Client struct {
	baseURL     string
	apiKey      string
	appID       string
	httpClient  *http.Client
	pollTimeout time.Duration
	pollInterval time.Duration
}

// Config interface to avoid import cycle
type Config interface {
	GetUpscalerAPIURL() string
	GetUpscalerAPIKey() string
	GetUpscalerAppID() string
}

// UpscaleType represents the available upscaling methods
type UpscaleType string

const (
	UpscaleFast        UpscaleType = "fast"
	UpscaleConservative UpscaleType = "conservative"
	UpscaleCreative    UpscaleType = "creative"
)

// UpscaleResult contains the result of an upscaling operation
type UpscaleResult struct {
	ID          string `json:"id,omitempty"`
	Status      string `json:"status,omitempty"`
	URL         string `json:"url,omitempty"`
	Error       string `json:"error,omitempty"`
	IsCompleted bool   `json:"is_completed"`
	// Additional fields that might be in the response
	Success     bool   `json:"success,omitempty"`
	Message     string `json:"message,omitempty"`
	Result      string `json:"result,omitempty"`
	// The API might return the URL in a different field
	ImageURL    string `json:"image_url,omitempty"`
	OutputURL   string `json:"output_url,omitempty"`
}

// UpscaleOptions contains parameters for image upscaling
type UpscaleOptions struct {
	Type            UpscaleType // Upscaling type: fast, conservative, creative
	Prompt          string      // Prompt for conservative/creative types
	NegativePrompt  string      // Negative prompt
	Seed            *int        // Seed for consistent results
	Creativity      *float64    // Creativity level (0.1-0.5)
	OutputFormat    string      // Output format: png, jpeg, webp
	StylePreset     string      // Style preset for creative upscaling
}

// NewClient creates a new upscaler client with the given configuration
func NewClient(config Config) *Client {
	return &Client{
		baseURL:      config.GetUpscalerAPIURL(),
		apiKey:       config.GetUpscalerAPIKey(),
		appID:        config.GetUpscalerAppID(),
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		pollTimeout:  5 * time.Minute,
		pollInterval: 2 * time.Second,
	}
}

// UpscaleImageFromPath upscales an image file and returns the result
func (c *Client) UpscaleImageFromPath(imagePath string, opts UpscaleOptions) (*UpscaleResult, error) {
	if imagePath == "" {
		return nil, errors.New("image path cannot be empty")
	}

	// Open the image file
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image file: %w", err)
	}
	defer file.Close()

	// Read file stats to verify size
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file stats: %w", err)
	}
	
	// Verify the file size is reasonable
	if fileInfo.Size() == 0 {
		return nil, fmt.Errorf("image file is empty")
	}
	
	// Check if the image is too large (over 5MB) to prevent server OOM
	const maxSizeBytes = 5 * 1024 * 1024 // 5MB
	if fileInfo.Size() > maxSizeBytes {
		return nil, fmt.Errorf("image file is too large (%d MB). Maximum size is 5MB. Please resize the image before upscaling", 
			fileInfo.Size()/(1024*1024))
	}
	
	// Print file information
	fmt.Printf("Image file: %s, size: %d bytes\n", imagePath, fileInfo.Size())

	// Create multipart form - using same approach as curl
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add the image file - IMPORTANT: field name must be "image"
	part, err := writer.CreateFormFile("image", filepath.Base(imagePath))
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	// Read the file into a buffer to ensure we get all data
	fileData, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file data: %w", err)
	}
	
	// Write the file data
	_, err = part.Write(fileData)
	if err != nil {
		return nil, fmt.Errorf("failed to write file data: %w", err)
	}

	// Simplify - just add the minimal required fields as your curl example does
	writer.WriteField("type", string(opts.Type))
	
	// Only add the other fields if they're explicitly set
	if opts.Type == UpscaleConservative || opts.Type == UpscaleCreative {
		if opts.Prompt != "" {
			writer.WriteField("prompt", opts.Prompt)
		}
	}

	if opts.NegativePrompt != "" {
		writer.WriteField("negative_prompt", opts.NegativePrompt)
	}

	if opts.Seed != nil {
		writer.WriteField("seed", fmt.Sprintf("%d", *opts.Seed))
	}

	if opts.Creativity != nil {
		writer.WriteField("creativity", fmt.Sprintf("%.2f", *opts.Creativity))
	}

	if opts.OutputFormat != "" {
		writer.WriteField("output_format", opts.OutputFormat)
	}

	if opts.StylePreset != "" {
		writer.WriteField("style_preset", opts.StylePreset)
	}

	// Close the multipart writer
	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create the request
	// Use the baseURL directly as it should already be the complete endpoint
	requestURL := c.baseURL
	
	// Create a new request
	req, err := http.NewRequest("POST", requestURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers exactly as in the example curl command
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("X-App-ID", c.appID)
	
	// Add similar headers as curl would to mimic it as closely as possible
	req.Header.Set("User-Agent", "curl/8.1.2")
	req.Header.Set("Accept", "*/*")

	// Print debug info about the request
	fmt.Printf("Upscaler request:\n")
	fmt.Printf("- URL: %s\n", requestURL)
	fmt.Printf("- Method: %s\n", req.Method)
	fmt.Printf("- Content-Type: %s\n", req.Header.Get("Content-Type"))
	
	// Safely print a truncated API key (if available)
	apiKeyPrefix := ""
	if len(c.apiKey) > 0 {
		apiKeyPrefix = c.apiKey[:min(len(c.apiKey), 5)]
	}
	fmt.Printf("- Authorization: Bearer %s...\n", apiKeyPrefix)
	
	fmt.Printf("- X-App-ID: %s\n", c.appID)
	fmt.Printf("- File path: %s\n", imagePath)
	
	// Try the request with retries for server errors
	maxRetries := 3
	var resp *http.Response
	var requestErr error
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Send the request
		resp, requestErr = c.httpClient.Do(req)
		if requestErr != nil {
			return nil, fmt.Errorf("request failed: %w", requestErr)
		}
		
		// If we get a 5xx server error and this isn't our last attempt, retry
		if resp.StatusCode >= 500 && attempt < maxRetries {
			fmt.Printf("Got server error %d, retrying (%d/%d)...\n", 
				resp.StatusCode, attempt, maxRetries)
			resp.Body.Close()
			time.Sleep(time.Second * time.Duration(attempt)) // Backoff
			continue
		}
		
		// Otherwise break out of the retry loop
		break
	}
	
	defer resp.Body.Close()
	
	// Print debug info about the response
	fmt.Printf("Upscaler response: Status=%s, ContentType=%s\n", 
		resp.Status, resp.Header.Get("Content-Type"))

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		// Try to read the response body
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyText := string(bodyBytes)
		
		// Print all response headers for debugging
		fmt.Println("Response headers:")
		for key, values := range resp.Header {
			for _, value := range values {
				fmt.Printf("  %s: %s\n", key, value)
			}
		}
		
		// Special handling for 502 Bad Gateway errors
		if resp.StatusCode == http.StatusBadGateway {
			return nil, fmt.Errorf("server temporarily unavailable (502 Bad Gateway). " +
				"The upscaler service might be down or overloaded. Please try again later")
		}
		
		return nil, fmt.Errorf("API returned error status: %d, body: %s, URL: %s", 
			resp.StatusCode, bodyText, requestURL)
	}

	// Parse the response
	var result UpscaleResult
	bodyBytes, _ := io.ReadAll(resp.Body)
	
	// Log minimal info about the response to avoid terminal bloat
	fmt.Printf("Response received (%d bytes)\n", len(bodyBytes))
	
	// Only print a small hex preview instead of the full response
	if len(bodyBytes) > 0 {
		fmt.Println("Response preview (first 32 bytes):")
		fmt.Println(hex.Dump(bodyBytes[:min(len(bodyBytes), 32)]))
	}
	
	// Check if the response is binary data (image)
	// Even if Content-Type is application/json, the server may be incorrectly sending binary data
	if len(bodyBytes) > 0 && (hasPNGSignature(bodyBytes) || hasJPEGSignature(bodyBytes) || !isJSONResponse(bodyBytes)) {
		fmt.Println("Response appears to be binary image data")
		
		// Determine file extension based on image signature
		ext := ".png" // Default to PNG
		if hasPNGSignature(bodyBytes) {
			ext = ".png"
			fmt.Println("Detected PNG image data")
		} else if hasJPEGSignature(bodyBytes) {
			ext = ".jpg"
			fmt.Println("Detected JPEG image data")
		} else {
			fmt.Println("Unknown image format, defaulting to PNG")
		}
		
		// Create a temporary file to save the image
		tmpFile, err := os.CreateTemp("", "upscaled-*"+ext)
		if err != nil {
			return nil, fmt.Errorf("failed to create temporary file: %w", err)
		}
		tmpPath := tmpFile.Name()
		
		// Write the binary data to the file
		if _, err := tmpFile.Write(bodyBytes); err != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
			return nil, fmt.Errorf("failed to write image data: %w", err)
		}
		tmpFile.Close()
		
		// Log clearly where the image is stored with a distinctive message
		fmt.Printf("📥 UPSCALED IMAGE STORED: %s (size: %d bytes)\n", 
			tmpPath, len(bodyBytes))
		fmt.Printf("   Image is temporarily stored. Use the Save button when prompted to save permanently.\n")
		
		// Set the URL to the local file path
		result.URL = tmpPath
		result.IsCompleted = true
		return &result, nil
	}
	
	// Try to decode the response
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w, body: %s", err, string(bodyBytes))
	}
	
	// Check if there's a URL in the response in any of the possible fields
	if result.URL == "" {
		// Try to use other possible URL fields
		if result.ImageURL != "" {
			result.URL = result.ImageURL
		} else if result.OutputURL != "" {
			result.URL = result.OutputURL
		} else if result.Result != "" && (strings.HasPrefix(result.Result, "http://") || strings.HasPrefix(result.Result, "https://")) {
			// Sometimes the URL might be in the Result field
			result.URL = result.Result
		}
		
		// If we still don't have a URL
		if result.URL == "" {
			return nil, fmt.Errorf("no upscaled image URL in response: %s", string(bodyBytes))
		}
	}

	// For creative/conservative upscaling, we need to poll for the result
	if opts.Type == UpscaleCreative || opts.Type == UpscaleConservative {
		if result.ID == "" {
			return nil, errors.New("no job ID returned for async upscaling")
		}

		// Poll for the result
		pollResult, err := c.pollForResult(result.ID)
		if err != nil {
			return nil, err
		}
		return pollResult, nil
	}

	return &result, nil
}

// UpscaleImageFromURL upscales an image from a URL (downloads it first)
func (c *Client) UpscaleImageFromURL(imageURL string, opts UpscaleOptions) (*UpscaleResult, error) {
	if imageURL == "" {
		return nil, errors.New("image URL cannot be empty")
	}

	// Download the image to a temporary file
	resp, err := http.Get(imageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download image, status: %d", resp.StatusCode)
	}

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "image-*."+detectExtension(imageURL))
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath) // Clean up temp file after we're done

	// Copy the image data to the temp file
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to write image data: %w", err)
	}
	tmpFile.Close()

	// Now upscale the local image
	return c.UpscaleImageFromPath(tmpPath, opts)
}

// pollForResult polls the API for the result of an async upscaling job
func (c *Client) pollForResult(jobID string) (*UpscaleResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.pollTimeout)
	defer cancel()

	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, errors.New("polling timeout exceeded")
		case <-ticker.C:
			result, err := c.checkResultStatus(jobID)
			if err != nil {
				return nil, err
			}

			if result.IsCompleted {
				return result, nil
			}
		}
	}
}

// checkResultStatus checks the status of an async upscaling job
func (c *Client) checkResultStatus(jobID string) (*UpscaleResult, error) {
	// Create the request
	// Get the base part of the URL (before /api/v1/upscale)
	baseURL := c.baseURL
	if strings.Contains(baseURL, "/api/v1/upscale") {
		baseURL = baseURL[:strings.Index(baseURL, "/api/v1/upscale")]
	}
	
	// Construct the result URL
	resultURL := fmt.Sprintf("%s/api/v1/upscale/result/%s", baseURL, jobID)
	
	req, err := http.NewRequest("GET", resultURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create status request: %w", err)
	}

	// Set headers exactly as in the example curl command
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("X-App-ID", c.appID)

	// Send the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("status request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned error status: %d, body: %s", resp.StatusCode, bodyBytes)
	}

	// Parse the response
	var result UpscaleResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode status response: %w", err)
	}

	return &result, nil
}

// detectExtension tries to determine the file extension from a URL
func detectExtension(url string) string {
	ext := filepath.Ext(url)
	if ext == "" {
		return "jpg" // Default to jpg if no extension found
	}
	return ext[1:] // Remove the leading dot
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// isJSONResponse checks if the response appears to be JSON data
func isJSONResponse(data []byte) bool {
	// Check if it starts with '{' or '[' which would indicate JSON
	if len(data) == 0 {
		return false
	}
	
	// Trim any whitespace
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return false
	}
	
	// Check first character
	firstChar := trimmed[0]
	return firstChar == '{' || firstChar == '['
}

// PNG signature: 89 50 4E 47 0D 0A 1A 0A
var pngSignature = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

// hasPNGSignature checks if data starts with the PNG file signature
func hasPNGSignature(data []byte) bool {
	if len(data) < len(pngSignature) {
		return false
	}
	return bytes.Equal(data[:len(pngSignature)], pngSignature)
}

// JPEG signatures: FF D8 FF
var jpegSignature = []byte{0xFF, 0xD8, 0xFF}

// hasJPEGSignature checks if data starts with a JPEG file signature
func hasJPEGSignature(data []byte) bool {
	if len(data) < len(jpegSignature) {
		return false
	}
	return bytes.Equal(data[:len(jpegSignature)], jpegSignature)
}

// SetPollSettings adjusts the polling timeout and interval
func (c *Client) SetPollSettings(timeout, interval time.Duration) {
	if timeout > 0 {
		c.pollTimeout = timeout
	}
	if interval > 0 {
		c.pollInterval = interval
	}
}