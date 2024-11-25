package app

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func (a *App) setupUI() {
	a.win = gtk.NewApplicationWindow(a.Application)
	a.win.SetTitle("Fluxxxer")
	a.win.SetDefaultSize(2000, 600)

	mainBox := gtk.NewBox(gtk.OrientationVertical, 10)
	mainBox.SetMarginTop(10)
	mainBox.SetMarginBottom(10)
	mainBox.SetMarginStart(10)
	mainBox.SetMarginEnd(10)

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

	scrollWin := gtk.NewScrolledWindow()
	a.imageBox = gtk.NewBox(gtk.OrientationHorizontal, 10)
	scrollWin.SetChild(a.imageBox)
	scrollWin.SetVExpand(true)

	a.statusBar = gtk.NewLabel("")
	a.statusBar.SetXAlign(0)

	mainBox.Append(inputBox)
	mainBox.Append(scrollWin)
	mainBox.Append(a.statusBar)

	a.win.SetChild(mainBox)
	a.win.Show()
}

func (a *App) clearImages() {
	for child := a.imageBox.FirstChild(); child != nil; child = a.imageBox.FirstChild() {
		a.imageBox.Remove(child)
	}
}
