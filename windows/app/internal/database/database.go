// Package database loads the built-in blocklists that ship with K10.
//
//   domains.json      — category-keyed: {"pornography": ["bad.com",...], ...}
//   urls.json         — category-keyed: {"pornography": [".xxx", "/porn/",...], ...}
//                       Matched as substring against the full raw URL (host+path+query).
//   url-patterns.json — category-keyed: {"pornography": ["porn","xxx",...], ...}
//                       Matched as substring against URL path+query only (not host).
//                       Category-aware: only fires for categories active at the filter level.
//   multi-words.json  — flat alpha-keyed keyword phrases for page-content matching.
package database

import (
	"embed"
	"encoding/json"
	"strings"
)

//go:embed domains.json urls.json url-patterns.json multi-words.json
var fs embed.FS

// DB is the global built-in database, loaded once at startup.
var DB *Database

type Database struct {
	catDomains     map[string]map[string]struct{} // category → hostname set
	totalDomains   int
	catUrls        map[string][]string // category → full-URL substring patterns
	catUrlPatterns map[string][]string // category → path-keyword wildcards
	keywords       []string            // page content keyword phrases
}

func (d *Database) DomainCount() int { return d.totalDomains }

// URLCount returns the total number of URL patterns across all categories
// (both full-URL patterns from urls.json and keyword patterns from url-patterns.json).
func (d *Database) URLCount() int {
	n := 0
	for _, v := range d.catUrls {
		n += len(v)
	}
	for _, v := range d.catUrlPatterns {
		n += len(v)
	}
	return n
}

func (d *Database) KeywordCount() int { return len(d.keywords) }

// BlocksDomainInCategories returns true if host (or any parent domain) is in
// any of the given categories. Pass nil to check all categories.
func (d *Database) BlocksDomainInCategories(host string, cats []string) bool {
	host = strings.ToLower(host)
	if len(cats) == 0 {
		for _, set := range d.catDomains {
			if matchInSet(host, set) {
				return true
			}
		}
		return false
	}
	for _, cat := range cats {
		if set, ok := d.catDomains[cat]; ok {
			if matchInSet(host, set) {
				return true
			}
		}
	}
	return false
}

// BlocksDomain checks all categories (legacy / custom level).
func (d *Database) BlocksDomain(host string) bool {
	return d.BlocksDomainInCategories(host, nil)
}

// BlocksURLInCategories returns true if the full raw URL (scheme+host+path+query)
// contains any url pattern from the given categories.
// Use this for TLD patterns (.xxx), path patterns (/porn/), and query patterns (?q=porn).
// Pass nil cats to check all categories.
func (d *Database) BlocksURLInCategories(rawURL string, cats []string) bool {
	u := strings.ToLower(rawURL)
	if len(cats) == 0 {
		for _, patterns := range d.catUrls {
			for _, p := range patterns {
				if strings.Contains(u, p) {
					return true
				}
			}
		}
		return false
	}
	for _, cat := range cats {
		for _, p := range d.catUrls[cat] {
			if strings.Contains(u, p) {
				return true
			}
		}
	}
	return false
}

// BlocksURLPatternInCategories returns true if the URL path+query string contains
// any keyword wildcard pattern from the given categories.
// urlPath should be r.URL.Path + "?" + r.URL.RawQuery (host is excluded to prevent
// false positives — the host is already covered by BlocksDomainInCategories).
// Pass nil cats to check all categories.
func (d *Database) BlocksURLPatternInCategories(urlPath string, cats []string) bool {
	path := strings.ToLower(urlPath)
	if len(cats) == 0 {
		for _, patterns := range d.catUrlPatterns {
			for _, kw := range patterns {
				if strings.Contains(path, kw) {
					return true
				}
			}
		}
		return false
	}
	for _, cat := range cats {
		for _, kw := range d.catUrlPatterns[cat] {
			if strings.Contains(path, kw) {
				return true
			}
		}
	}
	return false
}

// BlocksKeyword returns true if the URL contains any page-content keyword phrase.
func (d *Database) BlocksKeyword(rawURL string) bool {
	u := strings.ToLower(rawURL)
	for _, kw := range d.keywords {
		if strings.Contains(u, kw) {
			return true
		}
	}
	return false
}

// categoryPriority defines a deterministic lookup order for CategoryFor so that
// domains appearing in multiple categories always resolve to the same one.
// Security threats take precedence over content categories.
var categoryPriority = []string{
	"malware-spyware", "phishing", "suspicious",
	"pornography", "extreme", "violence-hate",
	"illegal-drugs", "hacking", "proxy-avoidance",
	"adult-mature", "alternative-sexuality", "nudity",
	"gambling", "personals-dating", "intimate-apparel",
	"sex-education", "chat-im", "social-networking",
	"alcohol", "tobacco", "weapons", "p2p",
	"alternative-spirituality", "newsgroups-forums",
	"abortion", "open-image-search", "personal-pages",
	"lgbt", "unrated",
}

