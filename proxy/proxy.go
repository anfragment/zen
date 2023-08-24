package proxy

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/anfragment/zen/matcher"
	"github.com/elazarl/goproxy"
)

type Proxy struct {
	host    string
	port    int
	matcher *matcher.Matcher
}

func NewProxy(host string, port int, matcher *matcher.Matcher) *Proxy {
	return &Proxy{host, port, matcher}
}

// ConfigureTLS configures the proxy to use the given certificate and key for TLS connections.
func (p *Proxy) ConfigureTLS(certData, keyData []byte) error {
	goproxyCa, err := tls.X509KeyPair(certData, keyData)
	if err != nil {
		return fmt.Errorf("parse certificate: %v", err)
	}
	if goproxyCa.Leaf, err = x509.ParseCertificate(goproxyCa.Certificate[0]); err != nil {
		return fmt.Errorf("parse leaf certificate: %v", err)
	}

	tlsConfig := goproxy.TLSConfigFromCA(&goproxyCa)
	goproxy.GoproxyCa = goproxyCa
	goproxy.OkConnect = &goproxy.ConnectAction{Action: goproxy.ConnectAccept, TLSConfig: tlsConfig}
	goproxy.MitmConnect = &goproxy.ConnectAction{Action: goproxy.ConnectMitm, TLSConfig: tlsConfig}
	goproxy.HTTPMitmConnect = &goproxy.ConnectAction{Action: goproxy.ConnectHTTPMitm, TLSConfig: tlsConfig}
	goproxy.RejectConnect = &goproxy.ConnectAction{Action: goproxy.ConnectReject, TLSConfig: tlsConfig}

	return nil
}

// Start starts the proxy on the given address.
func (p *Proxy) Start() error {
	proxy := goproxy.NewProxyHttpServer()
	// TODO: implement exclusions
	// https://github.com/AdguardTeam/HttpsExclusions
	proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)
	proxy.OnRequest().DoFunc(p.matcher.Middleware)
	errC := make(chan error, 1)
	go func() {
		errC <- http.ListenAndServe(fmt.Sprintf("%s:%d", p.host, p.port), proxy)
	}()

	if err := p.setSystemProxy(); err != nil {
		return fmt.Errorf("set system proxy: %v", err)
	}
	defer func() {
		log.Println("unsetting system proxy")
		if err := p.unsetSystemProxy(); err != nil {
			log.Printf("failed to unset system proxy: %v", err)
		} else {
			log.Println("system proxy unset")
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	select {
	case err := <-errC:
		return err
	case <-signals:
		return nil
	}
}
