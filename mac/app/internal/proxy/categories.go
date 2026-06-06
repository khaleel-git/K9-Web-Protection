package proxy

import (
	"encoding/base64"
	"net"
	"net/url"
	"strings"
)

// ── Filter level constants ────────────────────────────────────────────────────

const (
	LevelHigh     = "high"
	LevelDefault  = "default"
	LevelModerate = "moderate"
	LevelMinimal  = "minimal"
	LevelMonitor  = "monitor"
	LevelCustom   = "custom"
)

// LevelCategories maps each filter level to the database category names it blocks.
// Must match lists/levels/*.json exactly so build.py and the proxy agree.
var LevelCategories = map[string][]string{
	LevelHigh: {
		"pornography", "adult-mature", "alternative-sexuality",
		"alternative-spirituality", "abortion", "alcohol", "chat-im",
		"extreme", "gambling", "hacking", "illegal-drugs", "intimate-apparel",
		"lgbt", "malware-spyware", "newsgroups-forums", "nudity",
		"open-image-search", "p2p", "personal-pages", "personals-dating",
		"phishing", "proxy-avoidance", "sex-education", "social-networking",
		"suspicious", "tobacco", "unrated", "violence-hate", "weapons",
	},
	LevelDefault: {
		"pornography", "adult-mature", "alternative-sexuality", "extreme",
		"gambling", "hacking", "illegal-drugs", "intimate-apparel", "nudity",
		"personals-dating", "phishing", "malware-spyware", "proxy-avoidance",
		"sex-education", "abortion", "suspicious", "violence-hate",
	},
	LevelModerate: {
		"pornography", "adult-mature", "gambling", "hacking", "illegal-drugs",
		"phishing", "malware-spyware", "extreme", "violence-hate", "suspicious",
	},
	LevelMinimal: {
		"pornography", "phishing", "malware-spyware",
	},
	LevelMonitor: {},
}

// ── Proxy bypass / web proxy detection (always active) ───────────────────────

// knownProxyDomains — explicit list of web proxy / anonymiser services.
var knownProxyDomains = []string{
	// Web proxies
	"proxyium.com", "proxypal.net", "plainproxies.com",
	"croxyproxy.com", "croxyproxy.net", "croxyproxy.rocks",
	"kproxy.com", "hide.me", "hidester.com", "hidemyname.org",
	"4everproxy.com", "proxfree.com", "filterbypass.me",
	"anonymouse.org", "proxysite.com", "unblocker.cc",
	"zend2.com", "anonymiz.com", "websurf.in", "spys.one",
	"proxy-n-vpn.com", "ninja-web.net", "1proxy.de", "hidemy.name",
	"whoer.net", "freeproxy.io", "free-proxy.cz",
	"unblockvideos.com", "bypassblocker.com",
	"ultrasurf.us", "browsec.com",
	"webproxy.to", "proxy-online.de", "goproxy.win",
	"freeproxy.win", "proxyunblocker.org", "proxysites.io",
	"cactus.tools", "blockaway.net", "unblocksite.net",
	"proxysite.cc", "ssl.unblocksit.es", "unblocksit.es",
	// Known CDN/redirect-as-bypass domains (base64 redirect trick)
	"azureserv.com",
}

// knownBypassParams — URL query parameter names used to pass an encoded
// destination URL through a redirect/bypass service.
var knownBypassParams = []string{
	"__cpo", "cpo", "url", "site", "u", "go", "proxy",
	"redirect", "r", "dest", "destination", "target", "q",
}

// proxyURLParams — plain-text URL-in-URL indicators (fast path).
var proxyURLParams = []string{
	"?url=http", "&url=http", "?site=http", "&site=http",
	"?u=http", "&u=http", "?go=http", "?proxy=http",
	"?redirect=http", "&redirect=http", "?dest=http",
}

