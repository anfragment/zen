package main

import (
	"flag"
	"log"

	"github.com/anfragment/zen/matcher"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:9999", "proxy address")
	caCertFile := flag.String("cacertfile", "", "certificate .pem file for trusted CA")
	caKeyFile := flag.String("cakeyfile", "", "key .pem file for trusted CA")
	flag.Parse()

	matcher := matcher.NewMatcher()
	matcher.AddRemoteFilters([]string{
		"https://cdn.statically.io/gh/uBlockOrigin/uAssetsCDN/main/thirdparties/easylist.txt",
		"https://pgl.yoyo.org/adservers/serverlist.php?hostformat=hosts&showintro=1&mimetype=plaintext",
		"https://ublockorigin.pages.dev/thirdparties/easyprivacy.txt",
	})
	proxy := NewProxy(matcher)
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
