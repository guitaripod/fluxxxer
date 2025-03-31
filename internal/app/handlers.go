package app

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"fluxxxer/internal/flux"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// onGenerateClicked handles the generate button click event
func (a *App) onGenerateClicked() {
	prompt := a.entry.Text()
	if prompt == "" {
		a.setStatus("Please enter a prompt")
		return
	}

	a.spinner.Start()
	a.clearImages()
	a.setStatus("Generating images...")

	// Find aspect ratio dropdown and number of images slider
	aspectCombo := a.findAspectRatioCombo()
	numOutputsScale := a.findNumOutputsScale()
	
	// Get the selected options
	var aspectRatio string
	if aspectCombo != nil {
		selectedIdx := aspectCombo.Selected()
		if selectedIdx < uint(len(a.config.GetSupportedAspectRatios())) {
			aspectRatio = a.config.GetSupportedAspectRatios()[selectedIdx]
		} else {
			aspectRatio = a.config.GetDefaultAspectRatio()
		}
	} else {
		aspectRatio = a.config.GetDefaultAspectRatio()
	}
	
	numOutputs := a.config.GetDefaultNumOutputs()
	if numOutputsScale != nil {
		numOutputs = int(numOutputsScale.Adjustment().Value())
	}

	// Generate images with the selected options
	go func() {
		images, err := a.client.GenerateImagesWithOptions(prompt, flux.GenerateOptions{
			NumOutputs:   numOutputs,
			AspectRatio:  aspectRatio,
			OutputFormat: a.config.GetDefaultFormat(),
			Quality:      a.config.GetDefaultQuality(),
		})
		
		glib.IdleAdd(func() {
			a.spinner.Stop()
			if err != nil {
				a.setStatus(fmt.Sprintf("Error: %v", err))
				return
			}
			a.displayImages(images)
			a.setStatus(fmt.Sprintf("Generated %d images", len(images)))
		})
	}()
}

// Store references to our UI controls for easy access
var (
	aspectRatioCombo *gtk.DropDown
	numOutputsScale  *gtk.Scale
)

// findAspectRatioCombo finds the aspect ratio dropdown in the UI
func (a *App) findAspectRatioCombo() *gtk.DropDown {
	// Return cached reference if available
	if aspectRatioCombo != nil {
		return aspectRatioCombo
	}
	
	// If not found, return default values
	return nil
}

// findNumOutputsScale finds the number of outputs scale in the UI
func (a *App) findNumOutputsScale() *gtk.Scale {
	// Return cached reference if available
	if numOutputsScale != nil {
		return numOutputsScale
	}
	
	// If not found, return default values
	return nil
}

// displayImages shows the generated images in the UI
func (a *App) displayImages(urls []string) {
	// Get the available width for the images
	availableWidth := a.currentWidth
	if availableWidth == 0 {
		availableWidth = a.config.GetWindowWidth()
	}
	
	// Calculate optimal image size based on number of images and available space
	numImages := len(urls)
	if numImages == 0 {
		return
	}
	
	// Calculate how many images to show per row
	imagesPerRow := 2
	if numImages > 4 {
		imagesPerRow = 4
	} else if numImages > 2 {
		imagesPerRow = 3
	}
	
	// Minimum image size
	minImageSize := 320
	
	// Create image grid
	imageGrid := gtk.NewGrid()
	imageGrid.SetRowSpacing(16)
	imageGrid.SetColumnSpacing(16)
	imageGrid.SetRowHomogeneous(false)
	imageGrid.SetColumnHomogeneous(true)
	
	a.imageBox.Append(imageGrid)
	
	// Display each image
	for i, url := range urls {
		row := i / imagesPerRow
		col := i % imagesPerRow
		
		// Create a frame for the image
		imageFrame := gtk.NewFrame("")
		imageFrame.SetMarginStart(8)
		imageFrame.SetMarginEnd(8)
		imageFrame.SetMarginTop(8)
		imageFrame.SetMarginBottom(8)
		
		// Add styling to the frame
		imageFrame.AddCSSClass("frame")
		
		// Create a container for the image and buttons
		imageBox := gtk.NewBox(gtk.OrientationVertical, 8)
		imageBox.SetMarginStart(8)
		imageBox.SetMarginEnd(8)
		imageBox.SetMarginTop(8)
		imageBox.SetMarginBottom(8)
		
		// Add a placeholder while loading
		placeholder := gtk.NewSpinner()
		placeholder.Start()
		placeholder.SetSizeRequest(minImageSize, minImageSize)
		placeholder.SetHAlign(gtk.AlignCenter)
		placeholder.SetVAlign(gtk.AlignCenter)
		imageBox.Append(placeholder)
		
		// Set the frame content
		imageFrame.SetChild(imageBox)
		
		// Add the frame to the grid
		imageGrid.Attach(imageFrame, col, row, 1, 1)
		
		// Load the image in the background
		go func(url string, imageBox *gtk.Box, placeholder *gtk.Spinner) {
			texture, err := a.loadImageTexture(url)
			if err != nil {
				glib.IdleAdd(func() {
					// Remove the spinner
					imageBox.Remove(placeholder)
					
					// Show error message
					errorLabel := gtk.NewLabel(fmt.Sprintf("Error: %v", err))
					errorLabel.SetWrap(true)
					errorLabel.SetJustify(gtk.JustifyCenter)
					imageBox.Append(errorLabel)
				})
				return
			}
			
			glib.IdleAdd(func() {
				// Remove the spinner
				imageBox.Remove(placeholder)
				
				// Create picture widget
				picture := gtk.NewPicture()
				picture.SetPaintable(texture)
				picture.SetCanShrink(true)
				picture.SetHExpand(true)
				picture.SetVExpand(true)
				picture.SetContentFit(gtk.ContentFitContain)
				
				// Add some minimum image size
				picture.SetSizeRequest(minImageSize, minImageSize)
				
				// Create button container
				buttonBox := gtk.NewBox(gtk.OrientationHorizontal, 8)
				buttonBox.SetHAlign(gtk.AlignCenter)
				buttonBox.SetMarginTop(8)
				
				// Save button
				saveBtn := gtk.NewButtonWithLabel("Save")
				saveBtn.ConnectClicked(func() {
					a.saveImage(url)
				})
				
				// Copy button
				copyBtn := gtk.NewButtonWithLabel("Copy")
				copyBtn.ConnectClicked(func() {
					a.copyImageToClipboard(texture)
				})
				
				// Upscale button
				upscaleBtn := gtk.NewButtonWithLabel("Upscale")
				
				// Enable upscale button if the upscaler is configured
				upscaleBtn.SetSensitive(a.isUpscalerConfigured())
				
				if a.isUpscalerConfigured() {
					upscaleBtn.ConnectClicked(func() {
						// Log which image we're trying to upscale
						fmt.Printf("Attempting to upscale image from URL: %s\n", url)
						
						// Create a temporary file to save the image for upscaling
						tmpFile, err := os.CreateTemp("", "temp-image-*.png")
						if err != nil {
							a.setStatus(fmt.Sprintf("Error preparing image for upscaling: %v", err))
							return
						}
						tmpPath := tmpFile.Name()
						tmpFile.Close()
						
						// Download the image to the temp file
						go func() {
							err := a.downloadAndSaveImage(url, tmpPath)
							if err != nil {
								glib.IdleAdd(func() {
									a.setStatus(fmt.Sprintf("Error preparing image for upscaling: %v", err))
								})
								os.Remove(tmpPath)
								return
							}
							
							// Now handle the upscale
							glib.IdleAdd(func() {
								a.handleUpscaleFile(tmpPath)
							})
						}()
					})
				} else {
					upscaleBtn.SetTooltipText("Upscaler not configured. Set UPSCALER_API_URL and UPSCALER_API_KEY in your .env file.")
				}
				
				// Add buttons to container
				buttonBox.Append(saveBtn)
				buttonBox.Append(copyBtn)
				buttonBox.Append(upscaleBtn)
				
				// Add widgets to the image box
				imageBox.Append(picture)
				imageBox.Append(buttonBox)
			})
		}(url, imageBox, placeholder)
	}
}

