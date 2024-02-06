/*
 * This file contains some code originally licensed under the BSD-3-Clause license:
 * Copyright (c) 2012 Elazar Leibovich. All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are
 * met:
 *
 *    * Redistributions of source code must retain the above copyright
 * notice, this list of conditions and the following disclaimer.
 *    * Redistributions in binary form must reproduce the above
 * copyright notice, this list of conditions and the following disclaimer
 * in the documentation and/or other materials provided with the
 * distribution.
 *    * Neither the name of Elazar Leibovich. nor the names of its
 * contributors may be used to endorse or promote products derived from
 * this software without specific prior written permission.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
 * "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
 * LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
 * A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
 * OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
 * SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
 * LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
 * DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
 * THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 * (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
 * OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 *
 * The original code can be found at:
 * https://github.com/elazarl/goproxy/blob/master/websocket.go
 */
package proxy

import (
	"bufio"
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
)

func (p *Proxy) proxyWebsocketTLS(w http.ResponseWriter, req *http.Request, tlsConfig *tls.Config, clientConn *tls.Conn) {
	targetConn, err := tls.Dial("tcp", req.URL.Host, tlsConfig)
	if err != nil {
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		log.Printf("dialing websocket backend(%s): %v", req.URL.Host, err)
		return
	}
	defer targetConn.Close()

	if err := websocketHandshake(req, targetConn, clientConn); err != nil {
		return
	}

	linkBidirectionalTunnel(targetConn, clientConn)
}

func (p *Proxy) proxyWebsocket(w http.ResponseWriter, req *http.Request) {
	targetConn, err := net.Dial("tcp", req.URL.Host)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		log.Printf("dialing websocket backend(%s): %v", req.URL.Host, err)
		return
	}
	defer targetConn.Close()

	hj, ok := w.(http.Hijacker)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("http server does not support hijacking")
		return
	}
	clientConn, _, err := hj.Hijack()
	if err != nil {
		log.Printf("hijacking websocket client(%s): %v", req.URL.Host, err)
		return
	}

	if err := websocketHandshake(req, targetConn, clientConn); err != nil {
		return
	}

	linkBidirectionalTunnel(targetConn, clientConn)
}

func websocketHandshake(req *http.Request, targetConn io.ReadWriter, clientConn io.ReadWriter) error {
	err := req.Write(targetConn)
	if err != nil {
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		log.Printf("writing websocket request to backend(%s): %v", req.URL.Host, err)
		return err
	}

	targetReader := bufio.NewReader(targetConn)

	resp, err := http.ReadResponse(targetReader, req)
	if err != nil {
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		log.Printf("reading websocket response from backend(%s): %v", req.URL.Host, err)
		return err
	}
	defer resp.Body.Close()

	err = resp.Write(clientConn)
	if err != nil {
		log.Printf("writing websocket response to client(%s): %v", req.URL.Host, err)
		return err
	}

	return nil
}

func headerContains(h http.Header, name, value string) bool {
	for _, v := range h[name] {
		for _, s := range strings.Split(v, ",") {
			if strings.EqualFold(strings.TrimSpace(s), value) {
				return true
			}
		}
	}
	return false
}

func isWS(r *http.Request) bool {
	// RFC 6455, the WebSocket Protocol specification, does not explicitly specify if the Upgrade header
	// should only contain the value "websocket" or not, so we'll employ some defensive programming here.
	return headerContains(r.Header, "Connection", "upgrade") &&
		headerContains(r.Header, "Upgrade", "websocket")
}