// CategoryFor returns the highest-priority database category that contains host
// (or a parent domain), e.g. "pornography", "social-networking". Returns "" if
// not found. Result is deterministic — categoryPriority defines the order.
func (d *Database) CategoryFor(host string) string {
	host = strings.ToLower(host)
	for _, cat := range categoryPriority {
		if set, ok := d.catDomains[cat]; ok {
			if matchInSet(host, set) {
				return cat
			}
		}
	}
	// Fall through to any category not listed in categoryPriority
	for cat, set := range d.catDomains {
		if matchInSet(host, set) {
			return cat
		}
	}
	return ""
}

// ── helpers ───────────────────────────────────────────────────────────────────

func matchInSet(host string, set map[string]struct{}) bool {
	if _, ok := set[host]; ok {
		return true
	}
	for {
		dot := strings.IndexByte(host, '.')
		if dot < 0 {
			return false
		}
		host = host[dot+1:]
		if _, ok := set[host]; ok {
			return true
		}
	}
}

// ── loader ────────────────────────────────────────────────────────────────────

func init() { DB = load() }

func load() *Database {
	// domains.json — category-keyed
	catDomains, totalDomains := loadCategoryDomains()

	// urls.json — category-keyed URL substring patterns
	catUrls := loadCategoryStringSlices("urls.json")

	// url-patterns.json — category-keyed keyword wildcards
	catUrlPatterns := loadCategoryStringSlices("url-patterns.json")

	// multi-words.json — flat alpha-keyed keyword phrases
	keywords := loadFlatKeywords("multi-words.json")

	return &Database{
		catDomains:     catDomains,
		totalDomains:   totalDomains,
		catUrls:        catUrls,
		catUrlPatterns: catUrlPatterns,
		keywords:       keywords,
	}
}

func loadCategoryDomains() (map[string]map[string]struct{}, int) {
	data, err := fs.ReadFile("domains.json")
	if err != nil {
		return map[string]map[string]struct{}{}, 0
	}
	var raw map[string][]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return loadLegacyDomains(data)
	}
	catDomains := make(map[string]map[string]struct{}, len(raw))
	total := 0
	for cat, domains := range raw {
		set := make(map[string]struct{}, len(domains))
		for _, d := range domains {
			d = strings.ToLower(strings.TrimSpace(d))
			if d != "" {
				set[d] = struct{}{}
			}
		}
		catDomains[cat] = set
		total += len(set)
	}
	return catDomains, total
}

func loadLegacyDomains(data []byte) (map[string]map[string]struct{}, int) {
	var raw map[string][]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return map[string]map[string]struct{}{}, 0
	}
	set := make(map[string]struct{})
	for _, items := range raw {
		for _, item := range items {
			if s, ok := item.(string); ok {
				s = strings.ToLower(strings.TrimSpace(s))
				if s != "" {
					set[s] = struct{}{}
				}
			}
		}
	}
	return map[string]map[string]struct{}{"legacy": set}, len(set)
}

// loadCategoryStringSlices loads a category-keyed JSON file of string slices.
// Returns empty map (not nil) on error so callers can safely range over it.
func loadCategoryStringSlices(name string) map[string][]string {
	data, err := fs.ReadFile(name)
	if err != nil {
		return map[string][]string{}
	}
	var raw map[string][]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return map[string][]string{}
	}
	// Normalise: lowercase, strip scheme/www from URL patterns
	result := make(map[string][]string, len(raw))
	for cat, items := range raw {
		seen := make(map[string]struct{}, len(items))
		out := make([]string, 0, len(items))
		for _, s := range items {
			s = strings.ToLower(strings.TrimSpace(s))
			if s == "" {
				continue
			}
			// Strip leading scheme (only for full-URL patterns, not path patterns)
			for _, pfx := range []string{"https://", "http://"} {
				s = strings.TrimPrefix(s, pfx)
			}
			if _, dup := seen[s]; !dup {
				seen[s] = struct{}{}
				out = append(out, s)
			}
		}
		if len(out) > 0 {
			result[cat] = out
		}
	}
	return result
}

// loadFlatKeywords loads the alpha-keyed multi-words.json into a flat slice.
func loadFlatKeywords(name string) []string {
	data, err := fs.ReadFile(name)
	if err != nil {
		return nil
	}
	var raw map[string][]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}
	seen := make(map[string]struct{})
	var out []string
	for _, items := range raw {
		for _, s := range items {
			s = strings.ToLower(strings.TrimSpace(s))
			if s != "" {
				if _, dup := seen[s]; !dup {
					seen[s] = struct{}{}
					out = append(out, s)
				}
			}
		}
	}
	return out
}
