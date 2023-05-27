// https://github.com/eliben/code-for-blog/blob/master/2022/go-and-proxies/connect-mitm-proxy.go
package main

import (
	"bufio"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"time"
)

// createCert creates a new certificate/private key pair for the given domains,
// signed by the parent/parentKey certificate. hoursValid is the duration of
// the new certificate's validity.
func createCert(dnsNames []string, parent *x509.Certificate, parentKey crypto.PrivateKey, hoursValid int) (cert []byte, priv []byte) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Panicf("Failed to generate private key: %v", err)
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Panicf("Failed to generate serial number: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Sample MITM proxy"},
		},
		DNSNames:  dnsNames,
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Duration(hoursValid) * time.Hour),

		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, parent, &privateKey.PublicKey, parentKey)
	if err != nil {
		log.Panicf("Failed to create certificate: %v", err)
	}
	pemCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if pemCert == nil {
		log.Panic("failed to encode certificate to PEM")
	}

	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		log.Panicf("Unable to marshal private key: %v", err)
	}
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	if pemCert == nil {
		log.Panic("failed to encode key to PEM")
	}

	return pemCert, pemKey
}

// loadX509KeyPair loads a certificate/key pair from files, and unmarshals them
// into data structures from the x509 package. Note that private key types in Go
// don't have a shared named interface and use `any` (for backwards
// compatibility reasons).
func loadX509KeyPair(certFile, keyFile string) (cert *x509.Certificate, key any, err error) {
	cf, err := ioutil.ReadFile(certFile)
	if err != nil {
		return nil, nil, err
	}

	kf, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return nil, nil, err
	}
	certBlock, _ := pem.Decode(cf)
	cert, err = x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}

	keyBlock, _ := pem.Decode(kf)
	key, err = x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}

	return cert, key, nil
}

// mitmProxy is a type implementing http.Handler that serves as a MITM proxy
// for CONNECT tunnels. Create new instances of mitmProxy using createMitmProxy.
type mitmProxy struct {
	caCert *x509.Certificate
	caKey  any
}

// createMitmProxy creates a new MITM proxy. It should be passed the filenames
// for the certificate and private key of a certificate authority trusted by the
// client's machine.
func createMitmProxy(caCertFile, caKeyFile string) *mitmProxy {
	caCert, caKey, err := loadX509KeyPair(caCertFile, caKeyFile)
	if err != nil {
		log.Panic("Error loading CA certificate/key:", err)
	}
	log.Printf("loaded CA certificate and key; IsCA=%v\n", caCert.IsCA)

	return &mitmProxy{
		caCert: caCert,
		caKey:  caKey,
	}
}

func (p *mitmProxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("panic serving %v: %v. stacktrace from panic: %s", req.URL, r, string(debug.Stack()))
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
	}()

	if req.Method == http.MethodConnect {
		p.proxyConnect(w, req)
	} else {
		p.proxyHTTP(w, req)
	}
}

func (p *mitmProxy) proxyHTTP(w http.ResponseWriter, req *http.Request) {
	// The "Host:" header is promoted to Request.Host and is removed from
	// request.Header by net/http, so we print it out explicitly.
	log.Println(req.RemoteAddr, "\t\t", req.Method, "\t\t", req.URL, "\t\t Host:", req.Host)
	log.Println("\t\t\t\t\t", req.Header)

	if req.URL.Scheme != "http" {
		msg := "unsupported protocol scheme " + req.URL.Scheme
		http.Error(w, msg, http.StatusBadRequest)
		log.Println(msg)
		return
	}

	client := &http.Client{}
	// When a http.Request is sent through an http.Client, RequestURI should not
	// be set (see documentation of this field).
	req.RequestURI = ""

	removeHopHeaders(req.Header)
	removeConnectionHeaders(req.Header)

	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		appendHostToXForwardHeader(req.Header, clientIP)
	}

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}
	defer resp.Body.Close()

	log.Println(req.RemoteAddr, " ", resp.Status)

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

