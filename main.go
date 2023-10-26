package main

import (
	"embed"
	"runtime"

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
		MinWidth:  385,
		MaxWidth:  385,
		MinHeight: 650,
		MaxHeight: 650,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind: []interface{}{
			app,
			&config.Config,
		},
		// As the app doesn't yet have a tray icon, we want to hide it on close.
		// On Windows, setting this to true will cause the taskbar icon to dissapear,
		// but the app will still be running in the background with no apparent ways
		// to get it back. So we only do this on non-Windows platforms.
		HideWindowOnClose: runtime.GOOS != "windows",
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
