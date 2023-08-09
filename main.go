package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/anfragment/zen/certmanager"
	"github.com/anfragment/zen/config"
	"github.com/anfragment/zen/matcher"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:9999", "proxy address")
	flag.Parse()

	matcher := matcher.NewMatcher()
	for _, filter := range config.Config.Filter.FilterLists {
		file, err := http.Get(filter)
		if err != nil {
			log.Fatalf("failed to get filter %s: %v", filter, err)
		}
		defer file.Body.Close()
		matcher.AddRules(file.Body)
	}

	certmanager, err := certmanager.NewCertManager()
	if err != nil {
		log.Fatalf("failed to initialize certmanager: %v", err)
	}

	proxy := NewProxy(matcher)
	err = proxy.ConfigureTLS(certmanager.CertData, certmanager.KeyData)
	if err != nil {
		log.Fatalf("failed to configure TLS: %v", err)
	}
	log.Printf("starting proxy on %s", *addr)
	err = proxy.Start(*addr)
	if err != nil {
		log.Fatalf("failed to start proxy: %v", err)
	}
}