func isWebProxy(host, rawURL string) bool {
	h := strings.ToLower(host)
	for _, d := range knownProxyDomains {
		if h == d || strings.HasSuffix(h, "."+d) {
			return true
		}
	}
	// Keyword-in-label check, limited to shallow domains (≤2 dots) so that deep
	// corporate subdomains like proxy.individual.githubcopilot.com are never flagged.
	if strings.Count(h, ".") <= 2 {
		for _, part := range strings.Split(h, ".") {
			for _, kw := range []string{
				"proxy", "unblock", "bypass", "anonymize",
				"hideip", "hidemy", "anonym", "webproxy",
			} {
				if strings.Contains(part, kw) { // Contains, not ==
					return true
				}
			}
		}
	}
	lower := strings.ToLower(rawURL)
	for _, p := range proxyURLParams {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

// DecodeRedirectHost extracts and returns the hostname from any base64-encoded
// redirect URL hidden in the request's query parameters.
// Returns "" if no encoded redirect is found.
//
// Covers the Azure CDN "__cpo" trick and similar redirect-as-bypass patterns:
//
//	azureserv.com/?__cpo=aHR0cHM6Ly93d3cueG54eC5jb20  →  www.xnxx.com
func DecodeRedirectHost(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	q := parsed.Query()
	for _, param := range knownBypassParams {
		vals, ok := q[param]
		if !ok {
			continue
		}
		for _, val := range vals {
			if val == "" {
				continue
			}
			// Try all base64 variants (standard, URL-safe, without padding)
			for _, dec := range []func(string) ([]byte, error){
				base64.StdEncoding.DecodeString,
				base64.URLEncoding.DecodeString,
				base64.RawStdEncoding.DecodeString,
				base64.RawURLEncoding.DecodeString,
			} {
				b, err := dec(val)
				if err != nil {
					continue
				}
				s := strings.TrimSpace(string(b))
				if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
					u2, err := url.Parse(s)
					if err == nil && u2.Host != "" {
						return strings.ToLower(u2.Hostname())
					}
				}
			}
		}
	}
	return ""
}

// ── Communication apps allow list ────────────────────────────────────────────
// These apps must stay accessible even when their category (chat-im,
// social-networking) is active at the current filter level.
// User's custom block list and Focus Mode still override this — only
// the DB category lookup is bypassed, not user-defined rules.
//
// Trade-off: facebook.com is included because Messenger authenticates
// through it. This means Facebook's news feed is also unblocked at any level
// when Messenger access is needed. If that is undesirable, remove facebook.com
// and direct users to add messenger.com to their manual allow list instead.
var communicationAllowDomains = []string{
	// WhatsApp
	"whatsapp.com", "whatsapp.net", "wa.me",
	// Messenger (facebook.com required for login/auth flow)
	"messenger.com", "facebook.com",
	// Instagram
	"instagram.com", "cdninstagram.com",
	// Shared Facebook media CDN (Messenger + Instagram images/video)
	"fbcdn.net",
	// Apple iMessage / Messages — already covered by builtinAllowDomains (apple.com, icloud.com)
}

func IsCommunicationAllowed(host string) bool {
	h := strings.ToLower(host)
	for _, d := range communicationAllowDomains {
		if h == d || strings.HasSuffix(h, "."+d) {
			return true
		}
	}
	return false
}

// ── Built-in never-block list ────────────────────────────────────────────────
// Critical OS/platform services that must never be blocked regardless of level.

var builtinAllowDomains = []string{
	// Apple OS services
	"apple.com", "icloud.com", "mzstatic.com", "itunes.com",
	"push.apple.com", "gateway.icloud.com", "safebrowsing.apple",
	// Google safe browsing & OS services
	"safebrowsing.googleapis.com", "safebrowsing.google.com",
	"accounts.google.com", "googleapis.com",
	// Microsoft / Windows Update
	"microsoft.com", "windowsupdate.com", "live.com",
	// Certificate validation (OCSP/CRL)
	"ocsp.apple.com", "crl.apple.com", "ocsp.digicert.com",
	"ocsp.comodoca.com", "crl.sectigo.com",
}

func IsBuiltinAllowed(host string) bool {
	h := strings.ToLower(host)
	for _, d := range builtinAllowDomains {
		if h == d || strings.HasSuffix(h, "."+d) {
			return true
		}
	}
	return false
}

// ── Other always-on helpers ───────────────────────────────────────────────────

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

func isDirectIP(host string) bool {
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast()
}
