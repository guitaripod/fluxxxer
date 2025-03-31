package app

import (
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	
	// Import gio with underscore to use in file methods but avoid unused import error
	_ "github.com/diamondburned/gotk4/pkg/gio/v2"
)

// setupUI initializes the application UI
func (a *App) setupUI() {
	// Create main application window
	a.win = gtk.NewApplicationWindow(a.Application)
	a.win.SetTitle("Fluxxxer")
	a.win.SetDefaultSize(a.config.GetWindowWidth(), a.config.GetWindowHeight())

	// Create main container with margins
	mainBox := gtk.NewBox(gtk.OrientationVertical, 10)
	mainBox.SetMarginTop(16)
	mainBox.SetMarginBottom(16)
	mainBox.SetMarginStart(16)
	mainBox.SetMarginEnd(16)

	// Create header area with controls
	headerBox := a.createHeaderArea()
	mainBox.Append(headerBox)

	// Create a stack to switch between generator and upscaler modes
	stack := gtk.NewStack()
	stack.SetTransitionType(gtk.StackTransitionTypeCrossfade)
	stack.SetTransitionDuration(200)
	
	// Generator view (image display area)
	generatorView := a.createGeneratorView()
	stack.AddTitled(generatorView, "generator", "Generator")
	
	// Upscaler view
	upscalerView := a.createUpscalerView()
	stack.AddTitled(upscalerView, "upscaler", "Upscaler")
	
	// Add stack to main box
	stack.SetVExpand(true)
	mainBox.Append(stack)
	
	// Create status bar
	a.statusBar = gtk.NewLabel("")
	a.statusBar.SetXAlign(0)
	a.statusBar.SetMarginTop(8)
	mainBox.Append(a.statusBar)

	// Show the window
	a.win.SetChild(mainBox)
	a.win.ConnectShow(func() {
		// Use configured window width
		a.currentWidth = a.config.GetWindowWidth()
		
		// Set initial mode
		a.setMode(a.isGeneratorMode)
		
		// Update stack based on current mode
		if a.isGeneratorMode {
			stack.SetVisibleChildName("generator")
		} else {
			stack.SetVisibleChildName("upscaler")
		}
	})
	
	// Listen for mode changes
	a.generatorToggle.ConnectToggled(func() {
		if a.generatorToggle.Active() {
			stack.SetVisibleChildName("generator")
		}
	})
	
	a.upscalerToggle.ConnectToggled(func() {
		if a.upscalerToggle.Active() {
			stack.SetVisibleChildName("upscaler")
		}
	})
	
	// Setup simple drop to handle files for the upscaler
	a.setupFileDrop(upscalerView)
	
	a.win.Show()
}

