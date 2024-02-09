package app

import (
	"context"
	"log"

	"github.com/anfragment/zen/config"
	"github.com/anfragment/zen/filter"
	"github.com/anfragment/zen/proxy"
	"github.com/anfragment/zen/ruletree"
)

// App struct
type App struct {
	ctx   context.Context
	proxy *proxy.Proxy
}

// NewApp initializes the app.
func NewApp() *App {
	return &App{}
}

// Startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	eventsHandler := newEventsHandler(a.ctx)
	ruleMatcher := ruletree.NewRuleTree()
	exceptionRuleMatcher := ruletree.NewRuleTree()
	filter, err := filter.NewFilter(ruleMatcher, exceptionRuleMatcher, eventsHandler)
	if err != nil {
		log.Fatalf("failed to create filter: %v", err)
	}

	a.proxy, err = proxy.NewProxy(filter)
	if err != nil {
		log.Fatalf("failed to create proxy: %v", err)
	}
}

func (a *App) Shutdown(ctx context.Context) {
	if err := a.proxy.Stop(false); err != nil {
		log.Printf("failed to stop proxy: %v", err)
	}
}

func (a *App) DomReady(ctx context.Context) {
	config.RunMigrations()
	config.SelfUpdate(ctx)
}

// StartProxy starts the proxy.
func (a *App) StartProxy() error {
	log.Println("starting proxy")

	if err := a.proxy.Start(); err != nil {
		log.Printf("failed to start proxy: %v", err)
		return err
	}

	return nil
}

// StopProxy stops the proxy.
func (a *App) StopProxy() error {
	log.Println("stopping proxy")

	if err := a.proxy.Stop(true); err != nil {
		log.Printf("failed to stop proxy: %v", err)
		return err
	}

	return nil
}
