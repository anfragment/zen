package main

import (
	"embed"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/anfragment/zen/internal/app"
	"github.com/anfragment/zen/internal/autostart"
	"github.com/anfragment/zen/internal/cfg"
	"github.com/anfragment/zen/internal/files"
	"github.com/anfragment/zen/internal/logger"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

const (
	appName = "Zen"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	err := logger.SetupLogger()
	if err != nil {
		log.Printf("failed to setup logger: %v", err)
	}

	config, err := cfg.NewConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	files := files.NewFiles()

	var startOnDomReady bool
	for _, arg := range os.Args[1:] {
		if arg == "--start" {
			startOnDomReady = true
		}
	}
	app, err := app.NewApp(appName, config, files, startOnDomReady)
	if err != nil {
		log.Fatalf("failed to create app: %v", err)
	}

	autostart := &autostart.Manager{}

	err = wails.Run(&options.App{
		Title:     appName,
		MinWidth:  420,
		MaxWidth:  420,
		MinHeight: 650,
		MaxHeight: 650,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:     app.Startup,
		OnBeforeClose: app.BeforeClose,
		OnDomReady:    app.DomReady,
		Bind: []interface{}{
			app,
			config,
			autostart,
			files,
		},
		Mac: &mac.Options{
			About: &mac.AboutInfo{
				Title:   appName,
				Message: fmt.Sprintf("Your Comprehensive Ad-Blocker and Privacy Guard\nVersion: %s\n© 2024 Ansar Smagulov", cfg.Version),
			},
		},
		HideWindowOnClose: runtime.GOOS == "darwin" || runtime.GOOS == "windows", // only macOS keeps closed windows in taskbar
	})

	if err != nil {
		log.Fatal(err)
	}
}
