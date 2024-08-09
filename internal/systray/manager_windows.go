package systray

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed logo.ico
var logoFS embed.FS

type Manager struct {
	logoBytes         []byte
	appName           string
	proxyStateMu      sync.Mutex
	proxyActive       bool
	proxyStart        func()
	proxyStop         func()
	startStopMenuItem *menuItem
}

func NewManager(appName string, proxyStart func(), proxyStop func()) (*Manager, error) {
	if appName == "" {
		return nil, errors.New("appName is empty")
	}
	if proxyStart == nil {
		return nil, errors.New("proxyStart is nil")
	}
	if proxyStop == nil {
		return nil, errors.New("proxyStop is nil")
	}

	logoBytes, err := logoFS.ReadFile("logo.ico")
	if err != nil {
		return nil, fmt.Errorf("read logo from embed: %w", err)
	}

	return &Manager{
		logoBytes:  logoBytes,
		proxyStart: proxyStart,
		proxyStop:  proxyStop,
		appName:    appName,
	}, nil
}

func (m *Manager) Init(ctx context.Context) error {
	go func() {
		run(m.onReady(ctx), nil)
	}()

	return nil
}

// Quit needs to be called on application quit.
func (m *Manager) Quit() {
	quit()
}

// OnProxyStarted should be called when the proxy gets started.
func (m *Manager) OnProxyStarted() {
	m.proxyStateMu.Lock()
	defer m.proxyStateMu.Unlock()
	m.proxyActive = true

	if m.startStopMenuItem == nil {
		// Sanity check.
		log.Println("startStopMenuItem is nil")
		return
	}

	m.startStopMenuItem.SetTitle("Stop")
	m.startStopMenuItem.SetTooltip("Stop")
}

// OnProxyStopped should be called when the proxy gets stopped.
func (m *Manager) OnProxyStopped() {
	m.proxyStateMu.Lock()
	defer m.proxyStateMu.Unlock()
	m.proxyActive = false

	if m.startStopMenuItem == nil {
		// Sanity check.
		log.Println("startStopMenuItem is nil")
		return
	}

	m.startStopMenuItem.SetTitle("Start")
	m.startStopMenuItem.SetTooltip("Start")
}

func (m *Manager) onReady(ctx context.Context) func() {
	return func() {
		setIcon(m.logoBytes)
		setTooltip(m.appName)

		openMenuItem := addMenuItem("Open", "Open the application window")
		go func() {
			for range openMenuItem.ClickedCh {
				runtime.Show(ctx)
			}
		}()

		m.startStopMenuItem = addMenuItem("Start", "Start")
		go func() {
			for range m.startStopMenuItem.ClickedCh {
				m.proxyStateMu.Lock()
				active := m.proxyActive
				m.proxyStateMu.Unlock()
				if active {
					m.proxyStop()
				} else {
					m.proxyStart()
				}
			}
		}()

		addSeparator()

		quitMenuItem := addMenuItem("Quit", "Quit the application")
		go func() {
			for range quitMenuItem.ClickedCh {
				runtime.Quit(ctx)
			}
		}()
	}
}
