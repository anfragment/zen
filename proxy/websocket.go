package proxy

import (
	"bufio"
	"log"
	"net"
	"net/http"
)

// proxyWebsocket proxies websocket connections over HTTP.
func (p *Proxy) proxyWebsocket(w http.ResponseWriter, r *http.Request) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		log.Fatal("http server does not support hijacking")
	}

	clientConn, _, err := hj.Hijack()
	if err != nil {
		log.Fatalf("hijacking connection(%s): %v", r.Host, err)
	}
	defer clientConn.Close()

	targetConn, err := net.Dial("tcp", r.Host)
	if err != nil {
		log.Fatalf("dialing remote server(%s): %v", r.Host, err)
		http.Error(w, err.Error(), http.StatusBadGateway)
	}

	if err := r.Write(targetConn); err != nil {
		log.Printf("writing request to target(%s): %v", r.Host, err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	targetReader := bufio.NewReader(targetConn)

	resp, err := http.ReadResponse(targetReader, r)
	if err != nil {
		log.Printf("reading response from target(%s): %v", r.Host, err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	if err := resp.Write(clientConn); err != nil {
		log.Printf("writing response to client(%s): %v", r.Host, err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	doneC := make(chan struct{}, 2)
	go tunnelConn(targetConn, clientConn, doneC)
	go tunnelConn(clientConn, targetConn, doneC)
	<-doneC
	<-doneC
}
