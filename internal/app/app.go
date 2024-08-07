package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/anfragment/zen/internal/certgen"
	"github.com/anfragment/zen/internal/certstore"
	"github.com/anfragment/zen/internal/cfg"
	"github.com/anfragment/zen/internal/filter"
	"github.com/anfragment/zen/internal/proxy"
	"github.com/anfragment/zen/internal/ruletree"
)

type App struct {
	ctx             context.Context
	startOnDomReady bool
	config          *cfg.Config
	eventsHandler   *eventsHandler
	proxy           *proxy.Proxy
	proxyOn         bool
	// proxyMu ensures that proxy is only started or stopped once at a time.
	proxyMu   sync.Mutex
	certStore *certstore.DiskCertStore
}

// NewApp initializes the app.
func NewApp(config *cfg.Config, startOnDomReady bool) (*App, error) {
	if config == nil {
		return nil, errors.New("config is nil")
	}
	certStore, err := certstore.NewDiskCertStore(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create cert store: %v", err)
	}

	return &App{
		config:          config,
		certStore:       certStore,
		startOnDomReady: startOnDomReady,
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
	time.AfterFunc(time.Second, func() {
		// This is a workaround for the issue where not all React components are mounted in time.
		// StartProxy requires an active event listener on the frontend to show the user the correct proxy state.
		if a.startOnDomReady {
			a.StartProxy()
		}
	})
}

// StartProxy starts the proxy.
func (a *App) StartProxy() (err error) {
	a.proxyMu.Lock()
	defer a.proxyMu.Unlock()

	if a.proxyOn {
		return nil
	}

	a.eventsHandler.OnProxyStarting()
	defer func() {
		if err != nil {
			log.Println(err)
			a.eventsHandler.OnProxyStartError(err)
		} else {
			a.eventsHandler.OnProxyStarted()
		}
	}()

	log.Println("starting proxy")

	ruleMatcher := ruletree.NewRuleTree()
	exceptionRuleMatcher := ruletree.NewRuleTree()

	filter, err := filter.NewFilter(a.config, ruleMatcher, exceptionRuleMatcher, a.eventsHandler)
	if err != nil {
		return fmt.Errorf("failed to create filter: %v", err)
	}

	certGenerator, err := certgen.NewCertGenerator(a.certStore)
	if err != nil {
		return fmt.Errorf("failed to create cert manager: %v", err)
	}

	a.proxy, err = proxy.NewProxy(filter, certGenerator, a.config.GetPort(), a.config.GetIgnoredHosts())
	if err != nil {
		return fmt.Errorf("failed to create proxy: %v", err)
	}

	if err := a.certStore.Init(); err != nil {
		return fmt.Errorf("failed to initialize cert store: %v", err)
	}

	if err := a.proxy.Start(); err != nil {
		return fmt.Errorf("failed to start proxy: %v", err)
	}

	a.proxyOn = true

	return nil
}

// StopProxy stops the proxy.
func (a *App) StopProxy() error {
	a.proxyMu.Lock()
	defer a.proxyMu.Unlock()

	if !a.proxyOn {
		return nil
	}

	log.Println("stopping proxy")

	if err := a.proxy.Stop(); err != nil {
		log.Printf("failed to stop proxy: %v", err)
		return err
	}
	a.proxy = nil
	a.proxyOn = false

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
