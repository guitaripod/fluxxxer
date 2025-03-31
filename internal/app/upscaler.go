package app

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"fluxxxer/internal/upscaler"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// handleUpscaleFile processes an image file for upscaling
func (a *App) handleUpscaleFile(filePath string) {
	if !a.isUpscalerConfigured() {
		a.setStatus("Upscaler not configured. Please set UPSCALER_API_URL and UPSCALER_API_KEY in your .env file.")
		return
	}

	// Check if file exists and is an image
	if !isImageFile(filePath) {
		a.setStatus(fmt.Sprintf("File is not a supported image format: %s", filePath))
		return
	}

	// Show the upscale confirmation dialog
	a.showUpscaleConfirmDialog(filePath)
}

// showUpscaleConfirmDialog shows a dialog with upscale options
func (a *App) showUpscaleConfirmDialog(imagePath string) {
	// Create dialog
	dialog := gtk.NewDialog()
	dialog.SetTitle("Upscale Image")
	dialog.SetTransientFor(&a.win.Window)
	dialog.SetModal(true)
	dialog.SetDefaultSize(600, 400)

	// Get dialog content area
	contentArea := dialog.ContentArea()
	contentArea.SetMarginTop(16)
	contentArea.SetMarginBottom(16)
	contentArea.SetMarginStart(16)
	contentArea.SetMarginEnd(16)
	contentArea.SetSpacing(16)

	// Create a box for the image preview
	previewBox := gtk.NewBox(gtk.OrientationVertical, 8)
	previewBox.SetVExpand(true)
	previewBox.SetHExpand(true)

	// Add a preview label
	previewLabel := gtk.NewLabel("Image Preview")
	// previewLabel.AddCSSClass("title-3") - Not available in this version
	previewBox.Append(previewLabel)

	// Add the image preview
	imageFrame := gtk.NewFrame("")
	imageFrame.SetVExpand(true)
	imageFrame.SetHExpand(true)

	// Load and display the image preview
	texture, err := loadTextureFromFile(imagePath)
	if err != nil {
		errorLabel := gtk.NewLabel(fmt.Sprintf("Error loading image: %v", err))
		imageFrame.SetChild(errorLabel)
	} else {
		picture := gtk.NewPicture()
		picture.SetPaintable(texture)
		picture.SetCanShrink(true)
		picture.SetHExpand(true)
		picture.SetVExpand(true)
		picture.SetContentFit(gtk.ContentFitContain)
		imageFrame.SetChild(picture)
	}

	previewBox.Append(imageFrame)

	// Create options box
	optionsBox := gtk.NewBox(gtk.OrientationVertical, 8)
	optionsBox.SetMarginTop(16)

	// Upscale type
	typeBox := gtk.NewBox(gtk.OrientationHorizontal, 8)
	typeLabel := gtk.NewLabel("Upscale Type:")
	typeLabel.SetHAlign(gtk.AlignStart)
	typeLabel.SetXAlign(0)
	typeLabel.SetWidthChars(12)

	typeCombo := gtk.NewDropDown(nil, nil)
	typeCombo.SetModel(gtk.NewStringList(a.config.GetSupportedUpscaleTypes()))
	typeCombo.SetHExpand(true)

	// Set default upscale type
	defaultType := a.config.GetDefaultUpscaleType()
	for i, t := range a.config.GetSupportedUpscaleTypes() {
		if t == defaultType {
			typeCombo.SetSelected(uint(i))
			break
		}
	}

	typeBox.Append(typeLabel)
	typeBox.Append(typeCombo)

	// Prompt for conservative and creative modes
	promptBox := gtk.NewBox(gtk.OrientationHorizontal, 8)
	promptLabel := gtk.NewLabel("Prompt:")
	promptLabel.SetHAlign(gtk.AlignStart)
	promptLabel.SetXAlign(0)
	promptLabel.SetWidthChars(12)

	promptEntry := gtk.NewEntry()
	promptEntry.SetPlaceholderText("Enter a prompt to guide upscaling (for conservative/creative modes)")
	promptEntry.SetHExpand(true)

	promptBox.Append(promptLabel)
	promptBox.Append(promptEntry)

	// Output format
	formatBox := gtk.NewBox(gtk.OrientationHorizontal, 8)
	formatLabel := gtk.NewLabel("Output Format:")
	formatLabel.SetHAlign(gtk.AlignStart)
	formatLabel.SetXAlign(0)
	formatLabel.SetWidthChars(12)

	formatCombo := gtk.NewDropDown(nil, nil)
	formatCombo.SetModel(gtk.NewStringList([]string{"png", "jpeg", "webp"}))
	formatCombo.SetHExpand(true)

	// Default to png
	formatCombo.SetSelected(0)

	formatBox.Append(formatLabel)
	formatBox.Append(formatCombo)

	// Add options to the options box
	optionsBox.Append(typeBox)
	optionsBox.Append(promptBox)
	optionsBox.Append(formatBox)

	// Add spinner for loading state
	spinnerBox := gtk.NewBox(gtk.OrientationHorizontal, 8)
	spinnerBox.SetHAlign(gtk.AlignCenter)
	spinnerBox.SetMarginTop(16)
	
	spinner := gtk.NewSpinner()
	spinnerLabel := gtk.NewLabel("Upscaling image...")
	
	spinnerBox.Append(spinner)
	spinnerBox.Append(spinnerLabel)
	
	// Hide the spinner initially
	spinnerBox.SetVisible(false)

	// Add everything to the content area
	contentArea.Append(previewBox)
	contentArea.Append(optionsBox)
	contentArea.Append(spinnerBox)

	// Add buttons to the dialog
	dialog.AddButton("Cancel", int(gtk.ResponseCancel))
	dialog.AddButton("Upscale", int(gtk.ResponseAccept))
	// Note: Unable to style the button in this version

	// Connect response handler
	dialog.ConnectResponse(func(responseId int) {
		if responseId == int(gtk.ResponseAccept) {
			// Get selected options
			upscaleType := a.config.GetSupportedUpscaleTypes()[typeCombo.Selected()]
			prompt := promptEntry.Text()
			outputFormat := []string{"png", "jpeg", "webp"}[formatCombo.Selected()]

			// Show spinner
			spinnerBox.SetVisible(true)
			spinner.Start()
			// upscaleButton.SetSensitive(false) - Not available in this version

			// Check if the image file is too large and might cause OOM
			fileInfo, err := os.Stat(imagePath)
			if err == nil && fileInfo.Size() > 5*1024*1024 {
				// Display a warning that the image is large and might cause OOM
				a.setStatus(fmt.Sprintf("Warning: Image is large (%d MB). Server may run out of memory.", 
					fileInfo.Size()/(1024*1024)))
			}
			
			// Upscale the image
			go a.upscaleImage(imagePath, upscaler.UpscaleOptions{
				Type:         upscaler.UpscaleType(upscaleType),
				Prompt:       prompt,
				OutputFormat: outputFormat,
			}, func(result *upscaler.UpscaleResult, err error) {
				// Update UI on main thread
				glib.IdleAdd(func() {
					spinner.Stop()
					spinnerBox.SetVisible(false)
					
					if err != nil {
						errMsg := fmt.Sprintf("Error upscaling image: %v", err)
						a.setStatus(errMsg)
						
						// Show a detailed error in the console
						fmt.Println("======== UPSCALING FAILED ========")
						fmt.Println(errMsg)
						fmt.Println("=================================")
						
						dialog.Destroy()
						return
					}
					
					// Check if we have a URL in the result
					if result == nil || result.URL == "" {
						errMsg := "No upscaled image URL returned from server"
						a.setStatus(errMsg)
						fmt.Println("======== UPSCALING FAILED ========")
						fmt.Println(errMsg)
						fmt.Println("=================================")
						
						dialog.Destroy()
						return
					}
					
					// Check if the URL is a local file path (from direct binary response)
					if result.URL != "" && strings.HasPrefix(result.URL, "/tmp/upscaled-") {
						fmt.Println("Using direct upscaled image from local path:", result.URL)
						
						// Load image from the temporary file
						texture, err := loadTextureFromFile(result.URL)
						if err != nil {
							a.setStatus(fmt.Sprintf("Error loading upscaled image: %v", err))
							dialog.Destroy()
							return
						}
						
						// Show the image in a dialog
						a.showUpscaledImageDialog(texture, result.URL, filepath.Base(imagePath))
						dialog.Destroy()
					} else if result.URL != "" {
						// Download and save the upscaled image from URL
						fmt.Println("Downloading upscaled image from URL:", result.URL)
						a.handleUpscaledImage(result, filepath.Base(imagePath))
						dialog.Destroy()
					} else {
						a.setStatus("Error: No upscaled image URL returned")
						dialog.Destroy()
					}
				})
			})
		} else {
			dialog.Destroy()
		}
	})

	// Make the prompt entry initially insensitive if fast mode is selected
	if defaultType == "fast" {
		promptEntry.SetSensitive(false)
	}

	// TODO: Connect to change event when available
	// Make prompt entry sensitive/insensitive based on upscale type

	dialog.Show()
}

