// https://github.com/eliben/code-for-blog/blob/master/2022/go-and-proxies/connect-mitm-proxy.go
package main

import (
	"flag"
	"log"
	"net/http"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:9999", "proxy address")
	caCertFile := flag.String("cacertfile", "", "certificate .pem file for trusted CA")
	caKeyFile := flag.String("cakeyfile", "", "key .pem file for trusted CA")
	flag.Parse()

	filter := NewFilter()
	if err := filter.AddRemoteFilters([]string{
		"https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts",
		"https://easylist.to/easylist/easylist.txt",
	}); err != nil {
		log.Fatalf("error adding remote filters: %v", err)
	}

	certManager, err := NewCertManager(*caCertFile, *caKeyFile)
	if err != nil {
		log.Fatalf("error creating cert manager: %v", err)
	}
	proxy := NewMitmProxy(certManager, filter)

	log.Println("Starting proxy server on", *addr)
	if err := http.ListenAndServe(*addr, proxy); err != nil {
		log.Panic("ListenAndServe:", err)
	}
}
