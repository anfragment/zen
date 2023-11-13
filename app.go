package main

import (
	"context"
	"log"

	"github.com/anfragment/zen/certmanager"
	"github.com/anfragment/zen/filter"
	"github.com/anfragment/zen/proxy"
)

// App struct
type App struct {
	ctx   context.Context
	proxy *proxy.Proxy
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) shutdown(ctx context.Context) {
	if a.proxy != nil {
		a.proxy.Stop()
	}
}

// StartProxy initializes the associated resources and starts the proxy
func (a *App) StartProxy() string {
	if a.proxy == nil {
		filter := filter.NewFilter()
		certmanager, err := certmanager.NewCertManager()
		if err != nil {
			log.Printf("failed to initialize certmanager: %v", err)
			return err.Error()
		}
		a.proxy = proxy.NewProxy(filter, certmanager, a.ctx)
	}

	log.Println("starting proxy")
	if err := a.proxy.Start(); err != nil {
		log.Printf("failed to start proxy: %v", err)
		return err.Error()
	}
	return ""
}

// StopProxy stops the proxy
func (a *App) StopProxy() string {
	if a.proxy == nil {
		return "proxy not started"
	}

	log.Println("stopping proxy")
	if err := a.proxy.Stop(); err != nil {
		log.Printf("failed to stop proxy: %v", err)
		return err.Error()
	}
	return ""
}
