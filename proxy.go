package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
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

// ConfigureTLS configures the proxy to use the given certificate and key for
// TLS connections.
func (p *Proxy) ConfigureTLS(certFile, keyFile string) error {
	caCert, err := ioutil.ReadFile(certFile)
	if err != nil {
		return fmt.Errorf("failed to read CA certificate: %v", err)
	}
	caKey, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return fmt.Errorf("failed to read CA key: %v", err)
	}
	goproxyCa, err := tls.X509KeyPair(caCert, caKey)
	if err != nil {
		return fmt.Errorf("failed to parse CA certificate and key: %v", err)
	}
	if goproxyCa.Leaf, err = x509.ParseCertificate(goproxyCa.Certificate[0]); err != nil {
		return fmt.Errorf("failed to parse CA certificate: %v", err)
	}

	goproxy.GoproxyCa = goproxyCa
	goproxy.OkConnect = &goproxy.ConnectAction{Action: goproxy.ConnectAccept, TLSConfig: goproxy.TLSConfigFromCA(&goproxyCa)}
	goproxy.MitmConnect = &goproxy.ConnectAction{Action: goproxy.ConnectMitm, TLSConfig: goproxy.TLSConfigFromCA(&goproxyCa)}
	goproxy.HTTPMitmConnect = &goproxy.ConnectAction{Action: goproxy.ConnectHTTPMitm, TLSConfig: goproxy.TLSConfigFromCA(&goproxyCa)}
	goproxy.RejectConnect = &goproxy.ConnectAction{Action: goproxy.ConnectReject, TLSConfig: goproxy.TLSConfigFromCA(&goproxyCa)}
	return nil
}

// Start starts the proxy on the given address.
func (p *Proxy) Start(addr string) error {
	proxy := goproxy.NewProxyHttpServer()
	proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)
	proxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		if p.matcher.Match(req.URL.String()) {
			log.Printf("blocked %s", req.URL.String())
			return req, goproxy.NewResponse(req, goproxy.ContentTypeText, http.StatusForbidden, "blocked by zen")
		}
		log.Printf("allowed %s", req.URL.String())
		return req, nil
	})
	return http.ListenAndServe(addr, proxy)
}
