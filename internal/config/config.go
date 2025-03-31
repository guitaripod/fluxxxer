package config

import (
	"os"
	"strconv"
	"strings"
)

// Config holds application configuration
type Config struct {
	// Flux API settings
	APIEndpoint        string
	DefaultNumOutputs  int
	DefaultAspectRatio string
	DefaultFormat      string
	DefaultQuality     int
	DisableSafetyCheck bool
	
	// Upscaler API settings
	UpscalerAPIURL     string
	UpscalerAPIKey     string
	UpscalerAppID      string
	DefaultUpscaleType string
	
	// UI settings
	WindowWidth        int
	WindowHeight       int
}

// NewConfig creates a new configuration with default values and environment overrides
func NewConfig() *Config {
	cfg := &Config{
		// Flux API settings
		APIEndpoint:        os.Getenv("FLUX_API_URL"),
		DefaultNumOutputs:  4,
		DefaultAspectRatio: "1:1",
		DefaultFormat:      "png",
		DefaultQuality:     1,
		DisableSafetyCheck: true,
		
		// Upscaler API settings
		UpscalerAPIURL:     os.Getenv("UPSCALER_API_URL"),
		UpscalerAPIKey:     os.Getenv("UPSCALER_API_KEY"),
		UpscalerAppID:      os.Getenv("UPSCALER_APP_ID"),
		DefaultUpscaleType: "fast",
		
		// UI settings
		WindowWidth:        2000,
		WindowHeight:       800,
	}
	
	// Use the default upscaler URL if not set
	if cfg.UpscalerAPIURL == "" {
		cfg.UpscalerAPIURL = "https://stability-go.fly.dev/api/v1/upscale"
	}

	// Override Flux API defaults with environment variables
	if val := os.Getenv("FLUX_NUM_OUTPUTS"); val != "" {
		if num, err := strconv.Atoi(val); err == nil && num > 0 {
			cfg.DefaultNumOutputs = num
		}
	}

	if val := os.Getenv("FLUX_ASPECT_RATIO"); val != "" {
		cfg.DefaultAspectRatio = val
	}

	if val := os.Getenv("FLUX_FORMAT"); val != "" {
		cfg.DefaultFormat = strings.ToLower(val)
	}

	if val := os.Getenv("FLUX_QUALITY"); val != "" {
		if quality, err := strconv.Atoi(val); err == nil && quality > 0 {
			cfg.DefaultQuality = quality
		}
	}

	if val := os.Getenv("FLUX_DISABLE_SAFETY"); val != "" {
		cfg.DisableSafetyCheck = val == "true" || val == "1" || val == "yes"
	}
	
	// Override Upscaler API defaults with environment variables
	if val := os.Getenv("UPSCALER_TYPE"); val != "" {
		cfg.DefaultUpscaleType = strings.ToLower(val)
	}

	// Override UI defaults with environment variables
	if val := os.Getenv("FLUX_WINDOW_WIDTH"); val != "" {
		if width, err := strconv.Atoi(val); err == nil && width > 0 {
			cfg.WindowWidth = width
		}
	}

	if val := os.Getenv("FLUX_WINDOW_HEIGHT"); val != "" {
		if height, err := strconv.Atoi(val); err == nil && height > 0 {
			cfg.WindowHeight = height
		}
	}

	return cfg
}

// Flux API getters

// GetAPIEndpoint returns the API endpoint
func (c *Config) GetAPIEndpoint() string {
	return c.APIEndpoint
}

// GetDefaultNumOutputs returns the default number of outputs
func (c *Config) GetDefaultNumOutputs() int {
	return c.DefaultNumOutputs
}

// GetDefaultAspectRatio returns the default aspect ratio
func (c *Config) GetDefaultAspectRatio() string {
	return c.DefaultAspectRatio
}

// GetDefaultFormat returns the default output format
func (c *Config) GetDefaultFormat() string {
	return c.DefaultFormat
}

// GetDefaultQuality returns the default quality setting
func (c *Config) GetDefaultQuality() int {
	return c.DefaultQuality
}

// GetDisableSafetyCheck returns whether safety checks are disabled
func (c *Config) GetDisableSafetyCheck() bool {
	return c.DisableSafetyCheck
}

// Upscaler API getters

// GetUpscalerAPIURL returns the upscaler API URL
func (c *Config) GetUpscalerAPIURL() string {
	return c.UpscalerAPIURL
}

// GetUpscalerAPIKey returns the upscaler API key
func (c *Config) GetUpscalerAPIKey() string {
	return c.UpscalerAPIKey
}

// GetUpscalerAppID returns the upscaler app ID
func (c *Config) GetUpscalerAppID() string {
	return c.UpscalerAppID
}

// GetDefaultUpscaleType returns the default upscaling type
func (c *Config) GetDefaultUpscaleType() string {
	return c.DefaultUpscaleType
}

// UI getters

// GetWindowWidth returns the default window width
func (c *Config) GetWindowWidth() int {
	return c.WindowWidth
}

// GetWindowHeight returns the default window height
func (c *Config) GetWindowHeight() int {
	return c.WindowHeight
}

// Helper methods

// GetSupportedAspectRatios returns a list of supported aspect ratios
func (c *Config) GetSupportedAspectRatios() []string {
	return []string{"1:1", "4:3", "3:4", "16:9", "9:16"}
}

// GetSupportedUpscaleTypes returns a list of supported upscaling types
func (c *Config) GetSupportedUpscaleTypes() []string {
	return []string{"fast", "conservative", "creative"}
}

// IsUpscalerConfigured returns true if the upscaler is configured
func (c *Config) IsUpscalerConfigured() bool {
	return c.UpscalerAPIURL != "" && c.UpscalerAPIKey != ""
}