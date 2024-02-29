package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/anfragment/zen/certgen"
	"github.com/anfragment/zen/certstore"
	"github.com/anfragment/zen/cfg"
	"github.com/anfragment/zen/filter"
	"github.com/anfragment/zen/proxy"
	"github.com/anfragment/zen/ruletree"
)

type App struct {
	ctx           context.Context
	config        *cfg.Config
	eventsHandler *eventsHandler
	proxy         *proxy.Proxy
	// proxyMu ensures that proxy is only started or stopped once at a time.
	proxyMu   sync.Mutex
	certStore *certstore.DiskCertStore
}

// NewApp initializes the app.
func NewApp(config *cfg.Config) (*App, error) {
	if config == nil {
		return nil, errors.New("config is nil")
	}
	certStore, err := certstore.NewDiskCertStore(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create cert store: %v", err)
	}

	return &App{
		config:    config,
		certStore: certStore,
	}, nil
}

// Startup is called when the app starts.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.eventsHandler = newEventsHandler(a.ctx)
}

func (a *App) Shutdown(context.Context) {
	a.proxyMu.Lock()
	defer a.proxyMu.Unlock()

	if a.proxy != nil {
		if err := a.proxy.Stop(); err != nil {
			log.Printf("failed to stop proxy: %v", err)
		}
	}
}

func (a *App) DomReady(ctx context.Context) {
	a.config.RunMigrations()
	cfg.SelfUpdate(ctx)
}

// StartProxy starts the proxy.
func (a *App) StartProxy() error {
	a.proxyMu.Lock()
	defer a.proxyMu.Unlock()

	log.Println("starting proxy")

	ruleMatcher := ruletree.NewRuleTree()
	exceptionRuleMatcher := ruletree.NewRuleTree()

	filter, err := filter.NewFilter(a.config, ruleMatcher, exceptionRuleMatcher, a.eventsHandler)
	if err != nil {
		err = fmt.Errorf("failed to create filter: %v", err)
		log.Println(err)
		return err
	}

	certGenerator, err := certgen.NewCertGenerator(a.certStore)
	if err != nil {
		err = fmt.Errorf("failed to create cert manager: %v", err)
		log.Println(err)
		return err
	}

	a.proxy, err = proxy.NewProxy(filter, certGenerator, a.config.GetPort())
	if err != nil {
		err = fmt.Errorf("failed to create proxy: %v", err)
		log.Println(err)
		return err
	}

	if err := a.certStore.Init(); err != nil {
		err = fmt.Errorf("failed to initialize cert store: %v", err)
		log.Println(err)
		return err
	}

	if err := a.proxy.Start(); err != nil {
		err = fmt.Errorf("failed to start proxy: %v", err)
		log.Println(err)
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
