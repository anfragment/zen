package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/anfragment/zen/internal/certgen"
	"github.com/anfragment/zen/internal/certstore"
	"github.com/anfragment/zen/internal/cfg"
	"github.com/anfragment/zen/internal/filter"
	"github.com/anfragment/zen/internal/logger"
	"github.com/anfragment/zen/internal/proxy"
	"github.com/anfragment/zen/internal/ruletree"
	"github.com/anfragment/zen/internal/scriptlet"
	"github.com/anfragment/zen/internal/scriptlet/triestore"
	"github.com/anfragment/zen/internal/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context
	// name is the name of the application.
	name            string
	startOnDomReady bool
	config          *cfg.Config
	eventsHandler   *eventsHandler
	proxy           *proxy.Proxy
	proxyOn         bool
	// proxyMu ensures that proxy is only started or stopped once at a time.
	proxyMu    sync.Mutex
	certStore  *certstore.DiskCertStore
	systrayMgr *systray.Manager
}

// NewApp initializes the app.
func NewApp(name string, config *cfg.Config, startOnDomReady bool) (*App, error) {
	if name == "" {
		return nil, errors.New("name is empty")
	}
	if config == nil {
		return nil, errors.New("config is nil")
	}

	certStore, err := certstore.NewDiskCertStore(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create cert store: %v", err)
	}

	return &App{
		name:            name,
		config:          config,
		certStore:       certStore,
		startOnDomReady: startOnDomReady,
	}, nil
}

// Startup is called when the app starts.
func (a *App) Startup(context.Context) {}

func (a *App) BeforeClose(ctx context.Context) bool {
	log.Println("shutting down")
	if err := a.StopProxy(); err != nil {
		dialog, err := runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
			Type:          runtime.QuestionDialog,
			Title:         "Quit error",
			Message:       fmt.Sprintf("We've encountered an error while shutting down the proxy: %v. Do you want to quit anyway?", err),
			Buttons:       []string{"Yes", "No"},
			DefaultButton: "Yes",
			CancelButton:  "No",
		})
		if err != nil {
			return false
		}
		return dialog != "Yes"
	}
	a.systrayMgr.Quit()
	return false
}

func (a *App) DomReady(ctx context.Context) {
	systrayMgr, err := systray.NewManager(a.name, func() {
		a.StartProxy()
	}, func() {
		a.StopProxy()
	})
	if err != nil {
		log.Fatalf("failed to initialize systray manager: %v", err)
	}

	a.ctx = ctx
	a.systrayMgr = systrayMgr
	a.eventsHandler = newEventsHandler(a.ctx)

	a.config.RunMigrations()
	a.systrayMgr.Init(a.ctx)

	cfg.SelfUpdate(a.ctx)

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
	defer func() {
		// You might see this pattern both in this file and throughout the application.
		// It is used in functions that get called by the frontend, in which case we cannot log the error at the callerp level.
		if err != nil {
			log.Printf("error starting proxy: %v", err)
		} else {
			log.Println("proxy started successfully")
		}
	}()

	a.proxyMu.Lock()
	defer a.proxyMu.Unlock()

	if a.proxyOn {
		return nil
	}

	log.Println("starting proxy")

	a.eventsHandler.OnProxyStarting()
	defer func() {
		if err != nil {
			a.eventsHandler.OnProxyStartError(err)
		} else {
			a.eventsHandler.OnProxyStarted()
		}
	}()

	ruleMatcher := ruletree.NewRuleTree()
	exceptionRuleMatcher := ruletree.NewRuleTree()

	scriptletStore := triestore.NewTrieStore()
	scriptletInjector, err := scriptlet.NewInjector(scriptletStore)
	if err != nil {
		return fmt.Errorf("create scriptlets injector: %v", err)
	}

	filter, err := filter.NewFilter(a.config, ruleMatcher, exceptionRuleMatcher, scriptletInjector, a.eventsHandler)
	if err != nil {
		return fmt.Errorf("create filter: %v", err)
	}

	certGenerator, err := certgen.NewCertGenerator(a.certStore)
	if err != nil {
		return fmt.Errorf("create cert manager: %v", err)
	}

	a.proxy, err = proxy.NewProxy(filter, certGenerator, a.config.GetPort(), a.config.GetIgnoredHosts())
	if err != nil {
		return fmt.Errorf("create proxy: %v", err)
	}

	if err := a.certStore.Init(); err != nil {
		return fmt.Errorf("initialize cert store: %v", err)
	}

	if err := a.proxy.Start(); err != nil {
		if errors.Is(err, proxy.ErrUnsupportedDesktopEnvironment) {
			a.eventsHandler.OnUnsupportedDE(err)
		} else {
			return fmt.Errorf("start proxy: %v", err)
		}
	}

	a.proxyOn = true

	a.systrayMgr.OnProxyStarted()

	return nil
}

