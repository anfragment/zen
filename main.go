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
		/*
			As the app doesn't yet have a tray icon, the correct behaviour is to hide the window on close.
			However, on Windows, setting this to true causes the taskbar icon to disappear without any apparent way to restore the window.
			Therefore, we only set this to true on non-Windows platforms.
			This is supposed to be a temporary workaround, since Wails 3.0 with tray support is coming soon.
		*/
		HideWindowOnClose: runtime.GOOS != "windows",
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
