package main

import (
	"embed"
	"fmt"
	"runtime"

	"github.com/anfragment/zen/certmanager"
	"github.com/anfragment/zen/config"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
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
		OnDomReady: app.domReady,
		Bind: []interface{}{
			app,
			&config.Config,
			certmanager.GetCertManager(),
		},
		Mac: &mac.Options{
			About: &mac.AboutInfo{
				Title:   "Zen",
				Message: fmt.Sprintf("Your Comprehensive Ad-Blocker and Privacy Guard\nVersion: %s\n© 2023 Ansar Smagulov", config.Version),
			},
		},
		HideWindowOnClose: runtime.GOOS == "darwin",
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
