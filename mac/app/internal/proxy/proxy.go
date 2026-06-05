package proxy

import (
	"context"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"k10webprotection/internal/config"
	"k10webprotection/internal/database"
)

const blockPageTpl = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Blocked — K10 Web Protection</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
html,body{height:100%%;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI','Helvetica Neue',Arial,sans-serif;background:#d4d0c8;display:flex;align-items:center;justify-content:center;padding:20px}
.frame{background:#fff;border:1px solid #888;box-shadow:2px 2px 10px rgba(0,0,0,.25);max-width:520px;width:100%%;overflow:hidden}
.hdr{background:linear-gradient(180deg,#1a5ca8 0%%,#144985 60%%,#0d3260 100%%);padding:12px 18px;display:flex;align-items:center;gap:12px}
.hdr-logo{width:40px;height:40px;background:rgba(255,255,255,.15);border-radius:50%%;display:flex;align-items:center;justify-content:center;flex-shrink:0}
.hdr-title{font-size:16px;font-weight:800;color:#fff;letter-spacing:-.2px}
.body{padding:36px 32px 28px;text-align:center}
.icon-wrap{margin:0 auto 18px}
.chip{display:inline-block;background:#fde8e8;color:#8b0000;font-size:10px;font-weight:800;letter-spacing:1.5px;text-transform:uppercase;padding:3px 12px;border-radius:2px;margin-bottom:14px;border:1px solid #f0c0c0}
h1{font-size:20px;font-weight:800;color:#144985;margin-bottom:14px}
.domain-row{display:flex;align-items:center;justify-content:center;gap:8px;margin-bottom:18px}
.domain-label{font-size:11px;font-weight:700;color:#555;text-transform:uppercase;letter-spacing:.8px}
.domain-val{background:#f0f4fa;border:1px solid #aab8cc;padding:5px 14px;font-size:14px;font-weight:700;color:#cc3333;max-width:340px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.msg{font-size:12px;color:#555;line-height:1.7;max-width:400px;margin:0 auto 22px}
.chips-row{display:flex;gap:8px;justify-content:center;flex-wrap:wrap}
.info-chip{background:#f4f7fb;border:1px solid #b0bdd0;padding:4px 12px;font-size:11px;color:#144985;font-weight:600;border-radius:2px}
.ftr-bar{background:linear-gradient(180deg,#1a5ca8 0%%,#0d3260 100%%);padding:8px 18px;display:flex;justify-content:flex-end;align-items:center}
.brand{font-size:14px;font-weight:900;color:#fff;letter-spacing:.3px}
.brand-star{color:#f0a500;margin:0 3px}
.brand-ver{font-size:9px;font-weight:600;color:#8bb8e8;margin-left:2px;vertical-align:super}
.ftr-copy{background:#e4e0d8;text-align:center;padding:4px 8px;font-size:9px;color:#777;border-top:1px solid #ccc}
</style>
</head>
<body>
<div class="frame">
  <div class="hdr">
    <div class="hdr-logo">
      <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke-linecap="round" stroke-linejoin="round">
        <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" fill="rgba(240,165,0,.35)" stroke="#f0a500" stroke-width="1.8"/>
        <polyline points="9 12 11 14 15 10" stroke="white" stroke-width="2.2"/>
      </svg>
    </div>
    <div class="hdr-title">K10 Web Protection Administration</div>
  </div>
  <div class="body">
    <div class="icon-wrap">
      <svg width="76" height="76" viewBox="0 0 24 24" fill="none" stroke-linecap="round" stroke-linejoin="round">
        <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" fill="#fde8e8" stroke="#cc3333" stroke-width="1.4"/>
        <line x1="15" y1="9" x2="9" y2="15" stroke="#cc3333" stroke-width="2.2"/>
        <line x1="9" y1="9" x2="15" y2="15" stroke="#cc3333" stroke-width="2.2"/>
      </svg>
    </div>
    <div class="chip">Access Blocked</div>
    <h1>This website has been blocked</h1>
    <div class="domain-row">
      <span class="domain-label">Site:</span>
      <div class="domain-val">%s</div>
    </div>
    <p class="msg">This website has been blocked by K10 Web Protection because it may contain adult content, malware, phishing attempts, or other material that violates your configured filtering policy.</p>
    <div class="chips-row">
      <div class="info-chip">Filtered by K10 Web Protection</div>
      <div class="info-chip">Contact your administrator to request access</div>
    </div>
  </div>
  <div class="ftr-bar">
    <div class="brand">K10<span class="brand-star">&#9733;</span>WebProtection<span class="brand-ver">PRO</span></div>
  </div>
  <div class="ftr-copy">Copyright &copy; 2024&ndash;2026 K10WebProtection &mdash; All Rights Reserved.</div>
</div>
</body>
</html>`

func blockPageHTML(domain string) string {
	return fmt.Sprintf(blockPageTpl, html.EscapeString(domain))
}

type OnBlockFn func(domain string)

type Proxy struct {
	cfg     *config.Config
	db      *database.Database
	server  *http.Server
	onBlock OnBlockFn
}

func New(cfg *config.Config, onBlock OnBlockFn) *Proxy {
	initMITM()
	return &Proxy{cfg: cfg, db: database.DB, onBlock: onBlock}
}

func (p *Proxy) Start() error {
	p.server = &http.Server{
		Addr: fmt.Sprintf("127.0.0.1:%d", p.cfg.ProxyPort),
		Handler: http.HandlerFunc(p.handle),
		// Do NOT set ReadTimeout/WriteTimeout — they kill long-lived HTTPS tunnels.
		// Only limit the time to read the initial request headers.
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
	// Recover from any panic so one bad request can't crash the proxy
	defer func() {
		if rec := recover(); rec != nil {
			http.Error(w, "Internal proxy error", http.StatusInternalServerError)
		}
	}()

	host := hostname(r.Host)

	// Built-in critical services (OS updates, safe-browsing, OCSP) — never block
	if IsBuiltinAllowed(host) {
		p.passThrough(w, r)
		return
	}

	// User allow-list wins
	if p.cfg.IsAllowed(host) {
		p.passThrough(w, r)
		return
	}

	// User's custom block list
	if p.cfg.UserBlocks(host) {
		p.block(w, r, host)
		return
	}

	// ── Bypass prevention (always active) ────────────────────────────────────

	rawURL := r.URL.String()

	if isWebProxy(host, rawURL) {
		p.block(w, r, host)
		return
	}

	// Detect base64-encoded redirect destinations (?__cpo=, ?url=, etc.).
	// Catches e.g. azureserv.com/?__cpo=aHR0cHM6Ly93d3cueG54eC5jb20 → xnxx.com.
	if decodedHost := DecodeRedirectHost(rawURL); decodedHost != "" {
		if !IsBuiltinAllowed(decodedHost) && !p.cfg.IsAllowed(decodedHost) {
			if p.db.BlocksDomainInCategories(decodedHost, LevelCategories[p.cfg.FilterLevel]) {
				p.block(w, r, host)
				return
			}
		}
	}

	// ── Focus mode — block social media sites ────────────────────────────────

	if p.cfg.FocusBlocks(host) {
		p.block(w, r, host)
		return
	}

	// ── Level-based filtering ─────────────────────────────────────────────────

	level := p.cfg.FilterLevel

	if level == LevelMonitor {
		// Monitor: no blocking, pass through
	} else {
		cats := LevelCategories[level] // nil for custom/unknown → blocks nothing via DB

		// Direct-IP bypass prevention (all levels except monitor/custom)
		if level != LevelCustom && level != "" && isDirectIP(host) {
			p.block(w, r, host)
			return
		}

		// YouTube (high level only, or custom toggle)
		if (level == LevelHigh || (level == LevelCustom || level == "") && p.cfg.BlockYouTube) && isYouTube(host) {
			p.block(w, r, host)
			return
		}

		// Image search (high level only, or custom toggle)
		if (level == LevelHigh || (level == LevelCustom || level == "") && p.cfg.BlockImageSearch) && isImageSearch(host, rawURL) {
			p.block(w, r, host)
			return
		}

		// Database domain lookup (category-aware for named levels, all-category for custom)
		if len(cats) > 0 || level == LevelCustom || level == "" {
			dbCats := cats
			if (level == LevelCustom || level == "") && p.cfg.BlockAdultContent {
				dbCats = nil // nil = all categories
			}
			if p.db.BlocksDomainInCategories(host, dbCats) {
				p.block(w, r, host)
				return
			}
			// URL-level checks (HTTP only — CONNECT is an opaque tunnel)
			if r.Method != http.MethodConnect {
				// Full-URL substring patterns (.xxx TLD, /porn/ path, ?q=porn query)
				if p.db.BlocksURLInCategories(rawURL, dbCats) {
					p.block(w, r, host)
					return
				}
				// Keyword wildcards checked against path+query only (host covered above)
				urlPath := r.URL.Path
				if r.URL.RawQuery != "" {
					urlPath += "?" + r.URL.RawQuery
				}
				if p.db.BlocksURLPatternInCategories(urlPath, dbCats) {
					p.block(w, r, host)
					return
				}
				// Page-content keyword phrases
				if p.db.BlocksKeyword(rawURL) {
					p.block(w, r, host)
					return
				}
			}
		}
	}

	// User keyword matching (always active).
	// For HTTPS (CONNECT) we only have the hostname; for HTTP we have the full URL.
	{
		kwTarget := rawURL
		if r.Method == http.MethodConnect {
			kwTarget = host
		}
		if p.cfg.UserKeywordMatch(kwTarget) {
			p.block(w, r, host)
			return
		}
	}

	// SafeSearch MITM
	if r.Method == http.MethodConnect && p.cfg.SafeSearch && safeSearchDomains[host] {
		p.safeSearchIntercept(w, r, host)
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

// block sends the K10 block page (or a plain 403 for CONNECT).
func (p *Proxy) block(w http.ResponseWriter, r *http.Request, domain string) {
	if p.onBlock != nil {
		go p.onBlock(domain) // async so it never delays the response
	}
	if r.Method == http.MethodConnect {
		p.blockTunnel(w, domain)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte(blockPageHTML(domain)))
}

// tunnel handles HTTPS CONNECT — raw TCP pass-through (no TLS inspection).
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

	// Confirm tunnel to the browser
	conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	// Pipe both directions — wait for BOTH to finish before closing
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
	<-done // wait for BOTH goroutines before closing connections
	conn.Close()
	dest.Close()
}

// forward proxies plain HTTP requests.
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
