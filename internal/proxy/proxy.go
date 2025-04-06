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
	"sync"
	"time"

	"github.com/ZenPrivacy/zen-desktop/internal/logger"
)

// certGenerator is an interface capable of generating certificates for the proxy.
type certGenerator interface {
	GetCertificate(host string) (*tls.Certificate, error)
}

// filter is an interface capable of filtering HTTP requests.
type filter interface {
	HandleRequest(*http.Request) *http.Response
	HandleResponse(*http.Request, *http.Response) error
}

// Proxy is a forward HTTP/HTTPS proxy that can filter requests.
type Proxy struct {
	filter             filter
	certGenerator      certGenerator
	port               int
	server             *http.Server
	requestTransport   http.RoundTripper
	requestClient      *http.Client
	netDialer          *net.Dialer
	transparentHosts   []string
	transparentHostsMu sync.RWMutex
}

func NewProxy(filter filter, certGenerator certGenerator, port int) (*Proxy, error) {
	if filter == nil {
		return nil, errors.New("filter is nil")
	}
	if certGenerator == nil {
		return nil, errors.New("certGenerator is nil")
	}

	p := &Proxy{
		filter:        filter,
		certGenerator: certGenerator,
		port:          port,
	}

	p.netDialer = &net.Dialer{
		// Such high values are set to avoid timeouts on slow connections.
		Timeout:   60 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	p.requestTransport = &http.Transport{
		Dial:                p.netDialer.Dial,
		TLSHandshakeTimeout: 20 * time.Second,
	}
	p.requestClient = &http.Client{
		Timeout:   60 * time.Second,
		Transport: p.requestTransport,
		// Let the client handle any redirects.
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return p, nil
}

// Start starts the proxy on the given address.
//
// If Proxy was configured with a port of 0, the actual port will be returned.
func (p *Proxy) Start() (int, error) {
	p.server = &http.Server{
		Handler:           p,
		ReadHeaderTimeout: 10 * time.Second,
	}
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", "127.0.0.1", p.port))
	if err != nil {
		return 0, fmt.Errorf("listen: %v", err)
	}
	actualPort := listener.Addr().(*net.TCPAddr).Port
	log.Printf("proxy listening on port %d", actualPort)

	go func() {
		if err := p.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("serve: %v", err)
		}
	}()

	return actualPort, nil
}

// Stop stops the proxy.
func (p *Proxy) Stop() error {
	if err := p.shutdownServer(); err != nil {
		return fmt.Errorf("shut down server: %v", err)
	}

	return nil
}

func (p *Proxy) shutdownServer() error {
	if p.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := p.server.Shutdown(ctx); err != nil {
		// As per documentation:
		// Shutdown does not attempt to close nor wait for hijacked connections such as WebSockets. The caller of Shutdown should separately notify such long-lived connections of shutdown and wait for them to close, if desired. See RegisterOnShutdown for a way to register shutdown notification functions.
		// TODO: implement websocket shutdown
		return fmt.Errorf("server shutdown: %w", err)
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
	if filterResp := p.filter.HandleRequest(r); filterResp != nil {
		filterResp.Write(w)
		return
	}

	if isWS(r) {
		// should we remove hop-by-hop headers here?
		p.proxyWebsocket(w, r)
		return
	}

	r.RequestURI = ""

	removeHopHeaders(r.Header)

	resp, err := p.requestClient.Do(r)
	if err != nil {
		log.Printf("error making request: %v", logger.Redacted(err)) // The error might contain information about the hostname we are connecting to.
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	removeHopHeaders(resp.Header)

	if err := p.filter.HandleResponse(r, resp); err != nil {
		log.Printf("error handling response by filter: %v", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

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
func (p *Proxy) proxyConnect(w http.ResponseWriter, connReq *http.Request) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		log.Fatal("http server does not support hijacking")
	}

	clientConn, _, err := hj.Hijack()
	if err != nil {
		log.Printf("hijacking connection(%s): %v", logger.Redacted(connReq.Host), err)
		return
	}
	defer clientConn.Close()

	host, _, err := net.SplitHostPort(connReq.Host)
	if err != nil {
		log.Printf("splitting host and port(%s): %v", logger.Redacted(connReq.Host), err)
		return
	}

	if !p.shouldMITM(host) || net.ParseIP(host) != nil {
		// TODO: implement upstream certificate sniffing
		// https://docs.mitmproxy.org/stable/concepts-howmitmproxyworks/#complication-1-whats-the-remote-hostname
		p.tunnel(clientConn, connReq)
		return
	}

	tlsCert, err := p.certGenerator.GetCertificate(host)
	if err != nil {
		log.Printf("getting certificate(%s): %v", logger.Redacted(connReq.Host), err)
		return
	}

	if _, err := clientConn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n")); err != nil {
		log.Printf("writing 200 OK to client(%s): %v", logger.Redacted(connReq.Host), err)
		return
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{*tlsCert},
		MinVersion:   tls.VersionTLS12,
	}

	tlsConn := tls.Server(clientConn, tlsConfig)
	defer tlsConn.Close()
	connReader := bufio.NewReader(tlsConn)

	// Read requests in a loop to allow for HTTP connection reuse.
	// https://en.wikipedia.org/wiki/HTTP_persistent_connection
	for {
		req, err := http.ReadRequest(connReader)
		if err != nil {
			if err != io.EOF {
				if strings.Contains(err.Error(), "tls: ") {
					log.Printf("adding %s to ignored hosts", logger.Redacted(host))
					p.addTransparentHost(host)
				}

				log.Printf("reading request(%s): %v", logger.Redacted(connReq.Host), err)
			}
			break
		}
		req.URL.Host = connReq.Host

		if isWS(req) {
			// Establish transparent flow, no hop-by-hop header removal required.
			p.proxyWebsocketTLS(req, tlsConfig, tlsConn)
			break
		}

		// A standard CONNECT proxy establishes a TCP connection to the requested destination and relays the stream between the client and server.
		// Here, we are MITM-ing the traffic and handling the request-response flow ourselves.
		// Since the client and server do not share a direct TCP connection in this setup, we must strip hop-by-hop headers.
		removeHopHeaders(req.Header)
		req.URL.Scheme = "https"

		if filterResp := p.filter.HandleRequest(req); filterResp != nil {
			filterResp.Write(tlsConn)
			break
		}

		resp, err := p.requestTransport.RoundTrip(req)
		if err != nil {
			if strings.Contains(err.Error(), "tls: ") {
				log.Printf("adding %s to ignored hosts", logger.Redacted(host))
				p.addTransparentHost(host)
			}

			log.Printf("roundtrip(%s): %v", logger.Redacted(connReq.Host), err)
			// TODO: better error presentation
			response := fmt.Sprintf("HTTP/1.1 502 Bad Gateway\r\n\r\n%s", err.Error())
			tlsConn.Write([]byte(response))
			break
		}

		removeHopHeaders(resp.Header)

		if err := p.filter.HandleResponse(req, resp); err != nil {
			log.Printf("error handling response by filter for %s, %v", logger.Redacted(req.URL), err)
			response := fmt.Sprintf("HTTP/1.1 502 Bad Gateway\r\n\r\n%s", err.Error())
			tlsConn.Write([]byte(response))
			break
		}

		if err := resp.Write(tlsConn); err != nil {
			log.Printf("writing response(%q): %v", logger.Redacted(connReq.Host), err)
			if err := resp.Body.Close(); err != nil {
				log.Printf("closing body(%q): %v", logger.Redacted(connReq.Host), err)
			}
			break
		}
		if err := resp.Body.Close(); err != nil {
			log.Printf("closing body(%q): %v", logger.Redacted(connReq.Host), err)
		}

		if req.Close || resp.Close {
			break
		}
	}
}

// shouldMITM returns true if the host should be MITM'd.
func (p *Proxy) shouldMITM(host string) bool {
	p.transparentHostsMu.RLock()
	defer p.transparentHostsMu.RUnlock()

	for _, transparentHost := range p.transparentHosts {
		if host == transparentHost || strings.HasSuffix(host, "."+transparentHost) {
			return false
		}
	}

	return true
}

// addTransparentHost adds a host to the list of hosts that should be MITM'd.
func (p *Proxy) addTransparentHost(host string) {
	p.transparentHostsMu.Lock()
	defer p.transparentHostsMu.Unlock()

	p.transparentHosts = append(p.transparentHosts, host)
}

// tunnel tunnels the connection between the client and the remote server
// without inspecting the traffic.
func (p *Proxy) tunnel(w net.Conn, r *http.Request) {
	remoteConn, err := net.Dial("tcp", r.Host)
	if err != nil {
		log.Printf("dialing remote(%s): %v", logger.Redacted(r.Host), err)
		w.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer remoteConn.Close()

	if _, err := w.Write([]byte("HTTP/1.1 200 OK\r\n\r\n")); err != nil {
		log.Printf("writing 200 OK to client(%s): %v", logger.Redacted(r.Host), err)
		return
	}

	linkBidirectionalTunnel(w, remoteConn)
}

func linkBidirectionalTunnel(src, dst io.ReadWriter) {
	doneC := make(chan struct{}, 2)
	go tunnelConn(src, dst, doneC)
	go tunnelConn(dst, src, doneC)
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
// Note: this may be out of date, see RFC 7230 Section 6.1.
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