// upscaleImage sends a request to upscale the image
func (a *App) upscaleImage(imagePath string, opts upscaler.UpscaleOptions, callback func(*upscaler.UpscaleResult, error)) {
	// Validate options
	if opts.Type == upscaler.UpscaleConservative || opts.Type == upscaler.UpscaleCreative {
		if opts.Prompt == "" {
			callback(nil, fmt.Errorf("prompt is required for %s upscaling", opts.Type))
			return
		}
	}

	// Call the upscaler client
	result, err := a.upscalerClient.UpscaleImageFromPath(imagePath, opts)
	callback(result, err)
}

// handleUpscaledImage processes and displays the upscaled image
func (a *App) handleUpscaledImage(result *upscaler.UpscaleResult, originalName string) {
	// Check if the URL is already a local file (direct binary response handling)
	if strings.HasPrefix(result.URL, "/tmp/upscaled-") {
		fmt.Println("Image is already local at:", result.URL)
		a.setStatus("Loading upscaled image...")
		
		go func() {
			// Load the image for display
			texture, err := loadTextureFromFile(result.URL)
			if err != nil {
				glib.IdleAdd(func() {
					a.setStatus(fmt.Sprintf("Error loading upscaled image: %v", err))
				})
				return
			}
			
			// Show the upscaled image in a dialog
			glib.IdleAdd(func() {
				a.showUpscaledImageDialog(texture, result.URL, originalName)
			})
		}()
		return
	}

	// Otherwise download from URL
	a.setStatus("Downloading upscaled image...")
	
	go func() {
		// Create a temporary file
		ext := filepath.Ext(originalName)
		if ext == "" {
			ext = ".png"
		}
		
		tmpFile, err := os.CreateTemp("", "upscaled-*"+ext)
		if err != nil {
			glib.IdleAdd(func() {
				a.setStatus(fmt.Sprintf("Error creating temporary file: %v", err))
			})
			return
		}
		
		tmpPath := tmpFile.Name()
		defer tmpFile.Close()
		
		// Download the image
		fmt.Println("Downloading from URL:", result.URL)
		resp, err := http.Get(result.URL)
		if err != nil {
			glib.IdleAdd(func() {
				a.setStatus(fmt.Sprintf("Error downloading upscaled image: %v", err))
			})
			return
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			glib.IdleAdd(func() {
				a.setStatus(fmt.Sprintf("Error downloading upscaled image: status code %d", resp.StatusCode))
			})
			return
		}
		
		// Save the image to the temporary file
		_, err = io.Copy(tmpFile, resp.Body)
		if err != nil {
			glib.IdleAdd(func() {
				a.setStatus(fmt.Sprintf("Error saving upscaled image: %v", err))
			})
			return
		}
		
		// Close the file to ensure all data is written
		tmpFile.Close()
		
		// Load the image for display
		texture, err := loadTextureFromFile(tmpPath)
		if err != nil {
			glib.IdleAdd(func() {
				a.setStatus(fmt.Sprintf("Error loading upscaled image: %v", err))
			})
			return
		}
		
		// Show the upscaled image in a dialog
		glib.IdleAdd(func() {
			a.showUpscaledImageDialog(texture, tmpPath, originalName)
		})
	}()
}

