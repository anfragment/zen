package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"

	"github.com/anfragment/zen/matcher"
	"github.com/elazarl/goproxy"
)

type Proxy struct {
	matcher *matcher.Matcher
}

func NewProxy(matcher *matcher.Matcher) *Proxy {
	return &Proxy{matcher}
}

// ConfigureTLS configures the proxy to use the given certificate and key for TLS connections.
func (p *Proxy) ConfigureTLS(certData, keyData []byte) error {
	goproxyCa, err := tls.X509KeyPair(certData, keyData)
	if err != nil {
		return fmt.Errorf("failed to parse CA certificate and key: %v", err)
	}
	if goproxyCa.Leaf, err = x509.ParseCertificate(goproxyCa.Certificate[0]); err != nil {
		return fmt.Errorf("failed to parse CA certificate: %v", err)
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
func (p *Proxy) Start(addr string) error {
	proxy := goproxy.NewProxyHttpServer()
	// TODO: implement exclusions
	// https://github.com/AdguardTeam/HttpsExclusions
	proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)
	proxy.OnRequest().DoFunc(p.matcher.Middleware)
	return http.ListenAndServe(addr, proxy)
}
