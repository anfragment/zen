package proxy

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/anfragment/zen/certmanager"
	"github.com/anfragment/zen/matcher"
)

type Proxy struct {
	host        string
	port        int
	matcher     *matcher.Matcher
	certmanager *certmanager.CertManager
	server      *http.Server
}

func NewProxy(host string, port int, matcher *matcher.Matcher, certmanager *certmanager.CertManager) *Proxy {
	return &Proxy{host, port, matcher, certmanager, nil}
}

// Start starts the proxy on the given address.
func (p *Proxy) Start() error {
	p.server = &http.Server{
		Handler: p,
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", p.host, p.port))
	if err != nil {
		return fmt.Errorf("listen: %v", err)
	}

	go func() {
		if err := p.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Fatalf("serve: %v", err)
		}
	}()

	if err := p.setSystemProxy(); err != nil {
		return fmt.Errorf("set system proxy: %v", err)
	}

	return nil
}

// Stop stops the proxy.
func (p *Proxy) Stop() error {
	if p.server == nil {
		return errors.New("proxy not started")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := p.server.Shutdown(ctx); err != nil {
		log.Printf("shutdown failed: %v", err)
	}

	if err := p.unsetSystemProxy(); err != nil {
		return fmt.Errorf("unset system proxy: %v", err)
	}

	return nil
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		p.proxyConnect(w, r)
	} else {
		p.proxyHTTP(w, r)
	}
}

// proxyHTTP proxies the HTTP request to the remote server.
func (p *Proxy) proxyHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Upgrade") == "websocket" {
		p.proxyWebsocket(w, r)
		return
	}

	client := &http.Client{
		// let the client handle any redirects
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	r.RequestURI = ""

	removeConnectionHeaders(r.Header)
	removeHopHeaders(r.Header)

	resp, err := client.Do(r)
	if err != nil {
		log.Printf("client.Do: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	removeConnectionHeaders(resp.Header)
	removeHopHeaders(resp.Header)

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// proxyConnect proxies the initial CONNECT and subsequent data between the
// client and the remote server.
func (p *Proxy) proxyConnect(w http.ResponseWriter, r *http.Request) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		log.Fatal("http server does not support hijacking")
	}

	clientConn, _, err := hj.Hijack()
	if err != nil {
		log.Fatalf("hijacking connection(%s): %v", r.Host, err)
	}
	defer clientConn.Close()

	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		log.Printf("splitting host and port(%s): %v", r.Host, err)
		return
	}
	if net.ParseIP(host) != nil || r.Header.Get("Upgrade") == "websocket" {
		// TODO: implement upstream certificate sniffing
		// https://docs.mitmproxy.org/stable/concepts-howmitmproxyworks/#complication-1-whats-the-remote-hostname
		p.tunnel(clientConn, r)
		return
	}

	pemCert, pemKey := p.certmanager.GetCertificate(host)
	tlsCert, err := tls.X509KeyPair(pemCert, pemKey)
	if err != nil {
		log.Fatalf("failed to load key pair: %v", err)
	}

	if _, err := clientConn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n")); err != nil {
		log.Printf("writing 200 OK to client(%s): %v", r.Host, err)
		return
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
	}

	tlsConn := tls.Server(clientConn, tlsConfig)
	defer tlsConn.Close()

	for {
		req, err := http.ReadRequest(bufio.NewReader(tlsConn))
		if err != nil {
			log.Printf("reading request(%s): %v", r.Host, err)
			break
		}

		req.URL.Scheme = "https"
		req.URL.Host = r.Host

		resp, err := http.DefaultTransport.RoundTrip(req)
		if err != nil {
			log.Printf("roundtrip(%s): %v", r.Host, err)
			tlsConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
			break
		}
		defer resp.Body.Close()

		if err := resp.Write(tlsConn); err != nil {
			log.Printf("writing response(%s): %v", r.Host, err)
			break
		}

		if (resp.ContentLength == 0 || resp.ContentLength == -1) &&
			!resp.Close &&
			resp.ProtoAtLeast(1, 1) &&
			!resp.Uncompressed &&
			(len(resp.TransferEncoding) == 0 || resp.TransferEncoding[0] != "chunked") {
			break
		}
	}
}

// tunnel tunnels the connection between the client and the remote server
// without inspecting the traffic.
func (p *Proxy) tunnel(w net.Conn, r *http.Request) {
	remoteConn, err := net.Dial("tcp", r.Host)
	if err != nil {
		log.Printf("dialing remote(%s): %v", r.Host, err)
		w.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer remoteConn.Close()

	if _, err := w.Write([]byte("HTTP/1.1 200 OK\r\n\r\n")); err != nil {
		log.Printf("writing 200 OK to client(%s): %v", r.Host, err)
	}

	doneC := make(chan struct{}, 2)
	go tunnelConn(remoteConn, w, doneC)
	go tunnelConn(w, remoteConn, doneC)
	<-doneC
	<-doneC
}

// tunnelConn tunnels the data between src and dst.
func tunnelConn(dst io.Writer, src io.Reader, done chan<- struct{}) {
	if _, err := io.Copy(dst, src); err != nil && !isCloseable(err) {
		log.Printf("copying: %v", err)
	}
	done <- struct{}{}
}

// isCloseable returns true if the error is one that indicates the connection
// can be closed.
func isCloseable(err error) (ok bool) {
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}

	switch err {
	case io.EOF, io.ErrClosedPipe, io.ErrUnexpectedEOF:
		return true
	default:
		return false
	}
}

// Hop-by-hop headers. These are removed when sent to the backend.
// http://www.w3.org/Protocols/rfc2616/rfc2616-sec13.html
// Note: this may be out of date, see RFC 7230 Section 6.1
var hopHeaders = []string{
	"Connection",
	"Proxy-Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",      // canonicalized version of "TE"
	"Trailer", // spelling per https://www.rfc-editor.org/errata_search.php?eid=4522
	"Transfer-Encoding",
	"Upgrade",
}

func removeHopHeaders(header http.Header) {
	for _, h := range hopHeaders {
		header.Del(h)
	}
}

// removeConnectionHeaders removes hop-by-hop headers listed in the "Connection"
// header of h. See RFC 7230, section 6.1
func removeConnectionHeaders(h http.Header) {
	for _, f := range h["Connection"] {
		for _, sf := range strings.Split(f, ",") {
			if sf = strings.TrimSpace(sf); sf != "" {
				h.Del(sf)
			}
		}
	}
}
