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
		log.Fatal("http server does not support hijacking")
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