// createHeaderArea creates the top controls for the application
func (a *App) createHeaderArea() *gtk.Box {
	// Create main header container
	headerBox := gtk.NewBox(gtk.OrientationVertical, 8)
	
	// Create input area (prompt + generate button)
	inputBox := gtk.NewBox(gtk.OrientationHorizontal, 8)
	
	// Prompt entry field
	a.entry = gtk.NewEntry()
	a.entry.SetPlaceholderText("Enter your prompt...")
	a.entry.SetHExpand(true)
	a.entry.SetMarginEnd(8)
	a.entry.ConnectActivate(a.onGenerateClicked)
	
	// Generate button
	generateBtn := gtk.NewButtonWithLabel("Generate")
	// generateBtn.AddCSSClass("suggested-action") - Not available in this version
	generateBtn.ConnectClicked(a.onGenerateClicked)
	
	// Spinner for loading state
	a.spinner = gtk.NewSpinner()
	a.spinner.SetMarginStart(8)
	
	// Add elements to input box
	inputBox.Append(a.entry)
	inputBox.Append(generateBtn)
	inputBox.Append(a.spinner)
	
	// Create options area (aspect ratio, number of outputs, etc.)
	optionsBox := gtk.NewBox(gtk.OrientationHorizontal, 16)
	optionsBox.SetMarginTop(8)
	
	// Create aspect ratio dropdown
	aspectLabel := gtk.NewLabel("Aspect Ratio:")
	aspectLabel.SetMarginEnd(4)
	
	// Create and store reference to aspect ratio dropdown
	aspectRatioCombo = gtk.NewDropDown(nil, nil)
	aspectModel := gtk.NewStringList(a.config.GetSupportedAspectRatios())
	aspectRatioCombo.SetModel(aspectModel)
	
	// Set default aspect ratio
	for i, ratio := range a.config.GetSupportedAspectRatios() {
		if ratio == a.config.GetDefaultAspectRatio() {
			aspectRatioCombo.SetSelected(uint(i))
			break
		}
	}
	
	// Number of outputs slider
	numOutputsLabel := gtk.NewLabel("Images:")
	numOutputsLabel.SetMarginStart(16)
	numOutputsLabel.SetMarginEnd(4)
	
	// Create and store reference to outputs scale
	numOutputsScale = gtk.NewScale(gtk.OrientationHorizontal, gtk.NewAdjustment(
		float64(a.config.GetDefaultNumOutputs()), // value
		1,                                        // min
		8,                                        // max
		1,                                        // step
		0,                                        // page increment
		0,                                        // page size
	))
	numOutputsScale.SetDrawValue(true)
	numOutputsScale.SetHExpand(false)
	numOutputsScale.SetSizeRequest(120, -1)
	numOutputsScale.SetDigits(0)
	
	// Add options elements
	optionsBox.Append(aspectLabel)
	optionsBox.Append(aspectRatioCombo)
	optionsBox.Append(numOutputsLabel)
	optionsBox.Append(numOutputsScale)
	
	// Mode switcher section for switching between generator and upscaler
	modeBox := gtk.NewBox(gtk.OrientationHorizontal, 4)
	modeBox.SetHAlign(gtk.AlignEnd)
	modeBox.SetHExpand(true)
	
	// Store toggle buttons for later use
	a.generatorToggle = gtk.NewToggleButton()
	a.generatorToggle.SetLabel("Generator")
	a.generatorToggle.SetActive(a.isGeneratorMode)
	
	a.upscalerToggle = gtk.NewToggleButton()
	a.upscalerToggle.SetLabel("Upscaler")
	a.upscalerToggle.SetActive(!a.isGeneratorMode)
	
	// Disable upscaler button if not configured
	if !a.isUpscalerConfigured() {
		a.upscalerToggle.SetSensitive(false)
		a.upscalerToggle.SetTooltipText("Upscaler not configured. Set UPSCALER_API_URL and UPSCALER_API_KEY in your .env file.")
	}
	
	// Add toggles to mode box
	modeBox.Append(a.generatorToggle)
	modeBox.Append(a.upscalerToggle)
	
	// Connect toggle buttons to form a radio group
	a.generatorToggle.ConnectToggled(func() {
		if a.generatorToggle.Active() {
			a.upscalerToggle.SetActive(false)
			a.setMode(true)
		} else if !a.upscalerToggle.Active() {
			a.generatorToggle.SetActive(true)
		}
	})
	
	a.upscalerToggle.ConnectToggled(func() {
		if a.upscalerToggle.Active() {
			a.generatorToggle.SetActive(false)
			a.setMode(false)
		} else if !a.generatorToggle.Active() {
			a.upscalerToggle.SetActive(true)
		}
	})
	
	// Add the mode switcher to the options box
	optionsBox.Append(modeBox)
	
	// Add both rows to the header
	headerBox.Append(inputBox)
	headerBox.Append(optionsBox)
	
	return headerBox
}

// createGeneratorView creates the view for the image generator
func (a *App) createGeneratorView() *gtk.ScrolledWindow {
	scrollWin := gtk.NewScrolledWindow()
	a.imageBox = gtk.NewBox(gtk.OrientationHorizontal, 16)
	scrollWin.SetChild(a.imageBox)
	scrollWin.SetVExpand(true)
	scrollWin.SetHExpand(true)
	
	return scrollWin
}

