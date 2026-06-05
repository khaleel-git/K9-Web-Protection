package proxy

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// safeSearchDomains are HTTPS domains where we do full MITM to enforce SafeSearch.
var safeSearchDomains = map[string]bool{
	"www.google.com": true,
	"google.com":     true,
	"www.bing.com":   true,
	"bing.com":       true,
}

var (
	mitmOnce  sync.Once
	mitmCA    *x509.Certificate
	mitmCADER []byte
	mitmKey   *rsa.PrivateKey
	mitmErr   error
	leafCache sync.Map // domain -> *tls.Certificate
)

// CACertPath returns the path to the K10 root CA certificate.
func CACertPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".k9webprotection", "ca.crt")
}

func initMITM() {
	mitmOnce.Do(func() {
		home, _ := os.UserHomeDir()
		dir := filepath.Join(home, ".k9webprotection")
		os.MkdirAll(dir, 0700)

		keyPath := filepath.Join(dir, "ca.key")
		certPath := filepath.Join(dir, "ca.crt")

		if tryLoadCA(keyPath, certPath) {
			return
		}

		// Generate a new root CA key + self-signed cert
		mitmKey, mitmErr = rsa.GenerateKey(rand.Reader, 2048)
		if mitmErr != nil {
			return
		}

		tmpl := &x509.Certificate{
			SerialNumber: big.NewInt(1),
			Subject: pkix.Name{
				CommonName:   "K10 Web Protection CA",
				Organization: []string{"K10 Web Protection"},
			},
			NotBefore:             time.Now().Add(-time.Hour),
			NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
			IsCA:                  true,
			KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
			BasicConstraintsValid: true,
		}

		mitmCADER, mitmErr = x509.CreateCertificate(rand.Reader, tmpl, tmpl, &mitmKey.PublicKey, mitmKey)
		if mitmErr != nil {
			return
		}
		mitmCA, mitmErr = x509.ParseCertificate(mitmCADER)
		if mitmErr != nil {
			return
		}

		kf, _ := os.OpenFile(keyPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
		pem.Encode(kf, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(mitmKey)})
		kf.Close()

		cf, _ := os.OpenFile(certPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		pem.Encode(cf, &pem.Block{Type: "CERTIFICATE", Bytes: mitmCADER})
		cf.Close()
	})
}

func tryLoadCA(keyPath, certPath string) bool {
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return false
	}
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return false
	}

	kb, _ := pem.Decode(keyPEM)
	if kb == nil {
		return false
	}
	key, err := x509.ParsePKCS1PrivateKey(kb.Bytes)
	if err != nil {
		return false
	}

	cb, _ := pem.Decode(certPEM)
	if cb == nil {
		return false
	}
	cert, err := x509.ParseCertificate(cb.Bytes)
	if err != nil {
		return false
	}

	mitmKey = key
	mitmCA = cert
	mitmCADER = cb.Bytes
	return true
}

func leafCert(domain string) (*tls.Certificate, error) {
	if v, ok := leafCache.Load(domain); ok {
		return v.(*tls.Certificate), nil
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: domain},
		DNSNames:     []string{domain, "*." + domain},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, mitmCA, &key.PublicKey, mitmKey)
	if err != nil {
		return nil, err
	}

	cert := &tls.Certificate{
		Certificate: [][]byte{der, mitmCADER},
		PrivateKey:  key,
	}
	leafCache.Store(domain, cert)
	return cert, nil
}

// blockTunnel intercepts an HTTPS CONNECT tunnel and serves the block page over TLS.
func (p *Proxy) blockTunnel(w http.ResponseWriter, domain string) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Blocked by K10 Web Protection", http.StatusForbidden)
		return
	}
	conn, _, err := hj.Hijack()
	if err != nil {
		return
	}
	defer conn.Close()

	// Accept the tunnel — browser expects 200 before starting TLS
	conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	if mitmErr != nil || mitmCA == nil {
		// CA not ready; silently drop so access is still blocked
		return
	}

	cert, err := leafCert(domain)
	if err != nil {
		return
	}

	tlsConn := tls.Server(conn, &tls.Config{
		Certificates: []tls.Certificate{*cert},
	})
	defer tlsConn.Close()

	if err := tlsConn.Handshake(); err != nil {
		return
	}

	// Drain the browser's HTTP request (path doesn't matter)
	req, err := http.ReadRequest(bufio.NewReader(tlsConn))
	if err == nil {
		req.Body.Close()
	}

	body := blockPageHTML(domain)
	fmt.Fprintf(tlsConn,
		"HTTP/1.1 403 Forbidden\r\nContent-Type: text/html; charset=utf-8\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s",
		len(body), body)
}

