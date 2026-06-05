// Package database loads the built-in blocklists that ship with K9.
// These are read-only — user additions live in config.Config.
package database

import (
	"embed"
	"encoding/json"
	"strings"
)

//go:embed domains.json urls.json multi-words.json
var fs embed.FS

// DB is the global built-in database, loaded once at startup.
var DB *Database

func init() {
	DB = load()
}

// Database holds the three built-in lists.
// domains is a hash set for O(1) exact + subdomain lookup.
// urls and keywords stay as slices (substring search, no map shortcut).
type Database struct {
	domainSet map[string]struct{} // exact hostnames, lowercased
	domains   []string            // same entries as domainSet, kept for DomainCount
	urls      []string            // URL fragments   e.g. "reddit.com/r/nsfw"
	keywords  []string            // keyword phrases e.g. "anal porn"
}

func (d *Database) DomainCount()  int { return len(d.domains) }
func (d *Database) URLCount()     int { return len(d.urls) }
func (d *Database) KeywordCount() int { return len(d.keywords) }

// BlocksDomain returns true if host exactly matches a blocked domain
// or is a subdomain of one (e.g. "www.pornpics.de" matches "pornpics.de").
func (d *Database) BlocksDomain(host string) bool {
	host = strings.ToLower(host)
	// Exact match
	if _, ok := d.domainSet[host]; ok {
		return true
	}
	// Subdomain match: strip labels one at a time
	for {
		dot := strings.IndexByte(host, '.')
		if dot < 0 {
			return false
		}
		host = host[dot+1:]
		if _, ok := d.domainSet[host]; ok {
			return true
		}
	}
}

// BlocksURL returns true if the full URL contains any built-in URL pattern.
func (d *Database) BlocksURL(rawURL string) bool {
	u := strings.ToLower(rawURL)
	for _, p := range d.urls {
		if strings.Contains(u, p) {
			return true
		}
	}
	return false
}

// BlocksKeyword returns true if the URL contains any built-in keyword.
func (d *Database) BlocksKeyword(rawURL string) bool {
	u := strings.ToLower(rawURL)
	for _, kw := range d.keywords {
		if strings.Contains(u, kw) {
			return true
		}
	}
	return false
}

// ── loader ────────────────────────────────────────────────────────────────────

func load() *Database {
	rawDomains := loadList("domains.json")
	domainSet := make(map[string]struct{}, len(rawDomains))
	for _, d := range rawDomains {
		domainSet[d] = struct{}{}
	}
	return &Database{
		domainSet: domainSet,
		domains:   rawDomains,
		urls:      loadList("urls.json"),
		keywords:  loadList("multi-words.json"),
	}
}

// loadList parses a JSON file of the form {"key": ["item", ...], ...}
// and returns a deduplicated, lowercased slice of all values.
func loadList(name string) []string {
	data, err := fs.ReadFile(name)
	if err != nil {
		return nil
	}
	var raw map[string][]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}
	seen := make(map[string]struct{}, len(raw)*64)
	var out []string
	for _, items := range raw {
		for _, item := range items {
			str, ok := item.(string)
			if !ok {
				continue
			}
			s := strings.ToLower(strings.TrimSpace(str))
			for _, pfx := range []string{"https://", "http://", "www."} {
				s = strings.TrimPrefix(s, pfx)
			}
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			if _, dup := seen[s]; !dup {
				seen[s] = struct{}{}
				out = append(out, s)
			}
		}
	}
	return out
}
