package main

import (
	"embed"
	"fmt"
	"log"
	"runtime"

	"github.com/anfragment/zen/internal/app"
	"github.com/anfragment/zen/internal/cfg"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	config, err := cfg.NewConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	app, err := app.NewApp(config)
	if err != nil {
		log.Fatalf("failed to create app: %v", err)
	}

	err = wails.Run(&options.App{
		Title:     "Zen",
		MinWidth:  385,
		MaxWidth:  385,
		MinHeight: 650,
		MaxHeight: 650,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.Startup,
		OnShutdown: app.Shutdown,
		OnDomReady: app.DomReady,
		Bind: []interface{}{
			app,
			config,
		},
		Mac: &mac.Options{
			About: &mac.AboutInfo{
				Title:   "Zen",
				Message: fmt.Sprintf("Your Comprehensive Ad-Blocker and Privacy Guard\nVersion: %s\n© 2024 Ansar Smagulov", cfg.Version),
			},
		},
		HideWindowOnClose: runtime.GOOS == "darwin", // only macOS keeps closed windows in taskbar
	})

	if err != nil {
		log.Fatal(err)
	}
}
