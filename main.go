package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"net/http"
	_ "net/http/pprof"

	"github.com/anfragment/zen/certmanager"
	"github.com/anfragment/zen/filter"
	"github.com/anfragment/zen/proxy"
)

func main() {
	filter := filter.NewFilter()

	certmanager, err := certmanager.NewCertManager()
	if err != nil {
		log.Fatalf("failed to initialize certmanager: %v", err)
	}

	proxy := proxy.NewProxy("127.0.0.1", 8080, filter, certmanager)
	log.Println("starting proxy")
	if err := proxy.Start(); err != nil {
		log.Fatalf("failed to start proxy: %v", err)
	}

	go func() {
		http.ListenAndServe(":6060", nil)
	}()

	// Wait for SIGINT or SIGTERM.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	if err := proxy.Stop(); err != nil {
		log.Fatalf("failed to stop proxy: %v", err)
	}
	log.Println("proxy stopped")
}
