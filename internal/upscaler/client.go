package upscaler

import (
	"bytes"
	"encoding/base64"
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
	baseURL      string
	apiKey       string
	appID        string
	httpClient   *http.Client
	pollTimeout  time.Duration
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
	UpscaleFast         UpscaleType = "fast"
	UpscaleConservative UpscaleType = "conservative"
	UpscaleCreative     UpscaleType = "creative"
)

// UpscaleResult contains the result of an upscaling operation
type UpscaleResult struct {
	ID          string `json:"id,omitempty"`
	Status      string `json:"status,omitempty"`
	URL         string `json:"url,omitempty"`
	Error       string `json:"error,omitempty"`
	IsCompleted bool   `json:"is_completed"`
	// Additional fields that might be in the response
	Success bool   `json:"success,omitempty"`
	Message string `json:"message,omitempty"`
	Result  string `json:"result,omitempty"`
	// The API might return the URL in a different field
	ImageURL  string `json:"image_url,omitempty"`
	OutputURL string `json:"output_url,omitempty"`
}

// Base64Response represents a server response with base64-encoded image
type Base64Response struct {
	Success bool `json:"success"`
	Data    struct {
		Image string `json:"image"`
	} `json:"data"`
}

// NestedImageJSON represents the JSON structure within the base64 data
type NestedImageJSON struct {
	Image string `json:"image"`
}

// UpscaleOptions contains parameters for image upscaling
type UpscaleOptions struct {
	Type           UpscaleType // Upscaling type: fast, conservative, creative
	Prompt         string      // Prompt for conservative/creative types
	NegativePrompt string      // Negative prompt
	Seed           *int        // Seed for consistent results
	Creativity     *float64    // Creativity level (0.1-0.5)
	OutputFormat   string      // Output format: png, jpeg, webp
	StylePreset    string      // Style preset for creative upscaling
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

	// Read the full response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Log minimal info about the response to avoid terminal bloat
	fmt.Printf("Response received (%d bytes)\n", len(bodyBytes))

	// Only print a small hex preview instead of the full response
	if len(bodyBytes) > 0 {
		fmt.Println("Response preview (first 32 bytes):")
		fmt.Println(hex.Dump(bodyBytes[:min(len(bodyBytes), 32)]))
	}

	// Check if the response is binary data (image)
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
		result := UpscaleResult{
			URL:         tmpPath,
			IsCompleted: true,
		}
		return &result, nil
	}

	// Check if the response is a JSON with a base64 encoded image
	// Try to parse it as a base64 response
	var base64Response Base64Response
	if err := json.Unmarshal(bodyBytes, &base64Response); err == nil && base64Response.Success && base64Response.Data.Image != "" {
		fmt.Println("Response contains base64-encoded image data")

		// Extract outer base64 data (might be prefixed with data:image/png;base64, or similar)
		base64Data := base64Response.Data.Image
		
		// Debug the raw data URL
		fmt.Printf("Raw data URL prefix: %s\n", base64Data[:min(40, len(base64Data))])
		
		// Extract the base64 part after the "data:type;base64," prefix
		outerBase64 := ""
		if strings.HasPrefix(base64Data, "data:") && strings.Contains(base64Data, ";base64,") {
			parts := strings.SplitN(base64Data, ";base64,", 2)
			if len(parts) == 2 {
				fmt.Printf("Data URL MIME type: %s\n", parts[0])
				outerBase64 = parts[1]
				fmt.Printf("Outer base64 data (first 16 chars): %s...\n", 
					outerBase64[:min(16, len(outerBase64))])
			} else {
				return nil, fmt.Errorf("invalid data URL format: %s", base64Data[:min(50, len(base64Data))])
			}
		} else {
			return nil, fmt.Errorf("unexpected data format, missing data:type;base64, prefix")
		}
		
		// Decode the outer base64 layer
		fmt.Println("Decoding outer base64 layer")
		jsonData, err := base64.StdEncoding.DecodeString(outerBase64)
		if err != nil {
			return nil, fmt.Errorf("failed to decode outer base64 data: %w", err)
		}
		
		// The decoded data is actually JSON, so parse it
		fmt.Println("Parsing nested JSON containing the actual image")
		var nestedJSON NestedImageJSON
		if err := json.Unmarshal(jsonData, &nestedJSON); err != nil {
			// Print a hex dump of the JSON data for debugging
			fmt.Println("JSON decode failed. Data preview:")
			fmt.Println(hex.Dump(jsonData[:min(100, len(jsonData))]))
			return nil, fmt.Errorf("failed to parse nested JSON: %w", err)
		}

		// For debugging
		if nestedJSON.Image == "" {
			fmt.Println("Warning: Nested JSON image field is empty")
		}
		
		// Now we have the actual image data
		fmt.Println("Successfully extracted image data from nested JSON")
		
		// Decode the image data to get the binary data
		imgData, err := base64.StdEncoding.DecodeString(nestedJSON.Image)
		if err != nil {
			return nil, fmt.Errorf("failed to decode inner image data: %w", err)
		}

		// Determine file extension based on magic numbers
		ext := ".png" // Default to PNG
		if hasPNGSignature(imgData) {
			ext = ".png"
			fmt.Println("Detected PNG image data after decoding")
		} else if hasJPEGSignature(imgData) {
			ext = ".jpg"
			fmt.Println("Detected JPEG image data after decoding")
		} else {
			fmt.Println("Warning: Unknown image format, defaulting to PNG")
			// Print the first few bytes for debugging
			if len(imgData) > 16 {
				fmt.Println("First 16 bytes:", hex.EncodeToString(imgData[:16]))
			}
		}

		// Create a temporary file to save the image
		tmpFile, err := os.CreateTemp("", "upscaled-*"+ext)
		if err != nil {
			return nil, fmt.Errorf("failed to create temporary file: %w", err)
		}
		tmpPath := tmpFile.Name()

		// Write the binary data to the file
		if _, err := tmpFile.Write(imgData); err != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
			return nil, fmt.Errorf("failed to write image data: %w", err)
		}
		tmpFile.Close()

		fmt.Printf("📥 UPSCALED IMAGE STORED: %s (size: %d bytes)\n",
			tmpPath, len(imgData))

		result := UpscaleResult{
			URL:         tmpPath,
			IsCompleted: true,
		}
		return &result, nil
	}

	// If none of the above worked, try to decode as regular JSON response
	var result UpscaleResult
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		// If we can't decode the JSON, return an error with the response body
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
		pollResult, err := c.pollForResultID(result.ID)
		if err != nil {
			return nil, err
		}
		return pollResult, nil
	}

	return &result, nil
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