func (a *App) loadImageTexture(url string) (*gdk.Texture, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	texture, err := gdk.NewTextureFromBytes(glib.NewBytesWithGo(data))
	if err != nil {
		return nil, err
	}

	return texture, nil
}

func (a *App) saveImage(url string) {
	dialog := gtk.NewFileChooserNative(
		"Save Image",
		&a.win.Window,
		gtk.FileChooserActionSave,
		"_Save",
		"_Cancel",
	)

	defaultName := filepath.Base(url)
	if defaultName == "" || defaultName == "." {
		defaultName = "generated_image.png"
	}
	dialog.SetCurrentName(defaultName)

	filter := gtk.NewFileFilter()
	filter.AddPattern("*.png")
	filter.SetName("PNG images")
	dialog.AddFilter(filter)

	homeDir, err := os.UserHomeDir()
	if err == nil {
		picturesDir := filepath.Join(homeDir, "Pictures")
		if _, err := os.Stat(picturesDir); err == nil {
			gfile := gio.NewFileForPath(picturesDir)
			dialog.SetCurrentFolder(gfile)
		}
	}

	responseChan := make(chan int)
	dialog.ConnectResponse(func(response int) {
		responseChan <- response
	})

	dialog.Show()

	go func() {
		response := <-responseChan
		if response == int(gtk.ResponseAccept) {
			file := dialog.File()
			if file == nil {
				glib.IdleAdd(func() {
					a.setStatus("Error: No file selected")
				})
				return
			}

			path := file.Path()

			if !strings.HasSuffix(strings.ToLower(path), ".png") {
				path += ".png"
			}

			go func() {
				err := a.downloadAndSaveImage(url, path)
				glib.IdleAdd(func() {
					if err != nil {
						a.setStatus(fmt.Sprintf("Error saving image: %v", err))
					} else {
						a.setStatus(fmt.Sprintf("Image saved to: %s", path))
					}
				})
			}()
		}

		dialog.Destroy()
	}()
}

func (a *App) downloadAndSaveImage(url, destPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download image: status code %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(destPath), "*.png")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	tmpPath := tmpFile.Name()

	defer func() {
		tmpFile.Close()
		os.Remove(tmpPath)
	}()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return fmt.Errorf("failed to write image data: %w", err)
	}

	tmpFile.Close()

	if err := os.Rename(tmpPath, destPath); err != nil {
		return fmt.Errorf("failed to save image: %w", err)
	}

	return nil
}

func (a *App) copyImageToClipboard(texture *gdk.Texture) {
	clipboard := gdk.DisplayGetDefault().Clipboard()
	clipboard.SetTexture(texture)
	a.setStatus("Image copied to clipboard")
}
