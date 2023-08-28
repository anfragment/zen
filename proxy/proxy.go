package proxy

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"github.com/anfragment/zen/certmanager"
	"github.com/anfragment/zen/matcher"
)

type Proxy struct {
	host        string
	port        int
	matcher     *matcher.Matcher
	certmanager *certmanager.CertManager
}

func NewProxy(host string, port int, matcher *matcher.Matcher, certmanager *certmanager.CertManager) *Proxy {
	return &Proxy{host, port, matcher, certmanager}
}

// ConfigureTLS configures the proxy to use the given certificate and key for TLS connections.
func (p *Proxy) ConfigureTLS(certData, keyData []byte) error {
	return nil
}

// Start starts the proxy on the given address.
func (p *Proxy) Start() error {
	errC := make(chan error, 1)
	go func() {
		errC <- http.ListenAndServe(fmt.Sprintf("%s:%d", p.host, p.port), p)
	}()

	if err := p.setSystemProxy(); err != nil {
		return fmt.Errorf("set system proxy: %v", err)
	}
	defer func() {
		log.Println("unsetting system proxy")
		if err := p.unsetSystemProxy(); err != nil {
			log.Printf("failed to unset system proxy: %v", err)
		} else {
			log.Println("system proxy unset")
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	select {
	case err := <-errC:
		return err
	case <-signals:
		return nil
	}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		p.proxyConnect(w, r)
	} else {
		log.Printf("%s %s", r.Method, r.URL)
		p.proxyHTTP(w, r)
	}
}

// proxyHTTP proxies the HTTP request to the remote server.
func (p *Proxy) proxyHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Scheme != "http" {
		msg := "unsupported prootocol scheme " + r.URL.Scheme
		http.Error(w, msg, http.StatusBadRequest)
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

	if clientIP, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		appendHostToXForwardHeader(r.Header, clientIP)
	}

	resp, err := client.Do(r)
	if err != nil {
		log.Printf("client.Do: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	removeConnectionHeaders(resp.Header)
	removeHopHeaders(resp.Header)

	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
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

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
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

func appendHostToXForwardHeader(header http.Header, host string) {
	// If we aren't the first proxy retain prior
	// X-Forwarded-For information as a comma+space
	// separated list and fold multiple headers into one.
	if prior, ok := header["X-Forwarded-For"]; ok {
		host = strings.Join(prior, ", ") + ", " + host
	}
	header.Set("X-Forwarded-For", host)
}

func (p *Proxy) proxyConnect(w http.ResponseWriter, r *http.Request) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		log.Fatal("http server does not support hijacking")
	}

	clientConn, _, err := hj.Hijack()
	if err != nil {
		log.Fatalf("hijacking connection: %v", err)
	}
	defer clientConn.Close()

	host, _, err := net.SplitHostPort(r.Host)
	if err != nil || r.Header.Get("Upgrade") == "websocket" {
		p.tunnel(clientConn, r)
		return
	}

	pemCert, pemKey := p.certmanager.GetCertificate(host)
	tlsCert, err := tls.X509KeyPair(pemCert, pemKey)
	if err != nil {
		log.Fatalf("failed to load key pair: %v", err)
	}

	if _, err := clientConn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n")); err != nil {
		log.Fatalf("writing 200 OK to client (%s): %v", r.Host, err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
	}

	tlsConn := tls.Server(clientConn, tlsConfig)
	defer tlsConn.Close()
	if err := tlsConn.Handshake(); err != nil {
		log.Printf("handshake (%s): %v", r.Host, err)
		return
	}

	for {
		req, err := http.ReadRequest(bufio.NewReader(tlsConn))
		if err == io.EOF {
			break
		} else if err != nil {
			log.Printf("reading request (%s): %v", r.Host, err)
			break
		}

		req.URL.Scheme = "https"
		req.URL.Host = r.Host

		resp, err := http.DefaultTransport.RoundTrip(req)
		if err != nil {
			log.Printf("roundtrip(%s): %v", r.Host, err)
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

		if resp.Close || req.Close {
			break
		}
	}
}

// tunnel tunnels the connection between the client and the remote server
// without inspecting the traffic.
func (p *Proxy) tunnel(w net.Conn, r *http.Request) {
	remoteConn, err := net.Dial("tcp", r.Host)
	if err != nil {
		log.Fatalf("dialing remote: %v", err)
	}
	defer remoteConn.Close()

	if _, err := w.Write([]byte("HTTP/1.1 200 OK\r\n\r\n")); err != nil {
		log.Fatalf("writing 200 OK to client: %v", err)
	}

	doneC := make(chan bool, 2)
	go tunnelConn(remoteConn, w, doneC)
	go tunnelConn(w, remoteConn, doneC)
	<-doneC
	<-doneC
}

func tunnelConn(dst io.WriteCloser, src io.ReadCloser, done chan<- bool) {
	if _, err := io.Copy(dst, src); err != nil && !isCloseable(err) {
		log.Printf("copying: %v", err)
	}
	dst.Close()
	src.Close()
	done <- true
}

func isCloseable(err error) (ok bool) {
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}

	switch err {
	case io.EOF, io.ErrClosedPipe, io.ErrUnexpectedEOF:
		return true
	}

	return false
}
