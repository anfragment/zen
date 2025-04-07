package proxy

import (
	"bufio"
	"crypto/tls"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/ZenPrivacy/zen-desktop/internal/logger"
)

func (p *Proxy) proxyWebsocketTLS(req *http.Request, tlsConfig *tls.Config, clientConn *tls.Conn) {
	dialer := &tls.Dialer{NetDialer: p.netDialer, Config: tlsConfig}
	targetConn, err := dialer.Dial("tcp", req.URL.Host)
	if err != nil {
		log.Printf("dialing websocket backend(%s): %v", logger.Redacted(req.URL.Host), err)
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer targetConn.Close()

	if err := websocketHandshake(req, targetConn, clientConn); err != nil {
		return
	}

	linkBidirectionalTunnel(targetConn, clientConn)
}

func (p *Proxy) proxyWebsocket(w http.ResponseWriter, req *http.Request) {
	targetConn, err := p.netDialer.Dial("tcp", req.URL.Host)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		log.Printf("dialing websocket backend(%s): %v", logger.Redacted(req.URL.Host), err)
		return
	}
	defer targetConn.Close()

	hj, ok := w.(http.Hijacker)
	if !ok {
		panic("http server does not support hijacking")
	}
	clientConn, _, err := hj.Hijack()
	if err != nil {
		log.Printf("hijacking websocket client(%s): %v", logger.Redacted(req.URL.Host), err)
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
		log.Printf("writing websocket request to backend(%s): %v", logger.Redacted(req.URL.Host), err)
		return err
	}

	targetReader := bufio.NewReader(targetConn)

	resp, err := http.ReadResponse(targetReader, req)
	if err != nil {
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		log.Printf("reading websocket response from backend(%s): %v", logger.Redacted(req.URL.Host), err)
		return err
	}
	defer resp.Body.Close()

	err = resp.Write(clientConn)
	if err != nil {
		log.Printf("writing websocket response to client(%s): %v", logger.Redacted(req.URL.Host), err)
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
	// should only contain the value "websocket" or not, so we employ some defensive programming here.
	return headerContains(r.Header, "Connection", "upgrade") &&
		headerContains(r.Header, "Upgrade", "websocket")
}
