package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type FluxInput struct {
	Prompt             string `json:"prompt"`
	Seed               *int   `json:"seed,omitempty"`
	NumOutputs         int    `json:"num_outputs"`
	AspectRatio        string `json:"aspect_ratio"`
	OutputFormat       string `json:"output_format"`
	OutputQuality      int    `json:"output_quality"`
	DisableSafetyCheck bool   `json:"disable_safety_checker"`
}

type App struct {
	*gtk.Application
	win       *gtk.ApplicationWindow
	entry     *gtk.Entry
	spinner   *gtk.Spinner
	imageBox  *gtk.Box
	statusBar *gtk.Label
}

func main() {
	app := &App{
		Application: gtk.NewApplication("com.example.flux", gio.ApplicationFlagsNone),
	}
	app.Application.ConnectActivate(app.setupUI)

	if code := app.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}

func (a *App) setupUI() {
	// Create main window
	a.win = gtk.NewApplicationWindow(a.Application)
	a.win.SetTitle("Flux Image Generator")
	a.win.SetDefaultSize(800, 600)

	// Main layout
	mainBox := gtk.NewBox(gtk.OrientationVertical, 10)
	mainBox.SetMarginTop(10)
	mainBox.SetMarginBottom(10)
	mainBox.SetMarginStart(10)
	mainBox.SetMarginEnd(10)

	// Input area
	inputBox := gtk.NewBox(gtk.OrientationHorizontal, 5)
	a.entry = gtk.NewEntry()
	a.entry.SetPlaceholderText("Enter your prompt...")
	a.entry.SetHExpand(true)

	generateBtn := gtk.NewButtonWithLabel("Generate")
	generateBtn.ConnectClicked(a.onGenerateClicked)

	a.spinner = gtk.NewSpinner()

	inputBox.Append(a.entry)
	inputBox.Append(generateBtn)
	inputBox.Append(a.spinner)

	// Scrollable image area
	scrollWin := gtk.NewScrolledWindow()
	a.imageBox = gtk.NewBox(gtk.OrientationHorizontal, 10)
	scrollWin.SetChild(a.imageBox)
	scrollWin.SetVExpand(true)

	// Status bar
	a.statusBar = gtk.NewLabel("")
	a.statusBar.SetXAlign(0) // Left align

	mainBox.Append(inputBox)
	mainBox.Append(scrollWin)
	mainBox.Append(a.statusBar)

	a.win.SetChild(mainBox)
	a.win.Show()
}

func (a *App) onGenerateClicked() {
	prompt := a.entry.Text()
	if prompt == "" {
		a.setStatus("Please enter a prompt")
		return
	}

	a.spinner.Start()
	a.clearImages()

	go func() {
		images, err := a.generateImages(prompt)
		glib.IdleAdd(func() {
			a.spinner.Stop()
			if err != nil {
				a.setStatus(fmt.Sprintf("Error: %v", err))
				return
			}
			a.displayImages(images)
		})
	}()
}

func (a *App) clearImages() {
	for child := a.imageBox.FirstChild(); child != nil; child = a.imageBox.FirstChild() {
		a.imageBox.Remove(child)
	}
}

func (a *App) generateImages(prompt string) ([]string, error) {
	input := FluxInput{
		Prompt:             prompt,
		NumOutputs:         4,
		AspectRatio:        "1:1",
		OutputFormat:       "png",
		OutputQuality:      1,
		DisableSafetyCheck: false,
	}

	payload := map[string]interface{}{"input": input}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(
		"https://black-forest-flux.guitaripod.workers.dev/",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var urls []string
	if err := json.NewDecoder(resp.Body).Decode(&urls); err != nil {
		return nil, err
	}

	return urls, nil
}

func (a *App) displayImages(urls []string) {
	for _, url := range urls {
		imageFrame := gtk.NewFrame("")
		imageBox := gtk.NewBox(gtk.OrientationVertical, 5)

		// Download and display image
		go func(url string) {
			texture, err := a.loadImageTexture(url)
			if err != nil {
				glib.IdleAdd(func() {
					a.setStatus(fmt.Sprintf("Error loading image: %v", err))
				})
				return
			}

			glib.IdleAdd(func() {
				picture := gtk.NewPicture()
				picture.SetPaintable(texture)
				picture.SetCanShrink(true)
				picture.SetHExpand(true)
				picture.SetVExpand(true)
				picture.SetContentFit(gtk.ContentFitContain)

				saveBtn := gtk.NewButtonWithLabel("Save")
				saveBtn.ConnectClicked(func() {
					// TODO: Implement save functionality
				})

				imageBox.Append(picture)
				imageBox.Append(saveBtn)
				imageFrame.SetChild(imageBox)
				a.imageBox.Append(imageFrame)
			})
		}(url)
	}
}

func (a *App) loadImageTexture(url string) (*gdk.Texture, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the image data
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// Create texture from bytes
	texture, err := gdk.NewTextureFromBytes(glib.NewBytesWithGo(data))
	if err != nil {
		return nil, err
	}

	return texture, nil
}

func (a *App) setStatus(message string) {
	a.statusBar.SetText(message)
}