// showUpscaledImageDialog displays the upscaled image with options to save or copy
func (a *App) showUpscaledImageDialog(texture *gdk.Texture, tmpPath, originalName string) {
	// Create dialog
	dialog := gtk.NewDialog()
	dialog.SetTitle("Upscaled Image")
	dialog.SetTransientFor(&a.win.Window)
	dialog.SetModal(true)
	dialog.SetDefaultSize(800, 600)
	
	// Get content area
	contentArea := dialog.ContentArea()
	contentArea.SetMarginTop(16)
	contentArea.SetMarginBottom(16)
	contentArea.SetMarginStart(16)
	contentArea.SetMarginEnd(16)
	
	// Create main box
	mainBox := gtk.NewBox(gtk.OrientationVertical, 16)
	mainBox.SetVExpand(true)
	mainBox.SetHExpand(true)
	
	// Add a label
	titleLabel := gtk.NewLabel("Upscaled Image")
	// titleLabel.AddCSSClass("title-2") - Not available in this version
	mainBox.Append(titleLabel)
	
	// Add scroll window for the image
	scrollWin := gtk.NewScrolledWindow()
	scrollWin.SetVExpand(true)
	scrollWin.SetHExpand(true)
	
	// Create and add the picture
	picture := gtk.NewPicture()
	picture.SetPaintable(texture)
	picture.SetCanShrink(true)
	picture.SetHExpand(true)
	picture.SetVExpand(true)
	picture.SetContentFit(gtk.ContentFitContain)
	
	scrollWin.SetChild(picture)
	mainBox.Append(scrollWin)
	
	// Add button box
	buttonBox := gtk.NewBox(gtk.OrientationHorizontal, 8)
	buttonBox.SetHAlign(gtk.AlignCenter)
	buttonBox.SetMarginTop(16)
	
	// Add save button
	saveBtn := gtk.NewButtonWithLabel("Save As...")
	saveBtn.ConnectClicked(func() {
		a.saveUpscaledImage(tmpPath, originalName)
	})
	
	// Add copy button
	copyBtn := gtk.NewButtonWithLabel("Copy to Clipboard")
	copyBtn.ConnectClicked(func() {
		a.copyImageToClipboard(texture)
		a.setStatus("Upscaled image copied to clipboard")
	})
	
	// Add close button
	closeBtn := gtk.NewButtonWithLabel("Close")
	closeBtn.ConnectClicked(func() {
		dialog.Destroy()
	})
	
	// Add buttons to button box
	buttonBox.Append(saveBtn)
	buttonBox.Append(copyBtn)
	buttonBox.Append(closeBtn)
	
	// Add button box to main box
	mainBox.Append(buttonBox)
	
	// Add main box to content area
	contentArea.Append(mainBox)
	
	// Connect response handler
	dialog.ConnectResponse(func(responseId int) {
		// Clean up temporary file
		os.Remove(tmpPath)
		dialog.Destroy()
	})
	
	// Update status
	a.setStatus("Upscaling completed successfully")
	
	// Show the dialog
	dialog.Show()
}