// StopProxy stops the proxy.
func (a *App) StopProxy() (err error) {
	defer func() {
		if err != nil {
			log.Printf("error stopping proxy: %v", err)
		} else {
			log.Println("proxy stopped successfully")
		}
	}()

	a.proxyMu.Lock()
	defer a.proxyMu.Unlock()

	log.Println("stopping proxy")

	a.eventsHandler.OnProxyStopping()
	defer func() {
		if err != nil {
			a.eventsHandler.OnProxyStopError(err)
		} else {
			a.eventsHandler.OnProxyStopped()
		}
	}()

	if !a.proxyOn {
		return nil
	}

	if err := a.proxy.Stop(); err != nil {
		return fmt.Errorf("stop proxy: %w", err)
	}
	a.proxy = nil
	a.proxyOn = false

	a.systrayMgr.OnProxyStopped()

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

func (a *App) OpenLogsDirectory() error {
	if err := logger.OpenLogsDirectory(); err != nil {
		log.Printf("failed to open logs directory: %v", err)
		return err
	}

	return nil
}

// ExportCustomFilterListsToFile exports the custom filter lists to a file.
func (a *App) ExportCustomFilterLists() error {

	if a.ctx == nil {
		return errors.New("App DOM is not ready")
	}

	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Export Custom Filter Lists",
		DefaultFilename: "filter-lists.json",
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON", Pattern: "*.json"},
		},
	})

	if err != nil {
		log.Printf("failed to open file dialog: %v", err)
		return err
	}

	if filePath == "" {
		return errors.New("no file selected")
	}

	customFilterLists := a.config.GetTargetTypeFilterLists(cfg.FilterListTypeCustom)

	if len(customFilterLists) == 0 {
		return errors.New("no custom filter lists to export")
	}

	data, err := json.MarshalIndent(customFilterLists, "", "  ")
	if err != nil {
		log.Printf("failed to marshal filter lists: %v", err)
		return err
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		log.Printf("failed to write filter lists to file: %v", err)
		return err
	}

	return nil
}

// ImportCustomFilterLists imports the custom filter lists from a file.
func (a *App) ImportCustomFilterLists() error {
	if a.ctx == nil {
		return errors.New("App DOM is not ready")
	}

	filePath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Import Custom Filter Lists",
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON", Pattern: "*.json"},
		},
	})

	if err != nil {
		return err
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("failed to read filter lists file: %v", err)
		return err
	}

	var filterLists []cfg.FilterList
	if err := json.Unmarshal(data, &filterLists); err != nil {
		log.Printf("failed to unmarshal filter lists: %v", err)
		return errors.New("incorrect filter lists format")
	}

	if len(filterLists) == 0 {
		return errors.New("no custom filter lists to import")
	}

	if err := a.config.AddFilterLists(filterLists); err != nil {
		log.Printf("failed to add filter lists: %v", err)
		return err
	}

	return nil
}
