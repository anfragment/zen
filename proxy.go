package main

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
)

type MitmProxy struct {
	certManager *CertManager
	// httpClient is the http.Client used to make requests to the backend.
	httpClient *http.Client
	filter     *Filter
}

func NewMitmProxy(certManager *CertManager, filter *Filter) *MitmProxy {
	proxy := &MitmProxy{
		certManager: certManager,
		httpClient: &http.Client{
			// let the client handle redirects
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		filter: filter,
	}
	return proxy
}

func (p *MitmProxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodConnect {
		err := p.proxyConnect(w, req)
		if err != nil {
			log.Printf("proxying CONNECT request: %v", err)
		}
	} else {
		p.proxyHTTP(w, req)
	}
}

func (p *MitmProxy) proxyHTTP(w http.ResponseWriter, req *http.Request) {
	if req.URL.Scheme != "http" {
		msg := "unsupported protocol scheme " + req.URL.Scheme
		http.Error(w, msg, http.StatusBadRequest)
		log.Println(msg)
		return
	}

	req.RequestURI = ""

	// TODO: this is fine for most requests, but there are some cases where
	// this is not what the client wants, e.g. when client initiates a
	// websocket connection. Investigate how to handle this.
	removeHopHeaders(req.Header)
	removeConnectionHeaders(req.Header)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("proxying request: %v", err)
		return
	}
	defer resp.Body.Close()

	removeHopHeaders(resp.Header)
	removeConnectionHeaders(resp.Header)

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
		header.Del(strings.ToLower(h))
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

// proxyConnect implements the MITM proxy for CONNECT tunnels.
func (p *MitmProxy) proxyConnect(w http.ResponseWriter, req *http.Request) (err error) {
	// proxyReq.Host will hold the CONNECT target host, which will typically have
	// a port - e.g. example.org:443
	host, _, err := net.SplitHostPort(req.Host)
	if err != nil {
		http.Error(w, "invalid host", http.StatusBadRequest)
		return fmt.Errorf("splitting host and port: %s - %w", req.Host, err)
	}

	// Check if the CONNECT target host is allowed by the filter.
	if p.filter.IsBlocked(host) {
		log.Printf("%v blocked by filter", host)
		http.Error(w, "blocked by Zen", http.StatusForbidden)
		return nil
	}

	// and write arbitrary data to/from.
	hj, ok := w.(http.Hijacker)
	if !ok {
		return errors.New("hijacking not supported")
	}

	clientConn, _, err := hj.Hijack()
	if err != nil {
		return fmt.Errorf("hijacking failed: %w", err)
	}
	defer clientConn.Close()

	// Create a fake TLS certificate for the target host, signed by our CA. The
	// certificate will be valid for 10 days - this number can be changed.
	tlsCert, err := p.certManager.CreateCert([]string{host}, 240)
	if err != nil {
		return fmt.Errorf("creating cert: %w", err)
	}

	// Send an HTTP OK response back to the client; this initiates the CONNECT
	// tunnel. From this point on the client will assume it's connected directly
	// to the target.
	if _, err := clientConn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n")); err != nil {
		return fmt.Errorf("writing to client for host %s: %w", host, err)
	}

	// Configure a new TLS server, pointing it at the client connection, using
	// our certificate. This server will now pretend being the target.
	tlsConfig := &tls.Config{
		PreferServerCipherSuites: true,
		CurvePreferences:         []tls.CurveID{tls.X25519, tls.CurveP256},
		MinVersion:               tls.VersionTLS12,
		Certificates:             []tls.Certificate{*tlsCert},
	}

	tlsConn := tls.Server(clientConn, tlsConfig)
	defer tlsConn.Close()

	connReader := bufio.NewReader(tlsConn)

	r, err := http.ReadRequest(connReader)
	if err == io.EOF {
		return nil
	} else if err != nil {
		clientConn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
		return fmt.Errorf("reading request for host %s: %w", host, err)
	}

	changeRequestToTarget(r, req.Host)

	resp, err := p.httpClient.Do(r)
	if err != nil {
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return fmt.Errorf("error sending request for host %s: %w", host, err)
	}
	defer resp.Body.Close()

	// Proxy the response back to the client.
	if err := resp.Write(tlsConn); err != nil {
		return fmt.Errorf("error writing response for host %s: %w", host, err)
	}

	return nil
}

// changeRequestToTarget modifies req to be re-routed to the given target;
// the target should be taken from the Host of the original tunnel (CONNECT)
// request.
func changeRequestToTarget(req *http.Request, targetHost string) {
	targetUrl := addrToUrl(targetHost)
	targetUrl.Path = req.URL.Path
	targetUrl.RawQuery = req.URL.RawQuery
	req.URL = targetUrl
	// Make sure this is unset for sending the request through a client
	req.RequestURI = ""
}

func addrToUrl(addr string) *url.URL {
	if !strings.HasPrefix(addr, "https") {
		addr = "https://" + addr
	}
	u, err := url.Parse(addr)
	if err != nil {
		log.Panic(err)
	}
	return u
}
