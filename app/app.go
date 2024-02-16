package app

import (
	"context"
	"log"
	"sync"

	"github.com/anfragment/zen/certgen"
	"github.com/anfragment/zen/certstore"
	"github.com/anfragment/zen/config"
	"github.com/anfragment/zen/filter"
	"github.com/anfragment/zen/proxy"
	"github.com/anfragment/zen/ruletree"
)

// App struct
type App struct {
	ctx           context.Context
	eventsHandler *eventsHandler
	proxy         *proxy.Proxy
	// proxyMu ensures that proxy is only started or stopped once at a time.
	proxyMu   sync.Mutex
	certStore *certstore.DiskCertStore
}

// NewApp initializes the app.
func NewApp() *App {
	return &App{}
}

// Startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.eventsHandler = newEventsHandler(a.ctx)
	a.certStore = certstore.NewDiskCertStore()
}

func (a *App) Shutdown(ctx context.Context) {
	a.proxyMu.Lock()
	defer a.proxyMu.Unlock()

	if a.proxy != nil {
		if err := a.proxy.Stop(); err != nil {
			log.Printf("failed to stop proxy: %v", err)
		}
	}
}

func (a *App) DomReady(ctx context.Context) {
	config.RunMigrations()
	config.SelfUpdate(ctx)
}

// StartProxy starts the proxy.
func (a *App) StartProxy() error {
	a.proxyMu.Lock()
	defer a.proxyMu.Unlock()

	log.Println("starting proxy")

	ruleMatcher := ruletree.NewRuleTree()
	exceptionRuleMatcher := ruletree.NewRuleTree()

	filter, err := filter.NewFilter(ruleMatcher, exceptionRuleMatcher, a.eventsHandler)
	if err != nil {
		log.Fatalf("failed to create filter: %v", err)
	}

	certGenerator, err := certgen.NewCertGenerator(a.certStore)
	if err != nil {
		log.Fatalf("failed to create cert manager: %v", err)
	}

	a.proxy, err = proxy.NewProxy(filter, certGenerator)
	if err != nil {
		log.Fatalf("failed to create proxy: %v", err)
	}

	if err := a.certStore.Init(); err != nil {
		log.Printf("failed to initialize cert store: %v", err)
		return err
	}

	if err := a.proxy.Start(); err != nil {
		log.Printf("failed to start proxy: %v", err)
		return err
	}

	return nil
}

// StopProxy stops the proxy.
func (a *App) StopProxy() error {
	a.proxyMu.Lock()
	defer a.proxyMu.Unlock()

	log.Println("stopping proxy")

	if a.proxy != nil {
		if err := a.proxy.Stop(); err != nil {
			log.Printf("failed to stop proxy: %v", err)
			return err
		}

		a.proxy = nil
	}

	return nil
}

// UninstallCA uninstalls the CA.
func (a *App) UninstallCA() error {
	if err := a.certStore.UninstallCA(); err != nil {
		log.Printf("failed to uninstall CA: %v", err)
		return err
	}

	return nil
}
