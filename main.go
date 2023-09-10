package main

import (
	"embed"

	"github.com/anfragment/zen/config"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "Zen",
		MaxWidth:  356,
		MaxHeight: 600,
		MinWidth:  356,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind: []interface{}{
			app,
			&config.Config,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
