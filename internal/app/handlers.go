package app

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func (a *App) onGenerateClicked() {
	prompt := a.entry.Text()
	if prompt == "" {
		a.setStatus("Please enter a prompt")
		return
	}

	a.spinner.Start()
	a.clearImages()

	go func() {
		images, err := a.client.GenerateImages(prompt)
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

func (a *App) displayImages(urls []string) {
	for _, url := range urls {
		imageFrame := gtk.NewFrame("")
		imageBox := gtk.NewBox(gtk.OrientationVertical, 5)

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

				buttonBox := gtk.NewBox(gtk.OrientationHorizontal, 5)
				buttonBox.SetHAlign(gtk.AlignCenter)

				saveBtn := gtk.NewButtonWithLabel("Save")
				saveBtn.ConnectClicked(func() {
					a.saveImage(url)
				})

				copyBtn := gtk.NewButtonWithLabel("Copy")
				copyBtn.ConnectClicked(func() {
					a.copyImageToClipboard(texture)
				})

				buttonBox.Append(saveBtn)
				buttonBox.Append(copyBtn)

				imageBox.Append(picture)
				imageBox.Append(buttonBox)
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