// createUpscalerView creates the view for the image upscaler
func (a *App) createUpscalerView() *gtk.Box {
	upscalerBox := gtk.NewBox(gtk.OrientationVertical, 16)
	upscalerBox.SetHExpand(true)
	upscalerBox.SetVExpand(true)
	
	// Create a placeholder for the upscaler dropzone
	placeholderBox := gtk.NewBox(gtk.OrientationVertical, 8)
	placeholderBox.SetHAlign(gtk.AlignCenter)
	placeholderBox.SetVAlign(gtk.AlignCenter)
	placeholderBox.SetHExpand(true)
	placeholderBox.SetVExpand(true)
	
	// Add an icon
	iconTheme := gtk.IconThemeGetForDisplay(gdk.DisplayGetDefault())
	if iconTheme.HasIcon("document-save") {
		icon := gtk.NewImageFromIconName("document-save")
		icon.SetPixelSize(64)
		placeholderBox.Append(icon)
	}
	
	// Add a label
	dropLabel := gtk.NewLabel("Drag and drop an image here to upscale it")
	// dropLabel.AddCSSClass("title-2") - Not available in this version
	placeholderBox.Append(dropLabel)
	
	// Add instructions
	infoLabel := gtk.NewLabel("Or click the button below to select an image file")
	placeholderBox.Append(infoLabel)
	
	// Add a select file button
	selectBtn := gtk.NewButtonWithLabel("Select Image")
	selectBtn.SetHAlign(gtk.AlignCenter)
	selectBtn.SetMarginTop(16)
	selectBtn.ConnectClicked(func() {
		a.showFileChooserForUpscale()
	})
	placeholderBox.Append(selectBtn)
	
	// Add upscale options
	optionsFrame := gtk.NewFrame("Upscale Options")
	optionsBox := gtk.NewBox(gtk.OrientationVertical, 8)
	optionsBox.SetMarginTop(16)
	optionsBox.SetMarginBottom(16)
	optionsBox.SetMarginStart(16)
	optionsBox.SetMarginEnd(16)
	
	// Upscale type
	typeBox := gtk.NewBox(gtk.OrientationHorizontal, 8)
	typeLabel := gtk.NewLabel("Upscale Type:")
	typeLabel.SetHAlign(gtk.AlignStart)
	typeLabel.SetXAlign(0)
	
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
	
	promptEntry := gtk.NewEntry()
	promptEntry.SetPlaceholderText("Enter a prompt to guide upscaling (for conservative/creative modes)")
	promptEntry.SetHExpand(true)
	
	promptBox.Append(promptLabel)
	promptBox.Append(promptEntry)
	
	// Add options to the options box
	optionsBox.Append(typeBox)
	optionsBox.Append(promptBox)
	
	optionsFrame.SetChild(optionsBox)
	
	// Add elements to the upscaler box
	upscalerBox.Append(placeholderBox)
	upscalerBox.Append(optionsFrame)
	
	return upscalerBox
}

// setupFileDrop sets up a simple file drop handler
func (a *App) setupFileDrop(widget *gtk.Box) {
	// For now, this is a simplified version without drag-and-drop
	// We'll rely on the file picker button
}

// showFileChooserForUpscale shows a file chooser dialog for upscaling
func (a *App) showFileChooserForUpscale() {
	dialog := gtk.NewFileChooserNative(
		"Select Image to Upscale",
		&a.win.Window,
		gtk.FileChooserActionOpen,
		"_Open",
		"_Cancel",
	)
	
	// Add image filters
	filter := gtk.NewFileFilter()
	filter.AddPattern("*.png")
	filter.AddPattern("*.jpg")
	filter.AddPattern("*.jpeg")
	filter.AddPattern("*.webp")
	filter.SetName("Image files")
	dialog.AddFilter(filter)
	
	dialog.ConnectResponse(func(response int) {
		if response == int(gtk.ResponseAccept) {
			file := dialog.File()
			if file != nil {
				a.handleUpscaleFile(file.Path())
			}
		}
		dialog.Destroy()
	})
	
	dialog.Show()
}

// clearImages removes all images from the display area
func (a *App) clearImages() {
	for child := a.imageBox.FirstChild(); child != nil; child = a.imageBox.FirstChild() {
		a.imageBox.Remove(child)
	}
}