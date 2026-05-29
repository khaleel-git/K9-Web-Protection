package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"k9webprotection/internal/config"
	"k9webprotection/internal/database"
)

const blockPage = `<!DOCTYPE html>
<html>
<head><title>Blocked — K9 Web Protection</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;
     background:linear-gradient(135deg,#0f1b2d 0%%,#1a3a5c 100%%);
     min-height:100vh;display:flex;align-items:center;justify-content:center}
.card{background:rgba(255,255,255,0.05);backdrop-filter:blur(20px);
      border:1px solid rgba(255,255,255,0.1);border-radius:24px;
      padding:48px;text-align:center;max-width:480px;width:90%%}
.shield{font-size:72px;margin-bottom:24px}
h1{color:#fff;font-size:28px;font-weight:700;margin-bottom:12px}
p{color:rgba(255,255,255,0.6);font-size:16px;line-height:1.6}
.badge{display:inline-block;background:rgba(0,212,255,0.15);
       color:#00d4ff;border:1px solid rgba(0,212,255,0.3);
       border-radius:20px;padding:6px 16px;font-size:13px;margin-top:24px}
</style></head>
<body>
<div class="card">
  <div class="shield">🛡️</div>
  <h1>Blocked by K9 Web Protection</h1>
  <p>This website has been blocked to help you stay focused and protected.</p>
  <div class="badge">K9 Web Protection</div>
</div>
</body></html>`

type OnBlockFn func(domain string)

type Proxy struct {
	cfg     *config.Config
	db      *database.Database
	server  *http.Server
	onBlock OnBlockFn
}

func New(cfg *config.Config, onBlock OnBlockFn) *Proxy {
	return &Proxy{cfg: cfg, db: database.DB, onBlock: onBlock}
}

func (p *Proxy) Start() error {
	p.server = &http.Server{
		Addr:              fmt.Sprintf("127.0.0.1:%d", p.cfg.ProxyPort),
		Handler:           http.HandlerFunc(p.handle),
		ReadHeaderTimeout: 15 * time.Second,
	}
	return p.server.ListenAndServe()
}

func (p *Proxy) Stop() {
	if p.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		p.server.Shutdown(ctx)
		p.server = nil
	}
}

// ── Request handler ───────────────────────────────────────────────────────────

func (p *Proxy) handle(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if rec := recover(); rec != nil {
			http.Error(w, "Internal proxy error", http.StatusInternalServerError)
		}
	}()

	host := hostname(r.Host)

	if p.cfg.IsAllowed(host) {
		p.passThrough(w, r)
		return
	}

	if p.cfg.UserBlocks(host) {
		p.block(w, r, host)
		return
	}

	if p.cfg.BlockYouTube && isYouTube(host) {
		p.block(w, r, host)
		return
	}

	if p.cfg.BlockImageSearch && isImageSearch(host, r.URL.String()) {
		p.block(w, r, host)
		return
	}

	if p.cfg.BlockAdultContent {
		if p.db.BlocksDomain(host) {
			p.block(w, r, host)
			return
		}
		if r.Method != http.MethodConnect {
			rawURL := r.URL.String()
			if p.db.BlocksURL(rawURL) || p.db.BlocksKeyword(rawURL) {
				p.block(w, r, host)
				return
			}
		}
	}

	if r.Method != http.MethodConnect && p.cfg.UserKeywordMatch(r.URL.String()) {
		p.block(w, r, host)
		return
	}

	p.passThrough(w, r)
}

func (p *Proxy) passThrough(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		p.tunnel(w, r)
	} else {
		p.forward(w, r)
	}
}

func (p *Proxy) block(w http.ResponseWriter, r *http.Request, domain string) {
	if p.onBlock != nil {
		go p.onBlock(domain)
	}
	if r.Method == http.MethodConnect {
		http.Error(w, "Blocked by K9 Web Protection", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte(blockPage))
}

func (p *Proxy) tunnel(w http.ResponseWriter, r *http.Request) {
	dest, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		dest.Close()
		http.Error(w, "Hijack unsupported", http.StatusInternalServerError)
		return
	}
	conn, _, err := hj.Hijack()
	if err != nil {
		dest.Close()
		return
	}

	conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	done := make(chan struct{}, 2)
	go func() {
		io.Copy(dest, conn)
		if tc, ok := dest.(*net.TCPConn); ok {
			tc.CloseWrite()
		}
		done <- struct{}{}
	}()
	go func() {
		io.Copy(conn, dest)
		if tc, ok := conn.(*net.TCPConn); ok {
			tc.CloseWrite()
		}
		done <- struct{}{}
	}()
	<-done
	<-done
	conn.Close()
	dest.Close()
}

func (p *Proxy) forward(w http.ResponseWriter, r *http.Request) {
	r.RequestURI = ""
	if r.URL.Scheme == "" {
		r.URL.Scheme = "http"
	}
	if r.URL.Host == "" {
		r.URL.Host = r.Host
	}
	r.Header.Del("Proxy-Connection")

	resp, err := (&http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}).Do(r)
	if err != nil {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, vs := range resp.Header {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func hostname(host string) string {
	h, _, err := net.SplitHostPort(host)
	if err != nil {
		return strings.ToLower(strings.TrimSpace(host))
	}
	return strings.ToLower(h)
}

var youtubeDomains = []string{
	"youtube.com", "youtu.be", "youtube-nocookie.com",
	"youtubei.googleapis.com", "yt3.ggpht.com",
}

func isYouTube(host string) bool {
	for _, d := range youtubeDomains {
		if host == d || strings.HasSuffix(host, "."+d) {
			return true
		}
	}
	return false
}

var imageSearchPatterns = []string{
	"google.com/imghp", "google.com/search?", "/tbm=isch",
	"bing.com/images", "images.google.",
	"search.yahoo.com/image", "duckduckgo.com/?ia=images",
	"yandex.com/images",
}

func isImageSearch(host, rawURL string) bool {
	combined := strings.ToLower(host + rawURL)
	for _, p := range imageSearchPatterns {
		if strings.Contains(combined, p) {
			return true
		}
	}
	return false
}