var client = &http.Client{
	// do not follow redirects, let the client handle them
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
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

// proxyConnect implements the MITM proxy for CONNECT tunnels.
func (p *mitmProxy) proxyConnect(w http.ResponseWriter, proxyReq *http.Request) {
	log.Printf("CONNECT requested to %v (from %v)", proxyReq.Host, proxyReq.RemoteAddr)

	// "Hijack" the client connection to get a TCP (or TLS) socket we can read
	// and write arbitrary data to/from.
	hj, ok := w.(http.Hijacker)
	if !ok {
		log.Panic("http server doesn't support hijacking connection")
	}

	clientConn, _, err := hj.Hijack()
	if err != nil {
		log.Panic("http hijacking failed")
	}

	// proxyReq.Host will hold the CONNECT target host, which will typically have
	// a port - e.g. example.org:443
	// To generate a fake certificate for example.org, we have to first split off
	// the host from the port.
	host, _, err := net.SplitHostPort(proxyReq.Host)
	if err != nil {
		log.Panic("error splitting host/port:", err)
	}

	// Create a fake TLS certificate for the target host, signed by our CA. The
	// certificate will be valid for 10 days - this number can be changed.
	pemCert, pemKey := createCert([]string{host}, p.caCert, p.caKey, 240)
	tlsCert, err := tls.X509KeyPair(pemCert, pemKey)
	if err != nil {
		log.Panic(err)
	}

	// Send an HTTP OK response back to the client; this initiates the CONNECT
	// tunnel. From this point on the client will assume it's connected directly
	// to the target.
	if _, err := clientConn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n")); err != nil {
		log.Panic("error writing status to client:", err)
	}

	// Configure a new TLS server, pointing it at the client connection, using
	// our certificate. This server will now pretend being the target.
	tlsConfig := &tls.Config{
		PreferServerCipherSuites: true,
		CurvePreferences:         []tls.CurveID{tls.X25519, tls.CurveP256},
		MinVersion:               tls.VersionTLS12,
		Certificates:             []tls.Certificate{tlsCert},
	}

	tlsConn := tls.Server(clientConn, tlsConfig)
	// tls.Conn.Close() also closes the underlying net.Conn
	defer tlsConn.Close()

	connReader := bufio.NewReader(tlsConn)

	r, err := http.ReadRequest(connReader)
	if err != nil && err != io.EOF {
		log.Panic(err)
	} else if err == io.EOF {
		return
	}

	changeRequestToTarget(r, proxyReq.Host)

	resp, err := client.Do(r)
	if err != nil {
		log.Println("error sending request to target:", err)
		return
	}
	defer resp.Body.Close()

	if err := resp.Write(tlsConn); err != nil {
		log.Println("error writing response back:", err)
	}
}

// changeRequestToTarget modifies req to be re-routed to the given target;
// the target should be taken from the Host of the original tunnel (CONNECT)
// request.
func changeRequestToTarget(req *http.Request, targetHost string) {
	targetUrl := addrToUrl(targetHost)
	targetUrl.Path = req.URL.Path
	targetUrl.RawQuery = req.URL.RawQuery
	// Unescape the path - the client will have escaped it, but the target
	// server will expect it unescaped since it's part of the HTTP request.
	path, err := url.PathUnescape(targetUrl.String())
	if err != nil {
		log.Panic(err)
	}
	req.URL, err = url.Parse(path)
	if err != nil {
		log.Panicf("error parsing target URL %v: %v", path, err)
	}
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

func main() {
	var addr = flag.String("addr", "127.0.0.1:9999", "proxy address")
	caCertFile := flag.String("cacertfile", "", "certificate .pem file for trusted CA")
	caKeyFile := flag.String("cakeyfile", "", "key .pem file for trusted CA")
	flag.Parse()

	proxy := createMitmProxy(*caCertFile, *caKeyFile)

	log.Println("Starting proxy server on", *addr)
	if err := http.ListenAndServe(*addr, proxy); err != nil {
		log.Panic("ListenAndServe:", err)
	}
}