// saveUpscaledImage shows a file chooser dialog to save the upscaled image
func (a *App) saveUpscaledImage(sourcePath, originalName string) {
	// Create file chooser dialog
	dialog := gtk.NewFileChooserNative(
		"Save Upscaled Image",
		&a.win.Window,
		gtk.FileChooserActionSave,
		"_Save",
		"_Cancel",
	)
	
	// Set default name with "upscaled_" prefix
	baseName := fmt.Sprintf("upscaled_%s", originalName)
	dialog.SetCurrentName(baseName)
	
	// Add filters for image types
	filter := gtk.NewFileFilter()
	filter.AddPattern("*.png")
	filter.AddPattern("*.jpg")
	filter.AddPattern("*.jpeg")
	filter.AddPattern("*.webp")
	filter.SetName("Image files")
	dialog.AddFilter(filter)
	
	// Try to use the Pictures directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		picturesDir := filepath.Join(homeDir, "Pictures")
		if _, err := os.Stat(picturesDir); err == nil {
			gfile := gio.NewFileForPath(picturesDir)
			dialog.SetCurrentFolder(gfile)
		}
	}
	
	// Connect response handler
	dialog.ConnectResponse(func(response int) {
		if response == int(gtk.ResponseAccept) {
			file := dialog.File()
			if file == nil {
				a.setStatus("Error: No file selected")
				dialog.Destroy()
				return
			}
			
			destPath := file.Path()
			
			// Ensure the file has the correct extension
			if !strings.HasSuffix(strings.ToLower(destPath), ".png") &&
			   !strings.HasSuffix(strings.ToLower(destPath), ".jpg") &&
			   !strings.HasSuffix(strings.ToLower(destPath), ".jpeg") &&
			   !strings.HasSuffix(strings.ToLower(destPath), ".webp") {
				destPath += ".png"
			}
			
			// Copy the file
			go func() {
				err := copyFile(sourcePath, destPath)
				glib.IdleAdd(func() {
					if err != nil {
						a.setStatus(fmt.Sprintf("Error saving upscaled image: %v", err))
					} else {
						a.setStatus(fmt.Sprintf("Upscaled image saved to: %s", destPath))
					}
				})
			}()
		}
		
		dialog.Destroy()
	})
	
	dialog.Show()
}

// isImageFile checks if the file is a supported image format
func isImageFile(filePath string) bool {
	// Check file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	supportedExts := map[string]bool{
		".png":  true,
		".jpg":  true,
		".jpeg": true,
		".webp": true,
	}
	
	return supportedExts[ext]
}

// loadTextureFromFile loads a GDK texture from a file path
func loadTextureFromFile(path string) (*gdk.Texture, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	
	// Create texture
	texture, err := gdk.NewTextureFromBytes(glib.NewBytesWithGo(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create texture: %w", err)
	}
	
	return texture, nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	// Create parent directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Open source file
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer in.Close()

	// Create a temp file in the destination directory
	tmpFile, err := os.CreateTemp(filepath.Dir(dst), "*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Cleanup in case of error
	defer func() {
		tmpFile.Close()
		os.Remove(tmpPath)
	}()

	// Copy the data
	if _, err := io.Copy(tmpFile, in); err != nil {
		return fmt.Errorf("failed to copy data: %w", err)
	}

	// Close the file to ensure all data is written
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	// Rename the temp file to the destination path
	if err := os.Rename(tmpPath, dst); err != nil {
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return nil
}