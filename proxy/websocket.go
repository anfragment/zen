/*
 * This file contains some code originally licensed under the GPL-3.0 license.
 * Original Author: Andrey Meshkov <am@adguard.com>
 * Original Source: https://github.com/AdguardTeam/gomitmproxy
 *
 * Modifications made by: Ansar Smagulov <me@anfragment.net>
 *
 * This modified code is licensed under the GPL-3.0 license. The full text of the GPL-3.0 license
 * is included in the COPYING file in the root of this project.
 *
 * Note: This project as a whole is licensed under the MIT License, but this particular file,
 * due to its use of GPL-3.0 licensed code, is an exception and remains licensed under the GPL-3.0.
 */
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
		log.Printf("error hijacking connection(%s): %v", r.Host, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	targetConn, err := net.Dial("tcp", r.Host)
	if err != nil {
		log.Printf("error dialing remote server(%s): %v", r.Host, err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
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
