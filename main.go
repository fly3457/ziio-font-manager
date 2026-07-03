package main

import (
	"embed"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	wailslogger "github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app, err := NewApp()
	if err != nil {
		println("Startup error:", err.Error())
		return
	}

	// Create application with options
	err = wails.Run(&options.App{
		Title:              "Ziio Font Manager",
		Width:              1360,
		Height:             860,
		MinWidth:           1100,
		MinHeight:          700,
		Logger:             wailslogger.NewFileLogger(filepath.Join(app.Dirs.LogDir, "app.log")),
		LogLevel:           wailslogger.INFO,
		LogLevelProduction: wailslogger.INFO,
		AssetServer: &assetserver.Options{
			Assets:  assets,
			Handler: app.previewFontHandler(),
		},
		BackgroundColour: &options.RGBA{R: 246, G: 248, B: 251, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
			app.LibraryService,
			app.FontService,
			app.InstallService,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
