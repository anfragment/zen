// https://github.com/eliben/code-for-blog/blob/master/2022/go-and-proxies/connect-mitm-proxy.go
package main

import (
	"flag"
	"log"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:9999", "proxy address")
	caCertFile := flag.String("cacertfile", "", "certificate .pem file for trusted CA")
	caKeyFile := flag.String("cakeyfile", "", "key .pem file for trusted CA")
	flag.Parse()

	proxy := NewProxy()
	err := proxy.ConfigureTLS(*caCertFile, *caKeyFile)
	if err != nil {
		log.Fatalf("failed to configure TLS: %v", err)
	}
	log.Printf("starting proxy on %s", *addr)
	err = proxy.Start(*addr)
	if err != nil {
		log.Fatalf("failed to start proxy: %v", err)
	}
}