// pollForResultID polls for the result of an asynchronous upscaling operation
func (c *Client) pollForResultID(jobID string) (*UpscaleResult, error) {
	if jobID == "" {
		return nil, errors.New("job ID cannot be empty")
	}

	fmt.Printf("Polling for upscaling job result with ID: %s\n", jobID)
	
	// Set up timeout channel
	timeout := time.After(c.pollTimeout)
	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()

	// Poll until we get a completed result or timeout
	for {
		select {
		case <-ticker.C:
			// Create the request URL for polling
			requestURL := fmt.Sprintf("%s/result/%s", c.baseURL, jobID)
			
			// Create a new request
			req, err := http.NewRequest("GET", requestURL, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create poll request: %w", err)
			}
			
			// Set headers
			req.Header.Set("Authorization", "Bearer "+c.apiKey)
			req.Header.Set("X-App-ID", c.appID)
			
			// Send the request
			resp, err := c.httpClient.Do(req)
			if err != nil {
				return nil, fmt.Errorf("poll request failed: %w", err)
			}
			
			// Read the response body
			bodyBytes, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to read poll response: %w", err)
			}
			
			// Check status code
			if resp.StatusCode != http.StatusOK {
				fmt.Printf("Poll returned non-OK status: %d\n", resp.StatusCode)
				// Don't fail on non-200, just continue polling
				continue
			}
			
			// Try to parse the response
			var result UpscaleResult
			if err := json.Unmarshal(bodyBytes, &result); err != nil {
				fmt.Printf("Failed to parse poll response: %v\n", err)
				continue
			}
			
			// Check if the job is completed
			if result.IsCompleted || result.Status == "completed" || result.Status == "done" {
				fmt.Println("Upscaling job completed successfully")
				// Check if we have a URL
				if result.URL == "" {
					// Try alternative URL fields
					if result.ImageURL != "" {
						result.URL = result.ImageURL
					} else if result.OutputURL != "" {
						result.URL = result.OutputURL
					}
				}
				
				if result.URL != "" {
					return &result, nil
				}
			}
			
			// Check for errors
			if result.Error != "" {
				return nil, fmt.Errorf("upscaling job failed: %s", result.Error)
			}
			
			fmt.Printf("Job status: %s - continuing to poll...\n", result.Status)
			
		case <-timeout:
			return nil, fmt.Errorf("timeout waiting for upscaling job completion after %v", c.pollTimeout)
		}
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}