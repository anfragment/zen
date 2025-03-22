package sysproxy

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"text/template"
	"time"
)

var (
	ErrUnsupportedDesktopEnvironment = errors.New("system proxy configuration is currently only supported on GNOME")

	pacTemplate = template.Must(
		template.New("pac").Parse(`function FindProxyForURL(url, host) {return "PROXY 127.0.0.1:{{.ProxyPort}}";}`))
)

type Manager struct {
	pacPort int
	server  *http.Server
}

func NewManager(pacPort int) *Manager {
	return &Manager{
		pacPort: pacPort,
	}
}

func (m *Manager) Set(proxyPort int) error {
	pac := renderPac(proxyPort)
	mux := http.NewServeMux()
	mux.HandleFunc("/proxy.pac", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("serving PAC for %s", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/x-ns-proxy-autoconfig")
		w.WriteHeader(http.StatusOK)
		w.Write(pac)
	})

	m.server = &http.Server{
		Handler:      mux,
		ReadTimeout:  time.Minute,
		WriteTimeout: time.Minute,
	}
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", m.pacPort))
	if err != nil {
		return fmt.Errorf("listen: %v", err)
	}
	actualPort := listener.Addr().(*net.TCPAddr).Port
	log.Printf("PAC server listening on port %d", actualPort)

	go func() {
		if err := m.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("error serving PAC: %v", err)
		}
	}()

	pacURL := fmt.Sprintf("http://127.0.0.1:%d/proxy.pac", actualPort)
	if err := setSystemProxy(pacURL); err != nil {
		return fmt.Errorf("set system proxy with URL %q: %v", pacURL, err)
	}

	return nil
}

func (m *Manager) Clear() error {
	if m.server == nil {
		log.Println("warning: trying to clear system proxy without setting it first")
		return nil
	}

	if err := unsetSystemProxy(); err != nil {
		return fmt.Errorf("unset system proxy: %v", err)
	}

	if err := m.server.Close(); err != nil {
		return fmt.Errorf("close: %v", err)
	}
	m.server = nil
	return nil
}

func renderPac(proxyPort int) []byte {
	var buf bytes.Buffer
	pacTemplate.Execute(&buf, struct{ ProxyPort int }{ProxyPort: proxyPort})
	return buf.Bytes()
}
