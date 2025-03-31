package app

import (
	"fluxxxer/internal/config"
	"fluxxxer/internal/flux"
	"fluxxxer/internal/upscaler"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// App represents the main application structure
type App struct {
	*gtk.Application
	win            *gtk.ApplicationWindow
	entry          *gtk.Entry
	spinner        *gtk.Spinner
	imageBox       *gtk.Box
	statusBar      *gtk.Label
	currentWidth   int
	
	// Mode tracking
	isGeneratorMode bool
	generatorToggle *gtk.ToggleButton
	upscalerToggle  *gtk.ToggleButton
	
	// Service clients
	client         *flux.Client
	upscalerClient *upscaler.Client
	config         *config.Config
}

// New creates a new application instance
func New() *App {
	cfg := config.NewConfig()
	
	// Create the app instance
	app := &App{
		Application:     gtk.NewApplication("com.fluxxxer.app", gio.ApplicationFlagsNone),
		client:          flux.NewClient(cfg),
		config:          cfg,
		isGeneratorMode: true, // Default to generator mode
	}
	
	// Initialize upscaler client if configured
	if cfg.IsUpscalerConfigured() {
		app.upscalerClient = upscaler.NewClient(cfg)
	}
	
	// Connect activate handler
	app.Application.ConnectActivate(app.setupUI)
	
	return app
}

// setStatus updates the status bar with a message
func (a *App) setStatus(message string) {
	a.statusBar.SetText(message)
}

// setMode switches between generator and upscaler modes
func (a *App) setMode(isGeneratorMode bool) {
	a.isGeneratorMode = isGeneratorMode
	
	// Update UI to reflect mode change
	if a.generatorToggle != nil && a.upscalerToggle != nil {
		a.generatorToggle.SetActive(isGeneratorMode)
		a.upscalerToggle.SetActive(!isGeneratorMode)
	}
	
	// Update status message
	if isGeneratorMode {
		a.setStatus("Image Generator Mode")
	} else {
		// Check if upscaler is configured
		if a.config.IsUpscalerConfigured() {
			a.setStatus("Image Upscaler Mode - Drag and drop an image to upscale")
		} else {
			a.setStatus("Upscaler not configured. Set UPSCALER_API_URL and UPSCALER_API_KEY in your .env file.")
			// Revert to generator mode if upscaler not configured
			a.setMode(true)
		}
	}
}

// isUpscalerConfigured checks if the upscaler is properly configured
func (a *App) isUpscalerConfigured() bool {
	return a.config.IsUpscalerConfigured() && a.upscalerClient != nil
}