// safeSearchIntercept does full TLS MITM for Google/Bing to:
//   - Block /setprefs requests (prevents turning SafeSearch off)
//   - Inject safe=active into all /search query strings
//   - Forward everything else unchanged
func (p *Proxy) safeSearchIntercept(w http.ResponseWriter, r *http.Request, host string) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	clientConn, _, err := hj.Hijack()
	if err != nil {
		return
	}
	defer clientConn.Close()

	clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	if mitmCA == nil {
		// CA not ready — raw pass-through, SafeSearch not enforceable
		rawTunnel(clientConn, r.Host)
		return
	}

	cert, err := leafCert(host)
	if err != nil {
		return
	}

	tlsBrowser := tls.Server(clientConn, &tls.Config{Certificates: []tls.Certificate{*cert}})
	if err := tlsBrowser.Handshake(); err != nil {
		return
	}
	defer tlsBrowser.Close()

	tlsOrigin, err := tls.Dial("tcp", host+":443", &tls.Config{ServerName: host})
	if err != nil {
		return
	}
	defer tlsOrigin.Close()

	browserBuf := bufio.NewReader(tlsBrowser)
	originBuf := bufio.NewReader(tlsOrigin)

	for {
		req, err := http.ReadRequest(browserBuf)
		if err != nil {
			return
		}
		req.Header.Del("Proxy-Connection")

		// Block SafeSearch preference changes — prevents "Off" from taking effect
		if strings.Contains(req.URL.Path, "setprefs") || strings.Contains(req.URL.Path, "/preferences") {
			io.Copy(io.Discard, req.Body)
			req.Body.Close()
			const locked = `<html><body style="font-family:-apple-system,sans-serif;text-align:center;padding:60px">
<h2 style="color:#144985">SafeSearch is managed by K10 Web Protection</h2>
<p style="color:#555">This setting cannot be changed without administrator access.</p>
</body></html>`
			fmt.Fprintf(tlsBrowser,
				"HTTP/1.1 403 Forbidden\r\nContent-Type: text/html; charset=utf-8\r\nContent-Length: %d\r\nConnection: keep-alive\r\n\r\n%s",
				len(locked), locked)
			continue
		}

		// Force SafeSearch on every search request
		if strings.Contains(req.URL.Path, "/search") {
			q := req.URL.Query()
			q.Set("safe", "active")
			req.URL.RawQuery = q.Encode()
		}

		req.RequestURI = ""
		req.URL.Scheme = "https"
		req.URL.Host = host

		if err := req.Write(tlsOrigin); err != nil {
			return
		}

		resp, err := http.ReadResponse(originBuf, req)
		if err != nil {
			return
		}
		keepAlive := !resp.Close && !strings.EqualFold(resp.Header.Get("Connection"), "close")
		if err := resp.Write(tlsBrowser); err != nil {
			resp.Body.Close()
			return
		}
		resp.Body.Close()

		if !keepAlive {
			return
		}
	}
}

// rawTunnel is a dumb TCP pass-through used when MitM is not possible.
func rawTunnel(conn net.Conn, host string) {
	dest, err := net.DialTimeout("tcp", host, 10*time.Second)
	if err != nil {
		return
	}
	defer dest.Close()
	done := make(chan struct{}, 2)
	go func() { io.Copy(dest, conn); done <- struct{}{} }()
	go func() { io.Copy(conn, dest); done <- struct{}{} }()
	<-done
	<-done
}
