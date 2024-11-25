package app

import (
	"fluxxxer/internal/flux"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type App struct {
	*gtk.Application
	win       *gtk.ApplicationWindow
	entry     *gtk.Entry
	spinner   *gtk.Spinner
	imageBox  *gtk.Box
	statusBar *gtk.Label
	client    *flux.Client
}

func New() *App {
	app := &App{
		Application: gtk.NewApplication("com.example.flux", gio.ApplicationFlagsNone),
		client:      flux.NewClient(),
	}
	app.Application.ConnectActivate(app.setupUI)
	return app
}

func (a *App) setStatus(message string) {
	a.statusBar.SetText(message)
}
