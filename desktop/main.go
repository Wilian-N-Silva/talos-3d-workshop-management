package main

import (
	"embed"

	"github.com/Wilian-N-Silva/talos-3d-workshop-management/desktop/internal/desktopapp"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app, err := desktopapp.New()
	if err != nil {
		println("desktop startup failed:", err.Error())
		return
	}

	err = wails.Run(&options.App{
		Title:  "Gestão de Oficina 3D",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.Startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("desktop startup failed:", err.Error())
	}
}
